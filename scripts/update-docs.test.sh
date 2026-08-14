#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/update-docs"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/candidate-docs-update-tests.XXXXXX")"

cleanup() {
  if [[ -d "$TEST_ROOT" && "$TEST_ROOT" == *candidate-docs-update-tests.* ]]; then
    rm -rf -- "$TEST_ROOT"
  fi
}
trap cleanup EXIT

fail_test() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local expected="$2"
  rg -F --quiet "$expected" "$file" || fail_test "$file não contém: $expected"
}

assert_not_exists() {
  local path="$1"
  [[ ! -e "$path" ]] || fail_test "não deveria existir: $path"
}

create_fixture() {
  local name="$1"
  local fixture="$TEST_ROOT/$name"

  mkdir -p "$fixture/bin" "$fixture/docs/prompts" "$fixture/docs/architecture"
  printf '# prompt de teste\n' >"$fixture/docs/prompts/documentation-updater.md"
  printf '# visão inicial\n' >"$fixture/docs/architecture/system-overview.md"
  {
    printf 'version: 1\n'
    printf 'last_documented:\n'
    printf '  change_id: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n'
    printf '  commit_id: 1111111111111111111111111111111111111111\n'
    printf '  recorded_at: "2026-08-14T00:00:00Z"\n'
  } >"$fixture/docs/documentation-state.yaml"

  apply_fake_jj "$fixture/bin/jj"
  apply_fake_codex "$fixture/bin/codex"
  chmod +x "$fixture/bin/jj" "$fixture/bin/codex"
  printf '%s\n' "$fixture"
}

apply_fake_jj() {
  local destination="$1"
  cat >"$destination" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >>"$FAKE_JJ_LOG"
command_name="$1"
shift
arguments="$*"

case "$command_name" in
  log)
    if [[ "$arguments" == *'ancestors('* ]]; then
      printf '%s\n' "$FAKE_BASE_COMMIT"
    elif [[ "$arguments" == *'::'* ]]; then
      printf '%s\n' "$FAKE_COMMITS"
    elif [[ "$arguments" == *'empty ++'* ]]; then
      printf '%s\n' "$FAKE_CURRENT_EMPTY"
    elif [[ "$arguments" == *'change_id ++'* ]]; then
      if [[ "$arguments" == *'-r @ '* && ( -f "$FAKE_NEW_MARKER" || "$FAKE_CURRENT_IS_DOCS_CHANGE" == true ) ]]; then
        printf '%s\n' "$FAKE_DOC_CHANGE"
      elif [[ "$arguments" == *'-r @ '* && "$FAKE_CURRENT_EMPTY" == true ]]; then
        printf '%s\n' "$FAKE_DOC_CHANGE"
      else
        printf '%s\n' "$FAKE_TARGET_CHANGE"
      fi
    elif [[ "$arguments" == *'commit_id ++'* ]]; then
      printf '%s\n' "$FAKE_TARGET_COMMIT"
    else
      printf 'fake jj: log não reconhecido: %s\n' "$arguments" >&2
      exit 2
    fi
    ;;
  diff)
    if [[ "$arguments" == *'--from'* ]]; then
      printf '%b' "$FAKE_CHANGED_FILES"
    elif [[ -f "$FAKE_CODEX_CALLED" ]]; then
      printf '%b' "$FAKE_CODEX_OUTPUTS"
    else
      printf '%b' "$FAKE_CURRENT_OUTPUTS"
    fi
    ;;
  new)
    touch "$FAKE_NEW_MARKER"
    ;;
  describe)
    printf '%s\n' "$arguments" >"$FAKE_DESCRIBE_MARKER"
    ;;
  *)
    printf 'fake jj: comando não reconhecido: %s\n' "$command_name" >&2
    exit 2
    ;;
esac
EOF
}

apply_fake_codex() {
  local destination="$1"
  cat >"$destination" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >"$FAKE_CODEX_ARGS"
cat >"$FAKE_CODEX_PROMPT"
touch "$FAKE_CODEX_CALLED"
if [[ "${FAKE_CODEX_EXIT:-0}" != 0 ]]; then
  exit "$FAKE_CODEX_EXIT"
fi
printf '\nAtualizada pelo fake Codex.\n' >>"$DOCS_UPDATE_ROOT/docs/architecture/system-overview.md"
printf 'fake Codex concluído\n'
EOF
}

run_tool() {
  local fixture="$1"
  shift

  export DOCS_UPDATE_ROOT="$fixture"
  export DOCS_UPDATE_STATE="$fixture/docs/documentation-state.yaml"
  export DOCS_UPDATE_PROMPT="$fixture/docs/prompts/documentation-updater.md"
  export JJ_BIN="$fixture/bin/jj"
  export CODEX_BIN="$fixture/bin/codex"
  export FAKE_JJ_LOG="$fixture/jj.log"
  export FAKE_NEW_MARKER="$fixture/new.called"
  export FAKE_DESCRIBE_MARKER="$fixture/describe.called"
  export FAKE_CODEX_ARGS="$fixture/codex.args"
  export FAKE_CODEX_PROMPT="$fixture/codex.prompt"
  export FAKE_CODEX_CALLED="$fixture/codex.called"
  export FAKE_BASE_COMMIT='1111111111111111111111111111111111111111'
  export FAKE_TARGET_CHANGE='bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
  export FAKE_TARGET_COMMIT='2222222222222222222222222222222222222222'
  export FAKE_DOC_CHANGE='cccccccccccccccccccccccccccccccc'
  export FAKE_COMMITS='222222222222 bbbbbbbbbbbb feat: mudança de teste'
  export FAKE_CODEX_OUTPUTS="${FAKE_CODEX_OUTPUTS:-docs/architecture/system-overview.md\\n}"
  export FAKE_CURRENT_OUTPUTS="${FAKE_CURRENT_OUTPUTS:-back/example.go\\n}"
  export FAKE_CURRENT_IS_DOCS_CHANGE="${FAKE_CURRENT_IS_DOCS_CHANGE:-false}"
  export FAKE_CODEX_EXIT="${FAKE_CODEX_EXIT:-0}"

  "$TOOL" "$@"
}

test_dry_run_does_not_mutate() {
  local fixture
  fixture="$(create_fixture dry-run)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CHANGED_FILES='back/example.go\n'

  run_tool "$fixture" --dry-run >"$fixture/output"

  assert_contains "$fixture/output" 'Dry-run concluído'
  assert_contains "$fixture/output" 'back/example.go'
  assert_not_exists "$fixture/new.called"
  assert_not_exists "$fixture/codex.args"
  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
}

test_docs_only_range_is_ignored() {
  local fixture
  fixture="$(create_fixture docs-only)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CHANGED_FILES='docs/README.md\nREADME.md\n'

  run_tool "$fixture" >"$fixture/output"

  assert_contains "$fixture/output" 'Nenhuma mudança de código ou operação'
  assert_not_exists "$fixture/new.called"
  assert_not_exists "$fixture/codex.args"
}

test_success_updates_state_in_child_change() {
  local fixture
  fixture="$(create_fixture success)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CHANGED_FILES='back/example.go\n'

  run_tool "$fixture" >"$fixture/output"

  [[ -f "$fixture/new.called" ]] || fail_test 'jj new não foi chamado'
  [[ -f "$fixture/describe.called" ]] || fail_test 'jj describe não foi chamado'
  assert_contains "$fixture/codex.args" 'exec --ephemeral --sandbox workspace-write'
  assert_contains "$fixture/codex.prompt" '2222222222222222222222222222222222222222'
  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
  assert_contains "$fixture/docs/documentation-state.yaml" 'commit_id: 2222222222222222222222222222222222222222'
  assert_contains "$fixture/describe.called" 'docs: update through 222222222222'
  assert_contains "$fixture/docs/architecture/system-overview.md" 'Atualizada pelo fake Codex.'
}

test_empty_change_is_reused() {
  local fixture
  fixture="$(create_fixture empty-change)"
  export FAKE_CURRENT_EMPTY=true
  export FAKE_CHANGED_FILES='back/example.go\n'

  run_tool "$fixture" >"$fixture/output"

  assert_not_exists "$fixture/new.called"
  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
  assert_contains "$fixture/describe.called" 'docs: update through 222222222222'
}

test_codex_failure_does_not_advance_state() {
  local fixture
  fixture="$(create_fixture failure)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CHANGED_FILES='back/example.go\n'
  export FAKE_CODEX_EXIT=7

  if run_tool "$fixture" >"$fixture/output" 2>&1; then
    fail_test 'falha do Codex deveria propagar exit code'
  fi

  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  assert_not_exists "$fixture/describe.called"
}

test_retry_reuses_documentation_change() {
  local fixture
  fixture="$(create_fixture retry)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CURRENT_OUTPUTS='docs/architecture/system-overview.md\n'
  export FAKE_CURRENT_IS_DOCS_CHANGE=true
  export FAKE_CHANGED_FILES='back/example.go\n'
  export FAKE_CODEX_EXIT=0

  run_tool "$fixture" >"$fixture/output"

  assert_not_exists "$fixture/new.called"
  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
  assert_contains "$fixture/describe.called" 'docs: update through 222222222222'
}

test_disallowed_codex_output_blocks_marker() {
  local fixture
  fixture="$(create_fixture disallowed-output)"
  export FAKE_CURRENT_EMPTY=false
  export FAKE_CHANGED_FILES='back/example.go\n'
  export FAKE_CODEX_OUTPUTS='back/example.go\n'

  if run_tool "$fixture" >"$fixture/output" 2>&1; then
    fail_test 'arquivo de código alterado deveria bloquear o marcador'
  fi

  assert_contains "$fixture/docs/documentation-state.yaml" 'change_id: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  assert_not_exists "$fixture/describe.called"
  assert_contains "$fixture/output" 'Arquivos fora do escopo documental'
}

( test_dry_run_does_not_mutate )
( test_docs_only_range_is_ignored )
( test_success_updates_state_in_child_change )
( test_empty_change_is_reused )
( test_codex_failure_does_not_advance_state )
( test_retry_reuses_documentation_change )
( test_disallowed_codex_output_blocks_marker )

printf 'PASS: scripts/update-docs.test.sh\n'

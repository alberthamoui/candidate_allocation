#!/usr/bin/env bash
set -euo pipefail
task_root="$(cd "$(dirname "$0")/../.." && pwd)"
task_output="${CANDIDATE_NATIVE_OUTPUT:-$(mktemp -d /tmp/candidate-native-XXXXXX)}"
mkdir -p "$task_output"
task_modules="$(go env GOMODCACHE)"
task_cache="$(go env GOCACHE)"
mkdir -p "$task_cache"
docker build --network host --build-arg "VALIDATION_UID=$(id -u)" --build-arg "VALIDATION_GID=$(id -g)" -t candidate-native-validation:go1.25.9 \
  -f "$task_root/tools/native-validation/Dockerfile" "$task_root/tools/native-validation"
docker run --rm --network host --user "$(id -u):$(id -g)" \
  --mount "type=bind,src=$task_root,dst=/src,readonly" \
  --mount "type=bind,src=$task_output,dst=/out" \
  --mount "type=bind,src=$task_modules,dst=/go/pkg/mod" \
  --mount "type=bind,src=$task_cache,dst=/cache" \
  -e GOCACHE=/cache -e GOTOOLCHAIN=local \
  candidate-native-validation:go1.25.9 bash /src/tools/native-validation/smoke.sh
printf 'Artefatos nativos e screenshots: %s\n' "$task_output"

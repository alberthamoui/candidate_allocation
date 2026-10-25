#!/usr/bin/env bash
set -euo pipefail
cd /src
go build -tags production,webkit2_41 -o /out/candidate-release .
go build -tags production,debug,devtools,webkit2_41 -o /out/candidate-debug .
if /out/candidate-release debug performance > /out/release-debug-check.log 2>&1; then
  echo 'A versão release expôs diagnósticos.' >&2
  exit 1
fi
grep -q 'diagnósticos indisponíveis' /out/release-debug-check.log
mkdir -p /out/config /out/runtime /out/data
chmod 700 /out/runtime
export XDG_CONFIG_HOME=/out/config XDG_RUNTIME_DIR=/out/runtime
export CANDIDATE_ALLOCATOR_DATA_DIR=/out/data CANDIDATE_ALLOCATOR_DATABASE=/out/data/import.db
export CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE=/out/startup.ok
export GSETTINGS_BACKEND=memory LIBGL_ALWAYS_SOFTWARE=1
# Chromium/WebKit sandboxing inside an unprivileged test container cannot create
# another user namespace. This applies only to this isolated synthetic smoke.
export WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1
xvfb-run -a -s '-screen 0 1280x900x24' dbus-run-session -- bash -euc '
 /out/candidate-debug > /out/native.log 2>&1 &
 task_app=$!
 trap "kill $task_app 2>/dev/null || true" EXIT
 task_window=""
 for attempt in $(seq 1 100); do
  kill -0 "$task_app"
  task_window=$(xdotool search --name wails-events 2>/dev/null | head -1 || true)
  if [ -s /out/startup.ok ] && [ -n "$task_window" ]; then break; fi
  sleep 0.2
 done
 test -s /out/startup.ok
 test -n "$task_window"
 # Allow WebKit to finish loading the bundled frontend before capture.
 sleep 3
 scrot --overwrite /out/native-desktop.png
 # Exercise the native file chooser and Wails bridge with the repository fixture.
 xdotool mousemove --window "$task_window" 380 440 click 1
 sleep 1
 printf %s /src/Execelteste/Base4Restricao.xlsx | xclip -selection clipboard
 xdotool key --clearmodifiers ctrl+l
 xdotool key --clearmodifiers ctrl+a
 xdotool key --clearmodifiers ctrl+v
 sleep 1
 xdotool key Return
 sleep 1
 xdotool key --clearmodifiers alt+o
 sleep 1
 xdotool mousemove --window "$task_window" 512 530 click 1
 sleep 3
 scrot --overwrite /out/native-import.png
 xdotool windowsize "$task_window" 430 800
 sleep 1
 scrot --overwrite /out/native-narrow.png
 kill -0 "$task_app"
 kill -TERM "$task_app"
 wait "$task_app" || test "$?" -eq 143
 trap - EXIT
'
printf 'Builds release/debug, bloqueio de diagnósticos no release e startup GTK/WebKit concluídos.\n'

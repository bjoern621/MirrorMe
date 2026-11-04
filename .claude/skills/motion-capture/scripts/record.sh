#!/usr/bin/env bash
# Records every monitor to <out>/<monitor>.mp4 after a lead.
# Usage: record.sh <out> [seconds] [lead]
set -eu

out=$1
seconds=${2:-15}
lead=${3:-6}

hypr() { env -u LD_LIBRARY_PATH hyprctl "$@"; }
tell() { hypr notify 1 2000 "rgb(4caf50)" "$1" >/dev/null; }

mkdir -p "$out"
recorder="$(nix build nixpkgs#wf-recorder --print-out-paths --no-link)/bin/wf-recorder"
mapfile -t monitors < <(hypr monitors | awk '/^Monitor /{print $2}')

sleep "$lead"
tell "Recording $seconds s"
for monitor in "${monitors[@]}"; do
  # SIGINT lets wf-recorder finish the file. A monitor that stops changing ends short.
  timeout -s INT "$seconds" "$recorder" -y -o "$monitor" -c libx264 -p preset=ultrafast -p crf=18 -r 60 \
    -f "$out/$monitor.mp4" >"$out/$monitor.log" 2>&1 &
done
wait || true
tell "Recording done"

ls -la "$out"/*.mp4

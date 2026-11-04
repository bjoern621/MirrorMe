#!/usr/bin/env bash
# Lists a recording's scene changes and writes <video>-sheet.png, one frame a second.
# Usage: moments.sh <video.mp4> [threshold]
set -eu

video=$1
threshold=${2:-0.003}
sheet="${video%.mp4}-sheet.png"

seconds=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$video")
rows=$(awk -v s="$seconds" 'BEGIN { print int((s + 4.999) / 4) }')

ffmpeg -nostats -i "$video" -vf "select='gt(scene,$threshold)',metadata=print:file=-" -an -f null - 2>/dev/null \
  | { rg -o 'pts_time:[0-9.]+|scene_score=[0-9.]+' || true; } \
  | paste - - | sed 's/pts_time://; s/lavfi.scene_score=//'

ffmpeg -nostats -y -i "$video" -vf "fps=1,scale=640:-1,tile=4x$rows" -frames:v 1 "$sheet" 2>/dev/null
echo "$sheet"

#!/usr/bin/env bash
# Writes every nth frame of a stretch of a recording to one image, each frame stamped with its time.
# Usage: strip.sh <video.mp4> <start> <seconds> <nth> <W:H:X:Y> <out.png>
set -eu

video=$1
start=$2
seconds=$3
nth=$4
crop=$5
out=$6

rows=$(awk -v s="$seconds" -v n="$nth" 'BEGIN { print int((s * 60 / n + 3.999) / 4) }')
stamp="drawtext=text='%{pts\:flt\:$start}':fontcolor=yellow:fontsize=28:x=10:y=10:box=1:boxcolor=black"

ffmpeg -nostats -y -ss "$start" -t "$seconds" -i "$video" \
  -vf "crop=$crop,framestep=$nth,scale=768:-1,$stamp,tile=4x$rows" -frames:v 1 "$out" 2>/dev/null
echo "$out"

#!/usr/bin/env bash
# Renders the comparison crops listed in the manifest, each through the encoder its entry names.
# Run from the repository root in the dev shell after editing the manifest: tools/crops.sh
set -euo pipefail

dir=avalonia/ScreenShare.App/Assets/Crops
manifest=$dir/manifest.json
font=avalonia/ScreenShare.App/Assets/Fonts/JetBrainsMonoNL-Medium.ttf
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# A slice of desktop: coloured text at three sizes over a dark surface, a grey ramp and a fine texture under it.
ffmpeg -loglevel error -y \
  -f lavfi -i "color=c=0x1e1e1e:s=1920x1080:d=1" \
  -f lavfi -i "gradients=s=104x8:c0=0x202020:c1=0xe0e0e0:x0=0:y0=0:x1=104:y1=0:d=1" \
  -f lavfi -i "mandelbrot=s=104x16:start_scale=0.4:end_scale=0.4,trim=end_frame=1" \
  -filter_complex "[0][1]overlay=36:88[ramp];[ramp][2]overlay=36:100,
    drawtext=fontfile=$font:text='fatal\: ref':fontcolor=0xff5555:fontsize=15:x=36:y=36,
    drawtext=fontfile=$font:text='main.go\:42':fontcolor=0x6cb6ff:fontsize=13:x=36:y=56,
    drawtext=fontfile=$font:text='Build passed':fontcolor=0xd0d0d0:fontsize=11:x=36:y=73" \
  -frames:v 1 "$work/source.png"

read -r x y w h zoom < <(jq -r '[.region.x, .region.y, .region.width, .region.height, .zoom] | @tsv' "$manifest")

jq -r '.comparisons[][] | [.file, .scale, .encode] | join("|")' "$manifest" |
while IFS='|' read -r file scale encode; do
  down=null
  [[ -n $scale ]] && down="scale=$scale:flags=bicubic"
  # shellcheck disable=SC2086
  ffmpeg -loglevel error -y -i "$work/source.png" -vf "$down" $encode -frames:v 1 "$work/coded.mkv"
  ffmpeg -loglevel error -y -i "$work/coded.mkv" \
    -vf "scale=1920:1080:flags=bicubic,crop=$w:$h:$x:$y,scale=iw*$zoom:ih*$zoom:flags=neighbor" \
    -frames:v 1 "$dir/$file"
done

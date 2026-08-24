#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
output_dir=${1:-$repo_root/assets}
work_dir=$(mktemp -d /tmp/harnezpad-icons.XXXXXX)
trap 'rm -rf "$work_dir"' EXIT

mkdir -p "$output_dir"
clang -fobjc-arc -framework AppKit "$repo_root/scripts/render-harnezpad-icons.m" \
  -o "$work_dir/render-harnezpad-icons"
"$work_dir/render-harnezpad-icons" "$output_dir"
iconutil --convert icns "$output_dir/HarnezPad.iconset" --output "$output_dir/HarnezPadNative.icns"
rm -rf "$output_dir/HarnezPad.iconset"

echo "Generated Luma terminal app and menu-bar assets in $output_dir"

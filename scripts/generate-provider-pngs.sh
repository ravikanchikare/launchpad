#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
output_dir=${1:-$repo_root/assets}
work_dir=$(mktemp -d /tmp/harnezpad-provider-icons.XXXXXX)
trap 'rm -rf "$work_dir"' EXIT
mkdir -p "$output_dir"

for asset_name in codex claude-code opencode; do
  qlmanage -t -s 512 -o "$work_dir" "$repo_root/assets/$asset_name.svg" >/dev/null 2>&1
  cp "$work_dir/$asset_name.svg.png" "$output_dir/$asset_name.png"
done

echo "Generated provider PNG assets in $output_dir"

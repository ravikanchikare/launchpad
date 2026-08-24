#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
dist_dir="$repo_root/dist"
app_path="$dist_dir/HarnezPad.app"
zip_path="$dist_dir/HarnezPad-darwin-universal.zip"
dmg_path="$dist_dir/HarnezPad.dmg"
package_dir=$(mktemp -d /tmp/harnezpad-package.XXXXXX)
trap 'rm -rf "$package_dir"' EXIT

if [[ ! -d "$app_path" ]]; then
  echo "Missing packaged app: $app_path" >&2
  exit 1
fi
if [[ ! -x "$app_path/Contents/MacOS/HarnezPad" ]]; then
  echo "Missing HarnezPad application executable" >&2
  exit 1
fi
if [[ ! -x "$app_path/Contents/Resources/harnezpad" ]]; then
  echo "Missing bundled HarnezPad CLI/helper" >&2
  exit 1
fi

rm -f "$zip_path" "$dmg_path"
(
  cd "$dist_dir"
  # -X strips platform-specific extra fields; update archives must not contain
  # __MACOSX entries because the updater replaces the installed bundle in place.
  COPYFILE_DISABLE=1 /usr/bin/zip -r -X "$zip_path" HarnezPad.app
)
cp -R "$app_path" "$package_dir/HarnezPad.app"
ln -s /Applications "$package_dir/Applications"
hdiutil create -volname HarnezPad -srcfolder "$package_dir" -ov -format UDZO "$dmg_path" >/dev/null

if /usr/bin/unzip -Z1 "$zip_path" | grep -q '^__MACOSX/'; then
  echo "Update archive unexpectedly contains __MACOSX metadata" >&2
  exit 1
fi

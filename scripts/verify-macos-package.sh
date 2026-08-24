#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
app_path=${1:-$repo_root/dist/HarnezPad.app}
expected_bundle_id=${HARNEZPAD_BUNDLE_ID:-com.harnezai.launchpad}

if [[ ! -d "$app_path" ]]; then
  echo "Missing packaged app: $app_path" >&2
  exit 1
fi

plist="$app_path/Contents/Info.plist"
host="$app_path/Contents/MacOS/HarnezPad"
helper="$app_path/Contents/Resources/harnezpad"
app_icon="$app_path/Contents/Resources/HarnezPad.icns"
menu_icon="$app_path/Contents/Resources/HarnezPadMenuBar.png"
menu_icon_2x="$app_path/Contents/Resources/HarnezPadMenuBar@2x.png"
frontend_dist="$app_path/Contents/Resources/frontend/dist"
runtime_assets=(
  "$app_path/Contents/Resources/assets/HarnezPadNativeIcon.png"
  "$app_path/Contents/Resources/assets/HarnezPadMenuBarNative.png"
  "$app_path/Contents/Resources/assets/HarnezPadMenuBarNative@2x.png"
)

if find "$app_path/Contents/Resources/assets" -type f \
  \( -name '*.go' -o -name '*.zig' -o -name '*.native' -o -name '*.m' -o -name '*.sh' -o -name '*.md' -o -name '*.zon' \) \
  -print -quit | grep -q .; then
  echo "Source or build-metadata file found in packaged runtime assets" >&2
  exit 1
fi
if find "$app_path/Contents/Resources/assets" -type f \
  ! -name 'HarnezPadNativeIcon.png' \
  ! -name 'HarnezPadMenuBarNative.png' \
  ! -name 'HarnezPadMenuBarNative@2x.png' \
  -print -quit | grep -q .; then
  echo "Unexpected legacy native asset found in packaged runtime assets" >&2
  exit 1
fi
node "$repo_root/scripts/verify-frontend-dist.mjs" "$frontend_dist"
for package_entry in "$plist" "$host" "$helper" "$app_icon" "$menu_icon" "$menu_icon_2x" "${runtime_assets[@]}"; do
  if [[ ! -e "$package_entry" ]]; then
    echo "Missing required package entry: $package_entry" >&2
    exit 1
  fi
done

bundle_id=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$plist")
executable=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$plist")
display_name=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleDisplayName' "$plist")
icon_file=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIconFile' "$plist")
if [[ "$bundle_id" != "$expected_bundle_id" ]]; then
  echo "Unexpected bundle identifier: $bundle_id" >&2
  exit 1
fi
if [[ "$executable" != "HarnezPad" || "$display_name" != "HarnezPad" || "$icon_file" != "HarnezPad.icns" ]]; then
  echo "HarnezPad bundle name/executable metadata is inconsistent" >&2
  exit 1
fi

for binary in "$host" "$helper"; do
  arches=$(lipo -archs "$binary")
  if [[ "$arches" != *arm64* || "$arches" != *x86_64* ]]; then
    echo "Expected a universal binary at $binary; found: $arches" >&2
    exit 1
  fi
done

codesign --verify --deep --strict "$app_path"
echo "Verified universal HarnezPad package ($expected_bundle_id)"

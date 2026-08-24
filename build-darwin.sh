#!/bin/zsh
set -euo pipefail

# Build a universal HarnezPad application.
#
# Usage:
#   ./build-darwin.sh [version] [stable|test] [native|legacy]
#
# The native target packages the Native SDK host as Contents/MacOS/HarnezPad and
# the universal Go CLI/helper as Contents/Resources/harnezpad. The legacy target
# keeps the Cocoa/WebView application available as a rollback artifact while
# the migration is validated.

version=${1:-0.1.0}
channel=${2:-stable}
app_flavor=${3:-native}
update_api=${HARNEZPAD_UPDATE_API:-https://updates.example.com/api/update}
signing_identity=${HARNEZPAD_CODESIGN_IDENTITY:--}
bundle_id=${HARNEZPAD_BUNDLE_ID:-com.harnezai.launchpad}
repo_root=${0:A:h}
dist_dir="$repo_root/dist"
build_dir=$(mktemp -d /tmp/harnezpad-build.XXXXXX)
trap 'rm -rf "$build_dir"' EXIT

env_file="$repo_root/.env/local"
if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

typeset -a native_cli_command
if [[ -n "${NATIVE_CLI:-}" ]]; then
  native_cli_command=("$NATIVE_CLI")
else
  native_cli_command=(npx --yes @native-sdk/cli@0.7.0)
fi

if [[ ! "$version" =~ '^[0-9]+\.[0-9]+\.[0-9]+([+-][A-Za-z0-9.-]+)?$' ]]; then
  echo "Invalid version: $version" >&2
  exit 2
fi
if [[ "$channel" != "stable" && "$channel" != "test" ]]; then
  echo "Invalid update channel: $channel" >&2
  exit 2
fi
if [[ "$app_flavor" != "native" && "$app_flavor" != "legacy" ]]; then
  echo "Invalid application flavor: $app_flavor (expected native or legacy)" >&2
  exit 2
fi
if [[ ! "$bundle_id" =~ '^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$' ]]; then
  echo "Invalid bundle identifier: $bundle_id" >&2
  exit 2
fi

for command_name in go lipo codesign plutil clang iconutil qlmanage; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done
if [[ "$app_flavor" == "native" ]] && ! command -v pnpm >/dev/null 2>&1; then
  echo "Missing required command for the bundled frontend: pnpm" >&2
  exit 1
fi

ldflags="-s -w -X harnezpad/internal/version.Version=$version -X harnezpad/internal/version.UpdateChannel=$channel -X harnezpad/internal/version.DefaultUpdateAPI=$update_api"
if [[ -n "${HARNEZPAD_UPDATE_SIGNING_SECRET:-}" ]]; then
  ldflags="$ldflags -X harnezpad/internal/version.UpdateSigningSecret=$HARNEZPAD_UPDATE_SIGNING_SECRET"
fi

build_go_slice() {
  local arch=$1
  local clang_arch=$2
  (
    cd "$repo_root"
    CGO_ENABLED=1 \
      GOOS=darwin \
      GOARCH="$arch" \
      CGO_CFLAGS="-arch $clang_arch" \
      CGO_LDFLAGS="-arch $clang_arch" \
      go build -trimpath -ldflags "$ldflags" -o "$build_dir/harnezpad-helper-$arch" ./cmd/harnezpad
  )
}

build_go_helper() {
  build_go_slice amd64 x86_64
  build_go_slice arm64 arm64
  lipo -create "$build_dir/harnezpad-helper-amd64" "$build_dir/harnezpad-helper-arm64" \
    -output "$build_dir/harnezpad-helper"
}

build_native_slice() {
  local target=$1
  local output=$2
  (
    cd "$repo_root"
    "${native_cli_command[@]}" build . --yes -Dtarget="$target" -Doptimize=ReleaseFast
  )
  if [[ ! -x "$repo_root/zig-out/bin/HarnezPad" ]]; then
    echo "Native build did not produce zig-out/bin/HarnezPad" >&2
    exit 1
  fi
  cp "$repo_root/zig-out/bin/HarnezPad" "$output"
}

build_native_host() {
  if [[ -n "${HARNEZPAD_NATIVE_BINARY:-}" ]]; then
    if [[ ! -x "$HARNEZPAD_NATIVE_BINARY" ]]; then
      echo "HARNEZPAD_NATIVE_BINARY is not executable: $HARNEZPAD_NATIVE_BINARY" >&2
      exit 1
    fi
    cp "$HARNEZPAD_NATIVE_BINARY" "$build_dir/HarnezPad-native"
  else
    if ! command -v "${native_cli_command[1]}" >/dev/null 2>&1; then
      echo "Missing Native SDK CLI launcher: ${native_cli_command[1]}" >&2
      exit 1
    fi
    build_native_slice aarch64-macos "$build_dir/HarnezPad-native-arm64"
    build_native_slice x86_64-macos "$build_dir/HarnezPad-native-amd64"
    lipo -create "$build_dir/HarnezPad-native-arm64" "$build_dir/HarnezPad-native-amd64" \
      -output "$build_dir/HarnezPad-native"
  fi

  local native_arches
  native_arches=$(lipo -archs "$build_dir/HarnezPad-native")
  if [[ "$native_arches" != *arm64* || "$native_arches" != *x86_64* ]]; then
    echo "Native host must contain arm64 and x86_64 slices; found: $native_arches" >&2
    exit 1
  fi
}

apply_bundle_metadata() {
  local plist=$1
  /usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier $bundle_id" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleName HarnezPad" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleDisplayName HarnezPad" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleExecutable HarnezPad" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleIconFile HarnezPad.icns" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleVersion $version" "$plist"
  /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $version" "$plist"
}

package_native_app() {
  "$repo_root/scripts/build-frontend.sh"
  build_native_host
  (
    cd "$repo_root"
    "${native_cli_command[@]}" package \
      --target macos \
      --manifest app.zon \
      --output "$dist_dir/HarnezPad.app" \
      --binary "$build_dir/HarnezPad-native" \
      --assets "$repo_root/frontend/dist" \
      --web-layer auto \
      --signing none
  )
  if [[ ! -f "$dist_dir/HarnezPad.app/Contents/Info.plist" ]]; then
    echo "Native SDK did not produce dist/HarnezPad.app" >&2
    exit 1
  fi
  apply_bundle_metadata "$dist_dir/HarnezPad.app/Contents/Info.plist"
}

package_legacy_app() {
  mkdir -p "$dist_dir/HarnezPad.app/Contents/MacOS" "$dist_dir/HarnezPad.app/Contents/Resources"
  cp "$build_dir/harnezpad-helper" "$dist_dir/HarnezPad.app/Contents/MacOS/HarnezPad"
  cp "$repo_root/Info.plist" "$dist_dir/HarnezPad.app/Contents/Info.plist"
  apply_bundle_metadata "$dist_dir/HarnezPad.app/Contents/Info.plist"
}

sign_binary() {
  local binary_path=$1
  if [[ "$signing_identity" == "-" ]]; then
    codesign --force --sign - "$binary_path"
  else
    codesign --force --options runtime --timestamp --sign "$signing_identity" "$binary_path"
  fi
}

sign_app() {
  local app_path=$1
  if [[ "$signing_identity" == "-" ]]; then
    codesign --force --sign - "$app_path"
  else
    codesign --force --options runtime --timestamp --sign "$signing_identity" "$app_path"
  fi
}

build_go_helper
"$repo_root/scripts/generate-macos-icons.sh" "$build_dir/generated-icons"
rm -rf "$dist_dir"
mkdir -p "$dist_dir"

if [[ "$app_flavor" == "native" ]]; then
  package_native_app
else
  package_legacy_app
fi

# Native SDK copies the manifest asset directory verbatim. Rebuild it from the
# runtime allowlist below so source-only files and build metadata never ship.
rm -rf "$dist_dir/HarnezPad.app/Contents/Resources/assets"
mkdir -p "$dist_dir/HarnezPad.app/Contents/Resources/assets"
cp "$build_dir/harnezpad-helper" "$dist_dir/HarnezPad.app/Contents/Resources/harnezpad"
cp "$build_dir/generated-icons/HarnezPadNative.icns" "$dist_dir/HarnezPad.app/Contents/Resources/HarnezPad.icns"
cp "$build_dir/generated-icons/HarnezPadMenuBarNative.png" "$dist_dir/HarnezPad.app/Contents/Resources/HarnezPadMenuBar.png"
cp "$build_dir/generated-icons/HarnezPadMenuBarNative@2x.png" "$dist_dir/HarnezPad.app/Contents/Resources/HarnezPadMenuBar@2x.png"
cp "$build_dir/generated-icons/HarnezPadNativeIcon.png" "$dist_dir/HarnezPad.app/Contents/Resources/assets/HarnezPadNativeIcon.png"
cp "$build_dir/generated-icons/HarnezPadMenuBarNative.png" "$dist_dir/HarnezPad.app/Contents/Resources/assets/HarnezPadMenuBarNative.png"
cp "$build_dir/generated-icons/HarnezPadMenuBarNative@2x.png" "$dist_dir/HarnezPad.app/Contents/Resources/assets/HarnezPadMenuBarNative@2x.png"
chmod +x "$dist_dir/HarnezPad.app/Contents/MacOS/HarnezPad" "$dist_dir/HarnezPad.app/Contents/Resources/harnezpad"

# Sign nested code first, then seal the application bundle.
sign_binary "$dist_dir/HarnezPad.app/Contents/Resources/harnezpad"
sign_app "$dist_dir/HarnezPad.app"

if [[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$dist_dir/HarnezPad.app/Contents/Info.plist")" != "$bundle_id" ]]; then
  echo "Packaged bundle identifier does not match $bundle_id" >&2
  exit 1
fi
lipo -archs "$dist_dir/HarnezPad.app/Contents/Resources/harnezpad" | grep -q arm64
lipo -archs "$dist_dir/HarnezPad.app/Contents/Resources/harnezpad" | grep -q x86_64

if [[ "$app_flavor" == "native" ]]; then
  "$repo_root/scripts/verify-macos-package.sh" "$dist_dir/HarnezPad.app"
fi
"$repo_root/scripts/repackage-macos.sh"
echo "Built $app_flavor universal dist/HarnezPad.app, dist/HarnezPad-darwin-universal.zip, and dist/HarnezPad.dmg"
echo "Install with: make install-native   (or ./scripts/install-macos-app.sh dist/HarnezPad.app)"
echo "Note: cp -R into /Applications merges stale files; use install-macos-app.sh instead."

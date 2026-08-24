#!/bin/zsh
set -euo pipefail

# Replace /Applications/HarnezPad.app with a freshly built bundle.
#
# `cp -R` merges into an existing .app and leaves stale binaries behind;
# this script removes the destination first and copies with ditto instead.
#
# Usage:
#   ./scripts/install-macos-app.sh [path/to/HarnezPad.app]

repo_root=${0:A:h:h}
source_app=${1:-"$repo_root/dist/HarnezPad.app"}
dest_app="/Applications/HarnezPad.app"
cli_link="${HOME}/.local/bin/harnezpad"

if [[ ! -d "$source_app" ]]; then
  echo "Missing app bundle: $source_app" >&2
  echo "Run: make package-native VERSION=x.y.z CHANNEL=test" >&2
  exit 1
fi
if [[ ! -x "$source_app/Contents/Resources/harnezpad" ]]; then
  echo "Missing bundled CLI: $source_app/Contents/Resources/harnezpad" >&2
  exit 1
fi

source_version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$source_app/Contents/Info.plist")

echo "Installing HarnezPad ${source_version} to ${dest_app}"

# Stop a running instance so files aren't locked and the old build exits.
if pgrep -x HarnezPad >/dev/null 2>&1; then
  echo "Quitting running HarnezPad…"
  osascript -e 'tell application "HarnezPad" to quit' >/dev/null 2>&1 || true
  for _ in {1..20}; do
    pgrep -x HarnezPad >/dev/null 2>&1 || break
    sleep 0.25
  done
  if pgrep -x HarnezPad >/dev/null 2>&1; then
    echo "HarnezPad is still running; quit it manually and rerun this script." >&2
    exit 1
  fi
fi

if [[ -d "$dest_app" ]]; then
  rm -rf "$dest_app"
fi
/usr/bin/ditto "$source_app" "$dest_app"

mkdir -p "$(dirname "$cli_link")"
if [[ -e "$cli_link" && ! -L "$cli_link" ]]; then
  echo "Refusing to replace existing file at $cli_link" >&2
  exit 1
fi
rm -f "$cli_link"
ln -s "$dest_app/Contents/Resources/harnezpad" "$cli_link"

installed_version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$dest_app/Contents/Info.plist")
cli_version=$("$dest_app/Contents/Resources/harnezpad" version)

echo "Installed HarnezPad.app ${installed_version}"
echo "CLI: ${cli_version} -> ${cli_link}"
echo "Launch with: open ${dest_app}"

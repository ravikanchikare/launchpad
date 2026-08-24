#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
pin_file="$repo_root/.native-sdk-version"
sdk_path=${NATIVE_SDK_PATH:-$HOME/code/native}

if [[ ! -f "$pin_file" ]]; then
  echo "Missing Native SDK pin: $pin_file" >&2
  exit 1
fi

source "$pin_file"
: "${NATIVE_SDK_VERSION:?missing NATIVE_SDK_VERSION in $pin_file}"
: "${NATIVE_SDK_COMMIT:?missing NATIVE_SDK_COMMIT in $pin_file}"
: "${NATIVE_SDK_REMOTE:?missing NATIVE_SDK_REMOTE in $pin_file}"

if [[ ! -d "$sdk_path/.git" ]]; then
  echo "Native SDK checkout not found at $sdk_path" >&2
  echo "Set NATIVE_SDK_PATH to a checkout of $NATIVE_SDK_REMOTE" >&2
  exit 1
fi

actual_commit=$(git -C "$sdk_path" rev-parse HEAD)
actual_version=$(git -C "$sdk_path" describe --tags --exact-match 2>/dev/null || true)
manifest_version=$(sed -n 's/^[[:space:]]*"version":[[:space:]]*"\([^"]*\)".*/\1/p' "$sdk_path/packages/native-sdk/package.json" | head -1)
if [[ "$actual_commit" != "$NATIVE_SDK_COMMIT" ]]; then
  echo "Native SDK commit mismatch: expected $NATIVE_SDK_COMMIT, found $actual_commit" >&2
  exit 1
fi
if [[ -n "$actual_version" && "$actual_version" != "$NATIVE_SDK_VERSION" ]]; then
  echo "Native SDK tag mismatch: expected $NATIVE_SDK_VERSION, found $actual_version" >&2
  exit 1
fi
if [[ "$manifest_version" != "${NATIVE_SDK_VERSION#v}" ]]; then
  echo "Native SDK package version mismatch: expected ${NATIVE_SDK_VERSION#v}, found ${manifest_version:-missing}" >&2
  exit 1
fi

echo "Native SDK pin verified: $NATIVE_SDK_VERSION ($NATIVE_SDK_COMMIT)"

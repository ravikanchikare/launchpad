#!/bin/zsh
set -euo pipefail

repo_root=${0:A:h:h}
frontend_dir="$repo_root/frontend"

if [[ ! -f "$frontend_dir/package.json" || ! -f "$frontend_dir/pnpm-lock.yaml" ]]; then
  echo "Frontend package.json and pnpm-lock.yaml are required for deterministic builds" >&2
  exit 1
fi
if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm is required to build the frontend" >&2
  exit 1
fi

(
  cd "$frontend_dir"
  pnpm install --frozen-lockfile
  pnpm lint
  pnpm test
  pnpm build
)
node "$repo_root/scripts/verify-frontend-dist.mjs" "$frontend_dir/dist"

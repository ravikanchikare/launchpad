# Plan: HarnezPad (as-built)

Source spec: intent/harnezpad/spec.md
Status: accepted
Source: reverse-documented from the shipping tree (2026-08-24)

This plan is the audit trail for the imported codebase, not a future implementation queue. Later changes copy `intent/_templates/` into a new slug and write a new plan before editing.

## Files that change (ownership)

| Area | Path | Role |
| --- | --- | --- |
| Native host | `src/*.zig`, `app.zon` | Window, menu, tray, helper spawn, JS bridge |
| Entrypoint | `cmd/harnezpad/main.go` | CLI vs desktop vs `serve-native` |
| App + API | `internal/app/` | Manager, Keychain-backed settings, JSON API, native helper |
| CLI | `internal/cli/` | `launch`, `help`, `version`, restore, hidden `_compat-probe` |
| Gateway | `internal/gateway/` | HTTP client for models, keys, account |
| Launch | `internal/launch/` | Claude / Codex / ChatGPT / OpenCode adapters |
| Picker | `internal/picker/` | Interactive model picker + ChatGPT restart confirm |
| Platform | `internal/platform/` | Keychain, menubar, SF Symbols, legacy window |
| UI (ship) | `frontend/` | React/Vite Launch, Models, Keys, Settings, onboarding |
| UI (rollback) | `internal/ui/` | Cocoa/WebView host — not the user path |
| Update | `internal/update/` | Signed zip/DMG install |
| Package | `build-darwin.sh`, `scripts/` | Universal `.app`, zip, DMG, verify |
| Brand | `assets/` | App/menu icons, agent tiles, README GIF |

## Order of work (how a change should land)

1. Intent → spec → this plan (or a new slug’s plan).
2. Helper/API and launch adapters (`internal/`) with `go test`.
3. Frontend (`frontend/src`) with `pnpm test` / `pnpm lint`.
4. Native host only if window/menu/bridge/packaging changes.
5. `make package-native` / `make verify-package` before a release.

## Tests that prove it

Drive shipped entry points; do not re-implement the unit under test.

| Proof | Where |
| --- | --- |
| CLI help lists the four `harnezpad launch` commands | `internal/cli/run_test.go` |
| README tells users how to connect and launch | `internal/cli/run_test.go` |
| Launch flags, Codex/ChatGPT config, env isolation | `internal/launch/*_test.go` |
| Key slugs, management-key pinning, settings API | `internal/keys`, `internal/app` |
| Native helper contract (bootstrap, CORS, auth) | `internal/app/native_helper_contract_test.go` |
| Frontend launch commands, routing, sidebar | `frontend/src/lib/*.test.ts` |
| Packaged bundle layout | `scripts/verify-macos-package.sh` |

```sh
make test
make frontend-test
```

## Risks

- Rewriting `internal/ui/` as if it were the product UI. It is rollback-only.
- Normalizing gateway model IDs. Pass them through.
- Storing the management key anywhere but Keychain.
- `cp -R` into an existing `.app` (merges stale binaries). Use `make install-native`.

## Out of scope

- Claude Code hooks as approval gates, CI agent-evals, production `bands.yaml`.
- Shipping `_compat-probe` inside the `.app` UX.
- Changing GitHub visibility or force-pushing after the initial import (except the authorized history rewrite that publishes this tree).

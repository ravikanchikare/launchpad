# HarnezPad agent instructions

HarnezPad is a macOS launcher that connects coding agents (Claude Code, Codex CLI, ChatGPT desktop, OpenCode) to a team LLM gateway. The shipping product is a Native SDK (Zig) window host, a Go helper/CLI (`harnezpad`), and a bundled React/Vite UI (`frontend/`, served at `zero://app`). The Cocoa/WebView host under `internal/ui/` is rollback-only.

Claude Code loads this file through `CLAUDE.md` (`@AGENTS.md`). Keep that import; do not fork a second set of project instructions.

## Commands

```sh
make test                 # go test ./... && go vet ./...
make frontend-test        # pnpm test in frontend/
make dev-native           # native-helper, then Native SDK `dev`
make package-native VERSION=x.y.z CHANNEL=test
make install-native       # quit, rm, ditto — never `cp -R` into an existing .app
```

`go run ./cmd/harnezpad` skips CLI install by design. The packaged app installs `~/.local/bin/harnezpad` → `HarnezPad.app/Contents/Resources/harnezpad`.

## Conventions

- Feature branches and pull requests after the initial import; do not push follow-up work straight to `main`.
- User-facing copy says **management key**, never “gateway token”, “API token”, or “Gateway key”. Store it in Keychain service `com.harnezai.launchpad.keys` as slug `management-key`. Never commit key values or `.env` files.
- The management key is pinned at the top of Keys and must not be editable, blockable, or deletable there.
- Native UI layout lives in `frontend/src/index.css`. Do not adjust Cocoa content insets for sidebar/content spacing. Sidebar vibrancy is native `NSVisualEffectMaterialSidebar` behind a transparent DOM — not CSS `backdrop-filter` on the sidebar.
- Application UI: flat cards (thin border, no shadow); keep elevation on Dialog, Sheet, Popover, and DropdownMenu. Compact font scale. Settings has no titlebar header.
- Setup lives in Settings and the onboarding dialog. There is no Help WebView.
- Launch passes gateway model IDs through unchanged (`--model` / `-m`). Default launch key slug is `management-key`.
- OpenCode keeps its dark `#131010` brand tile; fix dark-mode contrast with CSS only.

## AI-Native SDLC

Work in this repository follows the [AI-Native SDLC](https://claude.com/blog/the-ai-native-sdlc-playbook) loop. Each stage **writes a committed artifact** the next stage **reads**. Humans stay accountable at the gates (accept/reject). Do not start the next stage from a chat transcript.

```text
Plan → Design → Build → Test → Deploy → Maintain
  │       │        │       │        │         │
  ▼       ▼        ▼       ▼        ▼         ▼
intent.md → spec.md → plan.md → diff + tests → PR + review → next intent.md
```

| Stage | What you do | Artifact the next stage reads |
| --- | --- | --- |
| **Plan** | Capture the problem, outcome, users, and constraints in the originator’s words. Product owner accepts or closes. | `intent/<slug>/intent.md` |
| **Design** | Turn accepted intent into requirements and design, flagging policy conflicts. Product owner signs off. | `intent/<slug>/spec.md` |
| **Build** | Start in plan mode. Name files, order, risks, and tests. Engineer accepts the plan **before** edits. If implementation diverges, update the plan in the same commit. | `intent/<slug>/plan.md`, then the code diff |
| **Test** | Prove the shipped change. Tests drive real entry points and committed artifacts — no re-implementation, no hard-coded stand-ins. | the tests that land with the diff |
| **Deploy** | Open a pull request. Review against `plan.md`. Human review stays on regulated or high-risk diffs. | the PR and its review findings |
| **Maintain** | Production issues and broken expectations write the next intent. The loop restarts. | a new `intent/<slug>/intent.md` |

### Artifact home

Copy templates from `intent/_templates/` into `intent/<slug>/`. Do not implement from an uncommitted plan. Do not skip `intent.md` / `spec.md` / `plan.md` on non-trivial work. The founding product record is `intent/harnezpad/`.

This repo does **not** yet adopt later plays (hooks as approval gates, CI agent-evals, production `bands.yaml` / Claude Tag). The artifact chain above is the required lifecycle.

## Layout (where to change things)

| Path | Role |
| --- | --- |
| `src/` | Native SDK Zig window host |
| `cmd/harnezpad`, `internal/` | Go CLI, helper API, launchers, Keychain, updates |
| `frontend/` | Shipping React/Vite UI |
| `internal/ui/` | Legacy WebView UI (rollback only) |
| `assets/` | App icons, brand art, README GIF |
| `intent/` | SDLC intent/spec/plan home |

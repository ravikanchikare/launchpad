# Spec: HarnezPad

Source intent: intent/harnezpad/intent.md
Status: accepted
Source: reverse-documented from the shipping tree (2026-08-24)

## Requirements

### Connect

- R1. On first launch, or when the stored key is missing, expired, or invalid, show onboarding that accepts a Full Access key.
- R2. Settings → Gateway shows the managed endpoint and lets the user paste/save the same key.
- R3. Validate the key against the gateway before persist. Store it in Keychain as slug `management-key`. Settings JSON holds URL and metadata only.
- R4. Inference-only keys cannot call `/key/*`. Models, Keys, and spend load only after a valid management key.

### Launch

- R5. CLI: `harnezpad launch claude|codex|chatgpt|opencode` with optional `--model`, `--key`, and `--restore` (Codex CLI and ChatGPT).
- R6. Missing `--model` opens an interactive picker from `GET /model_group/info` for that key. Non-interactive use must pass `--model`.
- R7. Claude: process-scoped `ANTHROPIC_BASE_URL` / `ANTHROPIC_AUTH_TOKEN` / model-tier vars / `CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1`; strip AWS/Bedrock/Vertex inheritance.
- R8. Codex CLI: HarnezPad-owned `$CODEX_HOME` profile + catalog; `OPENAI_API_KEY` is the launch token.
- R9. ChatGPT: persist routing in `~/.codex/config.toml`, catalog + `harnezpad _chatgpt-token` auth, launch the desktop app. `--restore` reverts.
- R10. OpenCode: inject `OPENCODE_CONFIG_CONTENT` with an OpenAI-compatible HarnezPad provider.

### App shell

- R11. Native SDK window host (`src/`, `app.zon`) renders `frontend/` at `zero://app`.
- R12. Go helper (`harnezpad serve-native --parent-pid`) serves the JSON API on loopback with a per-session bearer, and exits with its parent.
- R13. Packaged app installs `~/.local/bin/harnezpad` → `HarnezPad.app/Contents/Resources/harnezpad` on first launch. `go run` skips install.
- R14. Keys page: management key pinned, not editable/blockable/deletable; named keys create/edit/block with optional model scope.
- R15. Launch page lists verified `harnezpad launch … --model …` commands per agent. Models page is the live catalog.
- R16. Signed self-update (HMAC-SHA256) is supported; menubar Quit fully exits so the `.app` can be replaced.

## Design

```text
HarnezPad.app
  Contents/MacOS/HarnezPad          Zig Native SDK host (windows, menus, tray)
  Contents/Resources/harnezpad      Go CLI + helper
  Contents/Resources/frontend/dist  Bundled React UI (zero://app)
```

- Host allowlists bridge commands and starts the helper on an ephemeral loopback port.
- Helper owns Keychain, gateway HTTP, launch, and updates (`internal/app`, `internal/gateway`, `internal/launch`, `internal/update`).
- CLI is the same binary: no args → legacy desktop path; `launch` / `help` / `version` / `serve-native` are the native path.
- Settings file: `~/Library/Application Support/HarnezPad/settings.json` (no secrets).
- Default gateway URL is compiled in (`https://gateway.example.com`) and shown as managed.

## Areas of concern

- ChatGPT restore mutates the user’s root `~/.codex/config.toml`; `--restore` must remain reliable.
- Claude Code’s `/model` picker only lists IDs starting `claude` or `anthropic`; other aliases still work via `--model`.
- Bedrock Anthropic models fail Codex `/v1/responses` streaming (created vs completed ID mismatch); Launch omits them for Codex/ChatGPT.
- Legacy WebView (`internal/ui/`) remains in the Go binary for `make package-legacy` only. Do not present it as the product.

## Open questions carried forward

None. New work gets a new intent.

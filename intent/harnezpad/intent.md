# Intent: HarnezPad

Author: Ravi Kanchikare
Status: accepted
Source: reverse-documented from the shipping tree (2026-08-24)

## Problem

Coding agents (Claude Code, Codex CLI, ChatGPT desktop, OpenCode) each speak a
different provider protocol and store credentials in different places. A team
that already runs one LLM gateway still has to hand-configure every agent:
base URLs, auth tokens, model IDs, and leftover AWS/Bedrock/Vertex variables
that silently override the gateway. That setup is easy to get wrong and easy
to leak into git or shell rc files.

## Proposed outcome

HarnezPad is a macOS app plus a `harnezpad` CLI that:

1. Connects to the team gateway with one Full Access key stored in Keychain.
2. Launches the user’s preferred coding agent already routed at that gateway.
3. Lets the user pick a gateway model interactively or with `--model`.
4. Manages additional virtual keys in-app without putting secrets on disk.

The user opens HarnezPad, pastes a key in onboarding or Settings, then runs
`harnezpad launch claude|codex|chatgpt|opencode`.

## Affected users and systems

- Individual engineers on macOS using Claude Code, Codex, ChatGPT, or OpenCode
- The team LLM gateway (LiteLLM-compatible: `/model_group/info`, `/key/*`, `/team/{id}/members/me`)
- macOS Keychain (`com.harnezai.launchpad.keys`), `~/.codex/config.toml` (ChatGPT), process-scoped env (Claude)

## Constraints

- Product name is **HarnezPad**; CLI and module are **harnezpad**. Never “Harness”.
- User-facing copy says **management key**, not “gateway token” / “API token”.
- The management key is never written to settings JSON, logs, or git.
- Gateway model IDs pass through unchanged (`--model` / `-m`).
- Native SDK (Zig) host + Go helper + bundled React UI is the shipping product.
- The Cocoa/WebView UI under `internal/ui/` is rollback-only.
- No Help WebView; setup lives in Settings and onboarding.
- Feature work after this import uses branches and pull requests.

## Open questions

None for the imported product. Later work starts a new `intent/<slug>/`.

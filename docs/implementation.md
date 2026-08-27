# Implementation

Launchpad is a Go application with an embedded React interface. The same process serves the management API, the production UI, and the local Claude Desktop compatibility endpoints.

## Request flow

```text
CLI or desktop UI
        |
        v
Provider settings + credential resolution
        |
        v
Model discovery and selection
        |
        v
Tool-specific adapter
        |
        +-- process arguments and environment
        +-- managed desktop profile
        `-- local Claude Desktop compatibility service
```

The main boundaries are:

- `internal/config` — provider URL and build-time CLI name.
- `internal/credentials` — `LITELLM_API_KEY` and macOS Keychain resolution.
- `internal/provider` — provider HTTP client and model discovery.
- `internal/picker` — terminal model and confirmation interfaces.
- `internal/launch` — tool-specific arguments, environments, profiles, and restore behavior.
- `internal/server` — desktop UI API and Claude Desktop compatibility endpoints.

## Provider configuration

Launchpad currently supports LiteLLM and compatible OpenAI-style providers.

| Setting | Source | Precedence |
| --- | --- | --- |
| Provider URL | `LITELLM_BASE_URL` | First |
| Provider URL | `providerUrl` in Launchpad settings | Second |
| Provider URL | `http://localhost:4000` | Default |
| Provider key | `LITELLM_API_KEY` | First |
| Provider key | macOS Keychain | Second |

The provider client requests `/model_group/info` first because it carries model-group metadata. It falls back to `/v1/models` when the key only has inference access. Model IDs are not rewritten by the CLI.

Provider keys are removed from the inherited environment before a child tool starts. Each adapter adds only the variables that tool requires.

## Claude Code

Claude Code starts as a child process with the selected model passed through `--model`. Launchpad also appends:

```text
--setting-sources project,local
```

The adapter configures:

| Variable | Value |
| --- | --- |
| `ANTHROPIC_BASE_URL` | Provider root URL |
| `ANTHROPIC_AUTH_TOKEN` | Provider key |
| `ANTHROPIC_API_KEY` | Empty, so the auth token is authoritative |
| `ANTHROPIC_MODEL` | Selected provider model ID |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | Selected provider model ID |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | Selected provider model ID |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | Selected provider model ID |
| `CLAUDE_CODE_SUBAGENT_MODEL` | Selected provider model ID |
| `CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY` | `1`, the upstream Claude Code variable for model discovery |

This configuration is process-scoped and does not overwrite the user's global Claude Code settings.

## Codex

Codex accepts provider configuration through repeated `-c` arguments. Launchpad starts it with:

```text
model_provider="launchpad"
model_providers.launchpad.name="Launchpad"
model_providers.launchpad.base_url="<provider-url>/v1"
model_providers.launchpad.env_key="OPENAI_API_KEY"
model_providers.launchpad.wire_api="responses"
```

The selected model is passed with `-m`, and `OPENAI_API_KEY` is set only in the child environment. Extra arguments after `--` are appended unchanged.

## ChatGPT

ChatGPT and Codex share the configuration under `CODEX_HOME`, which defaults to `~/.codex`.

Launchpad:

1. Captures the existing root `model`, `model_provider`, and `model_catalog_json` values.
2. Stores restore state under the user's Launchpad configuration directory.
3. Writes a Launchpad model catalog beside the ChatGPT configuration.
4. Adds a delimited `[model_providers.launchpad]` block to `config.toml`.
5. Configures the running Launchpad executable as the provider-key helper.
6. Asks before restarting ChatGPT when the application is already running.

`launchpad launch chatgpt --restore` removes the managed block, restores the captured values, removes the generated catalog, and offers to restart ChatGPT.

## Claude Desktop

Claude Desktop reads its third-party inference profile from its macOS configuration files. Launchpad preserves unrelated settings, writes the provider URL and key, and removes static model entries so Claude can discover the current catalog.

Claude Desktop expects Anthropic-compatible routes. Launchpad therefore exposes local `/v1/models`, `/v1/messages`, and `/v1/messages/count_tokens` endpoints. The adapter maps Claude's fixed model slots to available provider models and rewrites requests before forwarding them.

## Adding or extending providers

Provider behavior is concentrated in `internal/provider`.

To add another compatible provider:

1. Extend model discovery in `internal/provider/client.go` when its catalog differs from the existing LiteLLM and OpenAI response shapes.
2. Add focused HTTP fixtures in `internal/provider/client_test.go`.
3. Normalize provider-specific URL rules in `internal/config` rather than in individual launch adapters.
4. Add required protocol translation to the relevant adapter in `internal/launch`.
5. Keep credentials process-scoped and update environment-isolation tests.
6. If Claude Desktop requires a different wire protocol, implement that translation in `internal/server` and test request rewriting with an HTTP test server.

To add another application integration, register it in `internal/launch/registry.go`, implement its command or profile adapter, expose it through the server's enabled integration list, add its icon and UI ordering, and test that secrets do not leak into unrelated environment variables.

## Build and embedded UI

`npm --prefix ui run build` writes the production bundle to `ui/dist`. Go embeds that directory through `//go:embed`. Development mode keeps the Go API on port 3001 and loads the UI from Vite on port 5173.

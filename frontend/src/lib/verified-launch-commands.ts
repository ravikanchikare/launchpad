import claudeArtwork from "../../../assets/claude-code.svg"
import chatgptArtwork from "../../../assets/codex.png"
import codexArtwork from "../../../assets/codex-app.png"
import opencodeArtwork from "../../../assets/opencode.png"

export type LaunchAppId = "codex" | "chatgpt" | "claude" | "opencode"

export interface VerifiedModel {
  id: string
  provider: string
}

export interface LaunchAppConfig {
  id: LaunchAppId
  name: string
  description: string
  command: string
  artwork: string
  verifiedModels: VerifiedModel[]
  providerOrder?: string[]
  prerequisites: string[]
}

const OPENAI_MODELS: VerifiedModel[] = [
  { id: "gpt-5.5", provider: "OpenAI" },
  { id: "gpt-5.5[1m]", provider: "OpenAI" },
  { id: "gpt-5.6-luna", provider: "OpenAI" },
  { id: "gpt-5.6-luna[1m]", provider: "OpenAI" },
  { id: "gpt-5.6-sol", provider: "OpenAI" },
  { id: "gpt-5.6-sol[1m]", provider: "OpenAI" },
  { id: "gpt-5.6-terra", provider: "OpenAI" },
  { id: "gpt-5.6-terra[1m]", provider: "OpenAI" },
]

const BASETEN_MODELS: VerifiedModel[] = [
  { id: "kimi-k3", provider: "Baseten" },
  { id: "kimi-k3[1m]", provider: "Baseten" },
  { id: "kimi-k2.6", provider: "Baseten" },
  { id: "kimi-k2.7", provider: "Baseten" },
  { id: "kimi-k2.7-code", provider: "Baseten" },
  { id: "glm-5.2", provider: "Baseten" },
  { id: "glm-5.2[1m]", provider: "Baseten" },
]

const BEDROCK_ANTHROPIC_MODELS: VerifiedModel[] = [
  { id: "claude-sonnet-4-5", provider: "Bedrock" },
  { id: "claude-sonnet-5", provider: "Bedrock" },
  { id: "claude-opus-5", provider: "Bedrock" },
]

const OPENCODE_SAMPLE_MODELS: VerifiedModel[] = [
  { id: "gpt-5.5", provider: "OpenAI" },
  { id: "kimi-k3", provider: "Baseten" },
  { id: "claude-sonnet-5", provider: "Bedrock" },
]

const SHARED_PREREQUISITES = [
  "A valid management key saved in Settings (default slug: management-key).",
  "HarnezPad CLI on PATH (~/.local/bin/harnezpad) or run from HarnezPad.app.",
]

export const launchApps: LaunchAppConfig[] = [
  {
    id: "codex",
    name: "Codex",
    description: "OpenAI's coding agent CLI",
    command: "harnezpad launch codex",
    artwork: codexArtwork,
    verifiedModels: [...OPENAI_MODELS, ...BASETEN_MODELS],
    prerequisites: [
      ...SHARED_PREREQUISITES,
      "Requires the codex CLI installed locally.",
      "HarnezPad writes an harnezpad-launch profile under $CODEX_HOME and sets OPENAI_API_KEY to your launch token.",
      "Bedrock Anthropic models are excluded — gateway streaming breaks Codex /v1/responses.",
    ],
  },
  {
    id: "chatgpt",
    name: "ChatGPT",
    description: "ChatGPT desktop app via HarnezPad routing",
    command: "harnezpad launch chatgpt",
    artwork: chatgptArtwork,
    verifiedModels: [...OPENAI_MODELS, ...BASETEN_MODELS],
    prerequisites: [
      ...SHARED_PREREQUISITES,
      "Requires ChatGPT.app (or Codex.app) installed.",
      "HarnezPad updates ~/.codex/config.toml with wire_api = \"responses\" (OpenAI Responses API) and relaunches the desktop app.",
      "OpenAI and Baseten models are listed — same Responses route as Codex CLI. Bedrock Anthropic models are excluded (gateway streaming ID mismatch on /v1/responses).",
    ],
  },
  {
    id: "claude",
    name: "Claude Code",
    description: "Anthropic's coding tool",
    command: "harnezpad launch claude",
    artwork: claudeArtwork,
    verifiedModels: [
      ...BEDROCK_ANTHROPIC_MODELS,
      ...OPENAI_MODELS,
      ...BASETEN_MODELS,
    ],
    providerOrder: ["Bedrock", "OpenAI", "Baseten"],
    prerequisites: [
      ...SHARED_PREREQUISITES,
      "Requires the claude CLI installed locally.",
      "HarnezPad sets ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN, Claude default model env vars, and CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 for the child process.",
      "Bedrock column shows representative Anthropic models verified headless; other Bedrock Anthropic IDs on the gateway use the same Anthropic wire path.",
      "Inherited AWS/Bedrock/Anthropic credentials are stripped so the gateway route wins.",
    ],
  },
  {
    id: "opencode",
    name: "OpenCode",
    description: "Anomaly's open-source coding agent",
    command: "harnezpad launch opencode",
    artwork: opencodeArtwork,
    verifiedModels: OPENCODE_SAMPLE_MODELS,
    providerOrder: ["OpenAI", "Baseten", "Bedrock"],
    prerequisites: [
      ...SHARED_PREREQUISITES,
      "Requires the opencode CLI installed locally.",
      "HarnezPad injects OPENCODE_CONFIG_CONTENT with an OpenAI-compatible gateway provider (harnezpad/<model>).",
      "One model per provider verified headless; other gateway models on the same provider should work the same way.",
    ],
  },
]

export function launchCommandForModel(appCommand: string, modelId: string) {
  return `${appCommand} --model ${modelId}`
}

const PROVIDER_ORDER = ["OpenAI", "Baseten", "Bedrock"]
const TIER_VARIANT_SUFFIX = "[1m]"

export type ModelTierFilter = "normal" | "1m"

function isTierVariantModel(id: string) {
  return id.endsWith(TIER_VARIANT_SUFFIX)
}

export function filterModelsByTier(
  models: VerifiedModel[],
  tier: ModelTierFilter
) {
  return models.filter((model) =>
    tier === "1m" ? isTierVariantModel(model.id) : !isTierVariantModel(model.id)
  )
}

/** Base models first, then [1m] tier variants after all standard entries. */
export function sortModelsForDisplay(models: VerifiedModel[]) {
  const standard: VerifiedModel[] = []
  const tiered: VerifiedModel[] = []
  for (const model of models) {
    if (isTierVariantModel(model.id)) tiered.push(model)
    else standard.push(model)
  }
  return [...standard, ...tiered]
}

export function groupVerifiedModelsByProvider(
  models: VerifiedModel[],
  providerOrder: string[] = PROVIDER_ORDER
) {
  const groups = new Map<string, VerifiedModel[]>()
  for (const model of models) {
    const list = groups.get(model.provider) ?? []
    list.push(model)
    groups.set(model.provider, list)
  }

  const ordered = providerOrder
    .filter((provider) => groups.has(provider))
    .map((provider) => ({
      provider,
      models: sortModelsForDisplay(groups.get(provider)!),
    }))

  for (const [provider, providerModels] of groups) {
    if (!providerOrder.includes(provider)) {
      ordered.push({
        provider,
        models: sortModelsForDisplay(providerModels),
      })
    }
  }

  return ordered
}

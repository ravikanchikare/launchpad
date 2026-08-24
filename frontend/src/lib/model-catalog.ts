import type { ModelTierFilter } from "@/lib/verified-launch-commands"
import type { ModelCatalogEntry } from "@/types"

const TIER_VARIANT_SUFFIX = "[1m]"

const PROVIDER_LABELS: Record<string, string> = {
  openai: "OpenAI",
  baseten: "Baseten",
  bedrock: "Bedrock",
}

const KEY_PICKER_PROVIDER_ORDER = ["baseten", "bedrock"]

/** Gateway aggregate entries like "all-proxy-models" are not selectable models. */
export function isProxyAggregateModel(model: ModelCatalogEntry) {
  const raw = model.id.trim().toLowerCase()
  const slug = raw.replace(/[\s_]+/g, "-")
  return (
    slug === "all-proxy-models" ||
    slug.includes("all-proxy-models") ||
    /all[\s_-]*proxy[\s_-]*models/.test(raw)
  )
}

export function isOpenAIModel(model: ModelCatalogEntry) {
  const providers = model.providers?.map((provider) => provider.toLowerCase()) ?? []
  if (providers.includes("openai")) return true
  return model.id.toLowerCase().startsWith("openai/")
}

export function isTierVariantModelId(id: string) {
  return id.endsWith(TIER_VARIANT_SUFFIX)
}

export function isKeyPickerModel(model: ModelCatalogEntry) {
  return !isProxyAggregateModel(model) && !isOpenAIModel(model)
}

export function filterCatalogModelsByTier(
  models: ModelCatalogEntry[],
  tier: ModelTierFilter
) {
  return models.filter((model) =>
    tier === "1m"
      ? isTierVariantModelId(model.id)
      : !isTierVariantModelId(model.id)
  )
}

export function formatProviderLabel(provider: string) {
  return PROVIDER_LABELS[provider.toLowerCase()] ?? provider
}

export function getPrimaryProvider(model: ModelCatalogEntry) {
  return model.providers?.[0]?.toLowerCase() ?? "other"
}

export function listKeyPickerProviders(models: ModelCatalogEntry[]) {
  const providers = new Set<string>()
  for (const model of models) {
    if (!isKeyPickerModel(model)) continue
    providers.add(getPrimaryProvider(model))
  }

  const ordered = KEY_PICKER_PROVIDER_ORDER.filter((provider) =>
    providers.has(provider)
  )
  for (const provider of [...providers].sort()) {
    if (!KEY_PICKER_PROVIDER_ORDER.includes(provider)) ordered.push(provider)
  }
  return ordered
}

function sortCatalogModelsForDisplay(models: ModelCatalogEntry[]) {
  const standard: ModelCatalogEntry[] = []
  const tiered: ModelCatalogEntry[] = []
  for (const model of models) {
    if (isTierVariantModelId(model.id)) tiered.push(model)
    else standard.push(model)
  }
  return [...standard, ...tiered]
}

export function groupCatalogModelsByProvider(
  models: ModelCatalogEntry[],
  providerOrder: string[] = KEY_PICKER_PROVIDER_ORDER
) {
  const groups = new Map<string, ModelCatalogEntry[]>()
  for (const model of models) {
    const provider = getPrimaryProvider(model)
    const list = groups.get(provider) ?? []
    list.push(model)
    groups.set(provider, list)
  }

  const ordered = providerOrder
    .filter((provider) => groups.has(provider))
    .map((provider) => ({
      provider,
      models: sortCatalogModelsForDisplay(groups.get(provider)!),
    }))

  for (const [provider, providerModels] of groups) {
    if (!providerOrder.includes(provider)) {
      ordered.push({
        provider,
        models: sortCatalogModelsForDisplay(providerModels),
      })
    }
  }

  return ordered
}

export function filterKeyPickerModels(
  models: ModelCatalogEntry[],
  {
    provider = "all",
    tier,
    search = "",
  }: {
    provider?: "all" | string
    tier: ModelTierFilter
    search?: string
  }
) {
  let list = models.filter(isKeyPickerModel)
  if (provider !== "all") {
    list = list.filter((model) => getPrimaryProvider(model) === provider)
  }
  list = filterCatalogModelsByTier(list, tier)
  const query = search.trim().toLowerCase()
  if (query) {
    list = list.filter((model) => model.id.toLowerCase().includes(query))
  }
  return list
}

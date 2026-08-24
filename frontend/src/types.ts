export type Route = 'agents' | 'keys' | 'settings'
export type NativeRoute = Route | 'updates' | 'models'
export type Appearance = 'light' | 'dark'
export interface NativeBootstrap { ready: boolean; helperUrl: string; helperToken: string; sidebarOpen: boolean; appearance: Appearance; version?: string }
export interface Settings { gatewayUrl: string; tokenConfigured: boolean; tokenValid: boolean; setupReason?: string; defaultKeySlug: string }
export interface AccountSummary { teamId: string; teamAlias?: string; role?: string; userEmail?: string; spend: number; maxBudget?: number; budgetResetAt?: string; rpmLimit?: number }
export interface ModelCatalogEntry { id: string; providers?: string[]; mode?: string; maxInputTokens?: number; maxOutputTokens?: number; inputCostPerToken?: number; outputCostPerToken?: number; supportsVision: boolean; supportsTools: boolean; supportsReasoning: boolean; supportsWebSearch: boolean; healthStatus?: string }
export interface KeySummary { id: string; alias: string; slug: string; models?: string[]; allModels: boolean; blocked: boolean; spend?: number; maxBudget?: number; active: boolean; management: boolean; default: boolean }
export interface KeyListPage { keys: KeySummary[]; totalCount: number; currentPage: number; totalPages: number }
export interface KeyCapabilities { supported: boolean; reason?: string }
export interface CLIStatus { installed: boolean; path: string; error?: string }
export interface UpdateStatus { currentVersion?: string; available: boolean; downloaded: boolean; update?: { version: string }; error?: string }
export interface CreatedKey { key: string; summary: KeySummary }

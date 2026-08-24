import type { AccountSummary, CLIStatus, CreatedKey, KeyCapabilities, KeyListPage, ModelCatalogEntry, Settings, UpdateStatus } from '@/types'

export class HelperClient {
  private readonly baseUrl: string
  private readonly token: string

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl
    this.token = token
  }
  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, { ...init, headers: { authorization: `Bearer ${this.token}`, 'content-type': 'application/json', ...init.headers } })
    if (!response.ok) {
      const body = await response.text()
      try { const parsed = JSON.parse(body) as { error?: string }; throw new Error(parsed.error || body || `Request failed (${response.status})`) }
      catch (error) { if (error instanceof SyntaxError) throw new Error(body || `Request failed (${response.status})`, { cause: error }); throw error }
    }
    if (response.status === 204) return undefined as T
    return response.json() as Promise<T>
  }
  settings = () => this.request<Settings>('/api/settings')
  saveSettings = (body: { token?: string; defaultKeySlug?: string }) => this.request<Settings>('/api/settings', { method: 'PUT', body: JSON.stringify(body) })
  validateToken = (token: string) => this.request<void>('/api/settings/validate', { method: 'POST', body: JSON.stringify({ token }) })
  account = () => this.request<AccountSummary>('/api/account')
  models = () => this.request<ModelCatalogEntry[]>('/api/models/catalog')
  keyCapabilities = () => this.request<KeyCapabilities>('/api/keys/capabilities')
  keys = () => this.request<KeyListPage>('/api/keys?page=1&size=100')
  createKey = (alias: string, models: string[]) => this.request<CreatedKey>('/api/keys', { method: 'POST', body: JSON.stringify({ alias, models }) })
  registerKey = (slug: string, token: string) => this.request<void>('/api/keys/register', { method: 'POST', body: JSON.stringify({ slug, token }) })
  updateKey = (id: string, alias: string, models: string[]) => this.request<void>(`/api/keys/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify({ alias, models }) })
  keyAction = (id: string, action: 'default' | 'delete' | 'block' | 'unblock') => this.request<void>(`/api/keys/${encodeURIComponent(id)}/${action}`, { method: 'POST' })
  cliStatus = () => this.request<CLIStatus>('/api/cli-status')
  updateStatus = (check = false) => this.request<UpdateStatus>(`/api/update${check ? '?check=1' : ''}`)
  installUpdate = () => this.request<void>('/api/update/install', { method: 'POST' })
}

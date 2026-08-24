import { useState } from "react"

import { KeysPage } from "@/components/keys/KeysPage"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import {
  Field,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { toast } from "@/components/ui/toast"
import { formatMoney } from "@/lib/format-money"
import { HelperClient } from "@/lib/helper"
import type { SettingsTab } from "@/lib/nav-config"
import type {
  AccountSummary,
  KeyCapabilities,
  KeySummary,
  ModelCatalogEntry,
  Settings,
} from "@/types"

export function SettingsPage({
  tab,
  client,
  settings,
  account,
  models,
  keys,
  keyCapabilities,
  keysLoading,
  keysError,
  reload,
  reloadKeys,
}: {
  tab: SettingsTab
  client: HelperClient | null
  settings: Settings | null
  account: AccountSummary | null
  models: ModelCatalogEntry[]
  keys: KeySummary[]
  keyCapabilities: KeyCapabilities | null
  keysLoading: boolean
  keysError: string
  reload: () => Promise<void>
  reloadKeys: () => Promise<void>
}) {
  const [token, setToken] = useState("")
  const [saving, setSaving] = useState(false)
  const save = async () => {
    if (!client) return
    setSaving(true)
    try {
      if (token) await client.validateToken(token)
      await client.saveSettings({ token: token || undefined })
      setToken("")
      toast.add({ title: "Settings saved", type: "success" })
      await reload()
    } catch (cause) {
      toast.add({
        title:
          cause instanceof Error
            ? cause.message
            : "Couldn't save settings. Check your management key and try again.",
        type: "error",
      })
    } finally {
      setSaving(false)
    }
  }

  return (
    <ScrollArea className="settings-scroll">
      <div className="settings-page">
        {tab === "general" ? (
          <section className="settings-section">
            <h2 className="settings-section-heading">Account</h2>
            <Card className="settings-card">
              <div className="account-grid">
                <div>
                  <div className="setting-hint">Email</div>
                  <div className="setting-label">{account?.userEmail || "—"}</div>
                </div>
                <div>
                  <div className="setting-hint">Team</div>
                  <div className="setting-label">
                    {account?.teamAlias || account?.teamId || "—"}
                  </div>
                </div>
                <div>
                  <div className="setting-hint">Role</div>
                  <div className="setting-label">{account?.role || "—"}</div>
                </div>
                <div>
                  <div className="setting-hint">Spend</div>
                  <div className="setting-label">
                    {formatMoney(account?.spend)}
                  </div>
                </div>
                <div>
                  <div className="setting-hint">Budget</div>
                  <div className="setting-label">
                    {account?.maxBudget ? formatMoney(account.maxBudget) : "—"}
                  </div>
                </div>
                <div>
                  <div className="setting-hint">Rate limit</div>
                  <div className="setting-label">
                    {account?.rpmLimit ? `${account.rpmLimit} req/min` : "—"}
                  </div>
                </div>
              </div>
            </Card>
          </section>
        ) : (
          <section className="settings-section">
            <Card className="settings-card">
              <div className="settings-rows">
                <div className="settings-row">
                  <div className="settings-row-copy">
                    <div className="setting-label">Endpoint</div>
                    <div className="setting-hint">
                      {settings?.gatewayUrl || "Loading…"}
                    </div>
                  </div>
                  <Badge variant="secondary">Managed</Badge>
                </div>
                <div className="settings-row stacked">
                  <div className="settings-row-copy">
                    <div className="setting-label">Management key</div>
                    <div className="setting-hint">
                      {settings?.tokenValid
                        ? "Connected and verified"
                        : "Paste a Full Access management key"}
                    </div>
                  </div>
                  <FieldGroup>
                    <Field orientation="horizontal" className="inline-form">
                      <FieldLabel className="sr-only" htmlFor="management-token">
                        Management key
                      </FieldLabel>
                      <Input
                        id="management-token"
                        type="password"
                        value={token}
                        onChange={(event) => setToken(event.target.value)}
                        placeholder={
                          settings?.tokenConfigured ? "••••••••••••••••" : "sk-…"
                        }
                      />
                      <Button
                        onClick={save}
                        disabled={saving || (!token && settings?.tokenValid)}
                      >
                        {saving ? "Saving…" : "Save"}
                      </Button>
                    </Field>
                  </FieldGroup>
                </div>
              </div>
            </Card>
            <div className="settings-subsection">
              <KeysPage
                client={client}
                models={models}
                keys={keys}
                capabilities={keyCapabilities}
                loading={keysLoading}
                error={keysError}
                reload={reloadKeys}
                embedded
              />
            </div>
          </section>
        )}
      </div>
    </ScrollArea>
  )
}

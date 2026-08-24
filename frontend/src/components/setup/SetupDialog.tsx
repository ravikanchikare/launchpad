import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { toast } from "@/components/ui/toast"
import { HelperClient } from "@/lib/helper"
import type { Settings } from "@/types"

export function SetupDialog({
  client,
  settings,
  onComplete,
}: {
  client: HelperClient | null
  settings: Settings | null
  onComplete: () => Promise<void>
}) {
  const [token, setToken] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [dismissed, setDismissed] = useState(false)
  const submit = async () => {
    if (!client || !token) return
    setSaving(true)
    setError("")
    try {
      await client.validateToken(token)
      await client.saveSettings({ token })
      toast.add({ title: "Management key connected", type: "success" })
      setToken("")
      setDismissed(false)
      await onComplete()
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Management key is invalid"
      )
    } finally {
      setSaving(false)
    }
  }
  useEffect(() => {
    if (settings?.tokenValid && dismissed) setDismissed(false)
  }, [settings?.tokenValid, dismissed])
  const shouldOpen = Boolean(settings && !settings.tokenValid && !dismissed)
  const reason = settings?.setupReason
  return (
    <Dialog open={shouldOpen} onOpenChange={(open) => !open && setDismissed(true)}>
      <DialogContent showCloseButton={true}>
        <DialogHeader>
          <DialogTitle>
            {reason === "expired"
              ? "Management key expired"
              : reason === "invalid"
                ? "Management key needs attention"
                : "Welcome to HarnezPad"}
          </DialogTitle>
          <DialogDescription>
            Connect a Full Access management key to load your account, models,
            and virtual keys.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="setup-token">Management key</FieldLabel>
            <Input
              id="setup-token"
              type="password"
              value={token}
              aria-invalid={error ? true : undefined}
              onChange={(event) => {
                setToken(event.target.value)
                if (error) setError("")
              }}
              placeholder="Paste your management key"
            />
            {error ? <FieldError>{error}</FieldError> : null}
          </Field>
        </FieldGroup>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setDismissed(true)} disabled={saving}>
            Skip for now
          </Button>
          <Button onClick={submit} disabled={!token || saving}>
            {saving ? "Validating…" : "Connect HarnezPad"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

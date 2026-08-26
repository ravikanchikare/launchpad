import { useMemo, useState } from "react"
import { Copy01Icon, MoreHorizontalIcon, PlusSignIcon } from "@hugeicons/core-free-icons"

import { AppIcon } from "@/components/AppIcon"
import { EmptyState } from "@/components/layout/EmptyState"
import { PageHeading } from "@/components/layout/PageHeading"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { toast } from "@/components/ui/toast"
import { copyText } from "@/lib/clipboard"
import { formatMoney } from "@/lib/format-money"
import { HelperClient } from "@/lib/helper"
import {
  filterKeyPickerModels,
  formatProviderLabel,
  groupCatalogModelsByProvider,
  isKeyPickerModel,
  listKeyPickerProviders,
} from "@/lib/model-catalog"
import { type ModelTierFilter } from "@/lib/verified-launch-commands"
import type {
  KeyCapabilities,
  KeySummary,
  ModelCatalogEntry,
} from "@/types"

interface KeyEditorState {
  mode: "create" | "edit"
  key?: KeySummary
}

export function KeysPage({
  client,
  models,
  keys,
  capabilities,
  loading,
  error,
  reload,
  embedded = false,
}: {
  client: HelperClient | null
  models: ModelCatalogEntry[]
  keys: KeySummary[]
  capabilities: KeyCapabilities | null
  loading: boolean
  error: string
  reload: () => Promise<void>
  embedded?: boolean
}) {
  const [editor, setEditor] = useState<KeyEditorState | null>(null)
  const [confirm, setConfirm] = useState<{
    key: KeySummary
    action: "delete" | "block" | "unblock"
  } | null>(null)
  const [secret, setSecret] = useState("")
  const [alias, setAlias] = useState("")
  const [allModels, setAllModels] = useState(true)
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [providerFilter, setProviderFilter] = useState<"all" | string>("all")
  const [tierFilter, setTierFilter] = useState<ModelTierFilter>("normal")
  const [modelSearch, setModelSearch] = useState("")
  const [saving, setSaving] = useState(false)
  const keyPickerModels = useMemo(
    () => models.filter(isKeyPickerModel),
    [models]
  )
  const providerOptions = useMemo(
    () => listKeyPickerProviders(models),
    [models]
  )
  const filteredModels = useMemo(
    () =>
      filterKeyPickerModels(models, {
        provider: providerFilter,
        tier: tierFilter,
        search: modelSearch,
      }),
    [models, providerFilter, tierFilter, modelSearch]
  )
  const providerGroups = useMemo(() => {
    if (providerFilter !== "all") {
      return filteredModels.length
        ? [{ provider: providerFilter, models: filteredModels }]
        : []
    }
    return groupCatalogModelsByProvider(filteredModels)
  }, [filteredModels, providerFilter])
  const modelsPayload = allModels ? [] : selectedModels
  const canSave =
    Boolean(alias.trim()) && (allModels || selectedModels.length > 0)

  const openEditor = (next: KeyEditorState) => {
    setEditor(next)
    setAlias(next.key?.slug || "")
    setProviderFilter("all")
    setTierFilter("normal")
    setModelSearch("")
    if (next.key?.allModels ?? next.mode === "create") {
      setAllModels(true)
      setSelectedModels([])
    } else {
      setAllModels(false)
      setSelectedModels(next.key?.models || [])
    }
  }
  const toggleModel = (modelId: string, checked: boolean) => {
    if (allModels) {
      if (!checked) {
        setAllModels(false)
        setSelectedModels(
          keyPickerModels
            .map((model) => model.id)
            .filter((id) => id !== modelId)
        )
      }
      return
    }

    const next = checked
      ? selectedModels.includes(modelId)
        ? selectedModels
        : [...selectedModels, modelId]
      : selectedModels.filter((id) => id !== modelId)

    if (
      next.length === keyPickerModels.length &&
      keyPickerModels.length > 0
    ) {
      setAllModels(true)
      setSelectedModels([])
      return
    }

    setSelectedModels(next)
  }
  const selectVisibleModels = () => {
    const visibleIds = filteredModels.map((model) => model.id)
    if (visibleIds.length === 0) return
    if (visibleIds.length === keyPickerModels.length) {
      setAllModels(true)
      setSelectedModels([])
      return
    }
    setAllModels(false)
    setSelectedModels(visibleIds)
  }
  const save = async () => {
    if (!client || !editor || !canSave) return
    setSaving(true)
    try {
      if (editor.mode === "create") {
        const result = await client.createKey(alias.trim(), modelsPayload)
        await client.registerKey(alias.trim(), result.key)
        setSecret(result.key)
      } else if (editor.key) {
        await client.updateKey(editor.key.id, alias.trim(), modelsPayload)
        toast.add({ title: "Key updated", type: "success" })
      }
      setEditor(null)
      await reload()
    } catch (cause) {
      toast.add({
        title: cause instanceof Error ? cause.message : "Couldn't save key",
        type: "error",
      })
    } finally {
      setSaving(false)
    }
  }
  const performAction = async () => {
    if (!client || !confirm) return
    const action = confirm.action
    const successTitle =
      action === "delete"
        ? "Key deleted"
        : action === "block"
          ? "Key blocked"
          : "Key unblocked"
    const errorTitle =
      action === "delete"
        ? "Couldn't delete key"
        : action === "block"
          ? "Couldn't block key"
          : "Couldn't unblock key"
    try {
      await client.keyAction(confirm.key.id, action)
      toast.add({ title: successTitle, type: "success" })
      await reload()
    } catch (cause) {
      toast.add({
        title: cause instanceof Error ? cause.message : errorTitle,
        type: "error",
      })
    } finally {
      setConfirm(null)
    }
  }

  const newKeyButton = (
    <Button
      onClick={() => openEditor({ mode: "create" })}
      disabled={!capabilities?.supported}
    >
      <AppIcon icon={PlusSignIcon} dataIcon="inline-start" />
      New key
    </Button>
  )

  const keysContent = (
    <>
      {!embedded && (
        <PageHeading
          title="Keys"
          description="Manage Gateway keys stored securely in macOS Keychain."
          action={newKeyButton}
        />
      )}
      {embedded && (
        <div className="settings-section-toolbar">
          <h2 className="settings-section-heading">Keys</h2>
          {newKeyButton}
        </div>
      )}
      {loading && keys.length === 0 && !capabilities ? (
        <div className="skeleton-stack">
          {Array.from({ length: 5 }, (_, index) => (
            <Skeleton key={index} className="h-14" />
          ))}
        </div>
      ) : error ? (
        <EmptyState
          title="Keys unavailable"
          description={error}
          action={<Button onClick={reload}>Try again</Button>}
        />
      ) : !capabilities?.supported ? (
        <EmptyState
          title="Key management unavailable"
          description={
            capabilities?.reason || "A management-capable key is required."
          }
        />
      ) : keys.length === 0 ? (
        <EmptyState
          title="No keys yet"
          description="Create a key to launch tools with scoped access."
          action={
            <Button onClick={() => openEditor({ mode: "create" })}>
              Create key
            </Button>
          }
        />
      ) : (
        <Card>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Models</TableHead>
                  <TableHead>Spend</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell>
                      <div className="key-name">
                        <strong>{key.slug || key.alias}</strong>
                        <CardDescription>
                          {key.management
                            ? "Management key"
                            : key.default
                              ? "Default launch key"
                              : key.active
                                ? "Active credential"
                                : "Virtual key"}
                        </CardDescription>
                      </div>
                    </TableCell>
                    <TableCell>
                      {key.allModels
                        ? "All models"
                        : `${key.models?.length || 0} models`}
                    </TableCell>
                    <TableCell>{formatMoney(key.spend)}</TableCell>
                    <TableCell>
                      <Badge
                        variant={key.blocked ? "destructive" : "secondary"}
                      >
                        {key.blocked ? "Blocked" : "Active"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {!key.management && (
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            render={<Button variant="ghost" size="icon-sm" />}
                          >
                            <AppIcon icon={MoreHorizontalIcon} />
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuGroup>
                              <DropdownMenuItem
                                onClick={() =>
                                  openEditor({ mode: "edit", key })
                                }
                              >
                                Edit
                              </DropdownMenuItem>
                              {!key.default && (
                                <DropdownMenuItem
                                  onClick={async () => {
                                    if (!client) return
                                    try {
                                      await client.keyAction(key.id, "default")
                                      toast.add({
                                        title: "Default launch key updated",
                                        type: "success",
                                      })
                                      await reload()
                                    } catch (cause) {
                                      toast.add({
                                        title:
                                          cause instanceof Error
                                            ? cause.message
                                            : "Couldn't update default launch key",
                                        type: "error",
                                      })
                                    }
                                  }}
                                >
                                  Make default
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem
                                onClick={() =>
                                  setConfirm({
                                    key,
                                    action: key.blocked ? "unblock" : "block",
                                  })
                                }
                              >
                                {key.blocked ? "Unblock" : "Block"}
                              </DropdownMenuItem>
                            </DropdownMenuGroup>
                            <DropdownMenuSeparator />
                            <DropdownMenuGroup>
                              <DropdownMenuItem
                                variant="destructive"
                                onClick={() =>
                                  setConfirm({ key, action: "delete" })
                                }
                              >
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuGroup>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Sheet
        open={Boolean(editor)}
        onOpenChange={(open) => !open && setEditor(null)}
      >
        <SheetContent
          side="right"
          className="key-editor-sheet w-full sm:w-[36rem] sm:max-w-none"
        >
          <SheetHeader>
            <SheetTitle>
              {editor?.mode === "create" ? "Create key" : "Edit key"}
            </SheetTitle>
            <SheetDescription>
              Choose a memorable lowercase name and optionally limit model
              access.
            </SheetDescription>
          </SheetHeader>
          <div className="key-editor-body">
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="key-name">Key name</FieldLabel>
                <Input
                  id="key-name"
                  value={alias}
                  onChange={(event) =>
                    setAlias(
                      event.target.value
                        .toLowerCase()
                        .replace(/[^a-z0-9-]/g, "-")
                    )
                  }
                  placeholder="my-project"
                />
              </Field>
              <FieldSet className="key-editor-models">
                <div className="key-editor-models-header">
                  <FieldLegend variant="label" className="mb-0">
                    Models
                  </FieldLegend>
                  <div className="key-editor-models-actions">
                    <Button
                      type="button"
                      variant="link"
                      size="xs"
                      className="h-auto px-0"
                      onClick={selectVisibleModels}
                      disabled={filteredModels.length === 0}
                    >
                      Select visible
                    </Button>
                    <Button
                      type="button"
                      variant="link"
                      size="xs"
                      className="h-auto px-0"
                      onClick={() => {
                        setAllModels(false)
                        setSelectedModels([])
                      }}
                    >
                      Clear
                    </Button>
                  </div>
                </div>
                <div className="key-editor-filters">
                  <ToggleGroup
                    className="key-editor-provider-filter"
                    variant="outline"
                    size="sm"
                    spacing={0}
                    value={[providerFilter]}
                    onValueChange={(value) => {
                      const next = value[0]
                      if (!next) return
                      setProviderFilter(next)
                    }}
                    aria-label="Provider"
                  >
                    <ToggleGroupItem value="all" aria-label="All providers">
                      All
                    </ToggleGroupItem>
                    {providerOptions.map((provider) => (
                      <ToggleGroupItem
                        key={provider}
                        value={provider}
                        aria-label={formatProviderLabel(provider)}
                      >
                        {formatProviderLabel(provider)}
                      </ToggleGroupItem>
                    ))}
                  </ToggleGroup>
                  <ToggleGroup
                    className="key-editor-tier-filter"
                    variant="outline"
                    size="sm"
                    spacing={0}
                    value={[tierFilter]}
                    onValueChange={(value) => {
                      const next = value[0]
                      if (next === "normal" || next === "1m") setTierFilter(next)
                    }}
                    aria-label="Model tier"
                  >
                    <ToggleGroupItem value="normal" aria-label="Default models">
                      Default
                    </ToggleGroupItem>
                    <ToggleGroupItem value="1m" aria-label="1m tier variants">
                      [1m]
                    </ToggleGroupItem>
                  </ToggleGroup>
                </div>
                <Input
                  value={modelSearch}
                  onChange={(event) => setModelSearch(event.target.value)}
                  placeholder="Filter models…"
                  aria-label="Filter models"
                />
                <ScrollArea className="key-editor-model-scroll">
                  <FieldGroup className="model-picker-list">
                    <Field orientation="horizontal">
                      <Checkbox
                        id="all-models"
                        checked={allModels}
                        onCheckedChange={(checked) => {
                          setAllModels(checked === true)
                          if (checked === true) setSelectedModels([])
                        }}
                      />
                      <FieldLabel htmlFor="all-models">All models</FieldLabel>
                    </Field>
                    {providerGroups.length > 0 ? (
                      providerGroups.map(({ provider, models: groupModels }) => (
                        <div
                          key={provider}
                          className="key-editor-provider-group"
                        >
                          {providerFilter === "all" ? (
                            <div className="key-editor-provider-heading">
                              {formatProviderLabel(provider)}
                            </div>
                          ) : null}
                          {groupModels.map((model) => {
                            const checkboxId = `model-${model.id}`
                            const checked =
                              allModels || selectedModels.includes(model.id)
                            return (
                              <Field key={model.id} orientation="horizontal">
                                <Checkbox
                                  id={checkboxId}
                                  checked={checked}
                                  onCheckedChange={(nextChecked) =>
                                    toggleModel(model.id, nextChecked === true)
                                  }
                                />
                                <FieldLabel htmlFor={checkboxId}>
                                  {model.id}
                                </FieldLabel>
                              </Field>
                            )
                          })}
                        </div>
                      ))
                    ) : (
                      <p className="key-editor-empty">
                        No models match the current filters.
                      </p>
                    )}
                  </FieldGroup>
                </ScrollArea>
              </FieldSet>
            </FieldGroup>
          </div>
          <SheetFooter className="key-editor-footer flex-row justify-end gap-2">
            <Button variant="outline" onClick={() => setEditor(null)}>
              Cancel
            </Button>
            <Button onClick={save} disabled={saving || !canSave}>
              {saving ? "Saving…" : "Save key"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Dialog
        open={Boolean(secret)}
        onOpenChange={(open) => !open && setSecret("")}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Key created</DialogTitle>
            <DialogDescription>
              This secret is shown once. HarnezPad has already stored it securely in
              Keychain.
            </DialogDescription>
          </DialogHeader>
          <Button
            variant="secondary"
            className="clipboard-row"
            onClick={() => copyText(secret)}
          >
            <code className="truncate text-left">{secret}</code>
            <AppIcon icon={Copy01Icon} dataIcon="inline-end" />
          </Button>
          <DialogFooter>
            <Button onClick={() => setSecret("")}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(confirm)}
        onOpenChange={(open) => !open && setConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {confirm?.action === "delete"
                ? "Delete this key?"
                : `${confirm?.action === "block" ? "Block" : "Unblock"} this key?`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {confirm?.action === "delete"
                ? "This action is permanent and applications using this key will stop working."
                : "You can change this state again later."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={performAction}>
              {confirm?.action === "delete" ? "Delete" : "Continue"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )

  if (embedded) {
    return keysContent
  }

  return (
    <ScrollArea className="page-scroll">
      <div className="page-wide page-scroll-inner">{keysContent}</div>
    </ScrollArea>
  )
}

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react"

import { EmptyState } from "@/components/layout/EmptyState"
import { AppSidebar } from "@/components/layout/AppSidebar"
import { LaunchShell } from "@/components/launch/LaunchShell"
import { SettingsPage } from "@/components/settings/SettingsPage"
import { SetupDialog } from "@/components/setup/SetupDialog"
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
import { Button } from "@/components/ui/button"
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { toast } from "@/components/ui/toast"
import { HelperClient } from "@/lib/helper"
import { type LaunchTabId } from "@/lib/nav-config"
import { useRouteHistory } from "@/lib/route-history"
import {
  nativeBridge,
  onNativeEvent,
  onNativeNavigation,
} from "@/lib/native"
import {
  clampDragSidebarWidth,
  clampExpandedSidebarWidth,
  persistSidebarWidth,
  readStoredSidebarWidth,
  shouldCollapseSidebarWidth,
} from "@/lib/sidebar-width"
import { computeTitlebarDragRegions } from "@/lib/titlebar-drag"
import {
  confirmUpdateInstall,
  formatUpdateError,
  latestBuildMessage,
  showUpdateAlert,
  showUpdateError,
  updateReady,
} from "@/lib/update-utils"
import { cn } from "@/lib/utils"
import type {
  AccountSummary,
  KeyCapabilities,
  KeySummary,
  ModelCatalogEntry,
  NativeBootstrap,
  Route,
  Settings,
  UpdateStatus,
} from "@/types"

const sleep = (milliseconds: number) =>
  new Promise((resolve) => window.setTimeout(resolve, milliseconds))

export default function App() {
  const [bootstrap, setBootstrap] = useState<NativeBootstrap | null>(null)
  const [connectError, setConnectError] = useState("")
  const {
    location,
    canGoBack,
    canGoForward,
    pushLocation,
    goBack,
    goForward,
  } = useRouteHistory()
  const route = location.route
  const launchTab = location.launchTab
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [sidebarWidth, setSidebarWidth] = useState(readStoredSidebarWidth)
  const [settings, setSettings] = useState<Settings | null>(null)
  const [account, setAccount] = useState<AccountSummary | null>(null)
  const [models, setModels] = useState<ModelCatalogEntry[]>([])
  const [keys, setKeys] = useState<KeySummary[]>([])
  const [keyCapabilities, setKeyCapabilities] =
    useState<KeyCapabilities | null>(null)
  const [loading, setLoading] = useState(true)
  const [keysError, setKeysError] = useState("")
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null)
  const [updateInstalling, setUpdateInstalling] = useState(false)
  const [updateConfirmVersion, setUpdateConfirmVersion] = useState<
    string | null
  >(null)
  const client = useMemo(
    () =>
      bootstrap?.ready && bootstrap.helperUrl && bootstrap.helperToken
        ? new HelperClient(bootstrap.helperUrl, bootstrap.helperToken)
        : null,
    [bootstrap]
  )

  useEffect(() => {
    let cancelled = false
    const connect = async () => {
      for (let attempt = 0; attempt < 60 && !cancelled; attempt += 1) {
        try {
          const next = await nativeBridge.bootstrap()
          if (next.ready) {
            setBootstrap(next)
            setSidebarOpen(next.sidebarOpen)
            document.documentElement.dataset.appearance = next.appearance
            return
          }
        } catch (cause) {
          if (attempt === 59)
            setConnectError(
              cause instanceof Error
                ? cause.message
                : "Native bridge unavailable"
            )
        }
        await sleep(200)
      }
      if (!cancelled) setConnectError("HarnezPad helper did not become ready")
    }
    connect()
    return () => {
      cancelled = true
    }
  }, [])

  const loadKeys = useCallback(async () => {
    if (!client) return
    setKeysError("")
    try {
      const [capabilities, page] = await Promise.all([
        client.keyCapabilities(),
        client.keys(),
      ])
      setKeyCapabilities(capabilities)
      setKeys(page.keys || [])
    } catch (cause) {
      setKeysError(
        cause instanceof Error ? cause.message : "Could not load keys"
      )
    }
  }, [client])

  const loadAll = useCallback(async () => {
    if (!client) return
    setLoading(true)
    setKeysError("")
    const [settingsResult, accountResult, modelsResult] =
      await Promise.allSettled([
        client.settings(),
        client.account(),
        client.models(),
      ])
    if (settingsResult.status === "fulfilled") setSettings(settingsResult.value)
    if (accountResult.status === "fulfilled") setAccount(accountResult.value)
    if (modelsResult.status === "fulfilled") setModels(modelsResult.value || [])
    await loadKeys()
    setLoading(false)
  }, [client, loadKeys])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  const refreshUpdateStatus = useCallback(async () => {
    if (!client) return
    try {
      setUpdateStatus(await client.updateStatus(false))
    } catch {
      // Keep the last known status when polling fails transiently.
    }
  }, [client])

  const checkForUpdates = useCallback(async () => {
    if (!client) {
      await showUpdateError("Update checking is not available yet.")
      return
    }
    try {
      const status = await client.updateStatus(true)
      setUpdateStatus(status)
      if (status.error) {
        await showUpdateError(formatUpdateError(status.error))
        return
      }
      if (updateReady(status)) {
        const version = status.update!.version
        const confirmed = await confirmUpdateInstall(version)
        if (confirmed === true) {
          setUpdateInstalling(true)
          try {
            await client.installUpdate()
            await nativeBridge.updateComplete()
          } catch (cause) {
            setUpdateInstalling(false)
            await showUpdateError(
              cause instanceof Error ? cause.message : "Could not install update"
            )
            void refreshUpdateStatus()
          }
          return
        }
        if (confirmed === null) {
          setUpdateConfirmVersion(version)
        }
        return
      }
      await showUpdateAlert("HarnezPad Updates", latestBuildMessage(status.currentVersion))
    } catch (cause) {
      await showUpdateError(
        formatUpdateError(
          cause instanceof Error ? cause.message : "Could not check for updates"
        )
      )
    }
  }, [client, refreshUpdateStatus])

  const installPendingUpdate = useCallback(async () => {
    if (!client || updateInstalling) return
    setUpdateInstalling(true)
    try {
      await client.installUpdate()
      await nativeBridge.updateComplete()
    } catch (cause) {
      setUpdateInstalling(false)
      toast.add({
        title:
          cause instanceof Error ? cause.message : "Could not install update",
        type: "error",
      })
      void refreshUpdateStatus()
    }
  }, [client, updateInstalling, refreshUpdateStatus])

  useEffect(() => {
    if (!client) return
    void refreshUpdateStatus()
    const timer = window.setInterval(() => void refreshUpdateStatus(), 10_000)
    return () => window.clearInterval(timer)
  }, [client, refreshUpdateStatus])

  useEffect(
    () =>
      onNativeNavigation((next) => {
        if (next === "updates") {
          void checkForUpdates()
          return
        }
        if (next === "settings" || next === "keys") {
          pushLocation({ route: next, launchTab: location.launchTab })
          return
        }
        if (next === "models") {
          pushLocation({ route: "agents", launchTab: location.launchTab })
          return
        }
        pushLocation({ route: next, launchTab: location.launchTab })
      }),
    [checkForUpdates, location.launchTab, pushLocation]
  )
  useEffect(
    () =>
      onNativeEvent<{ open: boolean }>("harnezpad:sidebar", ({ open }) =>
        setSidebarOpen(open)
      ),
    []
  )
  useEffect(
    () =>
      onNativeEvent<{ appearance: "light" | "dark" }>(
        "harnezpad:appearance",
        ({ appearance }) => {
          document.documentElement.dataset.appearance = appearance
        }
      ),
    []
  )
  useLayoutEffect(() => {
    if (!bootstrap?.ready) return

    const syncTitlebarDragRegions = () => {
      const regions = computeTitlebarDragRegions({
        sidebarOpen,
        sidebarWidthPx: sidebarWidth,
        route,
      })
      nativeBridge.setTitlebarDragRegions(regions).catch(() => undefined)
    }

    syncTitlebarDragRegions()
    window.addEventListener("resize", syncTitlebarDragRegions)

    const shell = document.querySelector(".app-shell")
    const shellObserver =
      shell instanceof Element ? new ResizeObserver(syncTitlebarDragRegions) : null
    shellObserver?.observe(shell as Element)

    const observed = [
      document.querySelector(".titlebar-spacer"),
      document.querySelector(".launch-titlebar-tabs"),
      document.querySelector(".titlebar-spend-meta"),
    ].filter((node): node is Element => node instanceof Element)

    const observer =
      observed.length > 0 ? new ResizeObserver(syncTitlebarDragRegions) : null
    for (const node of observed) observer?.observe(node)

    return () => {
      window.removeEventListener("resize", syncTitlebarDragRegions)
      shellObserver?.disconnect()
      observer?.disconnect()
    }
  }, [bootstrap?.ready, sidebarOpen, sidebarWidth, route, account, loading])

  const requestSidebarOpen = useCallback(
    (open: boolean) => {
      if (open === sidebarOpen) return
      nativeBridge.toggleSidebar().catch((cause) =>
        toast.add({
          title:
            cause instanceof Error ? cause.message : "Couldn't toggle sidebar",
          type: "error",
        })
      )
    },
    [sidebarOpen]
  )

  const finishSidebarResize = useCallback(
    (handle: HTMLDivElement, pointerId: number) => {
      handle.releasePointerCapture(pointerId)
      document.body.classList.remove("sidebar-resizing")
    },
    []
  )

  const onSidebarResizePointerDown = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (!sidebarOpen) return
      event.preventDefault()
      const handle = event.currentTarget
      const startX = event.clientX
      const startWidth = sidebarWidth
      handle.setPointerCapture(event.pointerId)
      document.body.classList.add("sidebar-resizing")

      const cleanup = (upEvent: PointerEvent) => {
        handle.removeEventListener("pointermove", onPointerMove)
        handle.removeEventListener("pointerup", onPointerUp)
        handle.removeEventListener("pointercancel", onPointerUp)
        finishSidebarResize(handle, upEvent.pointerId)
      }

      const onPointerMove = (moveEvent: PointerEvent) => {
        const next = clampDragSidebarWidth(
          startWidth + (moveEvent.clientX - startX)
        )
        if (shouldCollapseSidebarWidth(next)) {
          cleanup(moveEvent)
          setSidebarOpen(false)
          requestSidebarOpen(false)
          return
        }
        setSidebarWidth(next)
      }

      const onPointerUp = (upEvent: PointerEvent) => {
        cleanup(upEvent)
        setSidebarWidth((current) => {
          if (shouldCollapseSidebarWidth(current)) {
            setSidebarOpen(false)
            requestSidebarOpen(false)
            return startWidth
          }
          const final = clampExpandedSidebarWidth(current)
          persistSidebarWidth(final)
          return final
        })
      }

      handle.addEventListener("pointermove", onPointerMove)
      handle.addEventListener("pointerup", onPointerUp)
      handle.addEventListener("pointercancel", onPointerUp)
    },
    [sidebarOpen, sidebarWidth, requestSidebarOpen, finishSidebarResize]
  )

  const pushRoute = useCallback(
    (next: Route, tab: LaunchTabId = location.launchTab) => {
      pushLocation({ route: next, launchTab: tab })
    },
    [location.launchTab, pushLocation]
  )

  const enterSettings = useCallback(() => {
    pushRoute("settings")
  }, [pushRoute])
  const navigate = useCallback(
    (next: Route) => {
      pushRoute(next)
    },
    [pushRoute]
  )
  const leaveSettings = useCallback(() => {
    if (canGoBack) goBack()
    else pushRoute("agents")
  }, [canGoBack, goBack, pushRoute])
  const settingsMode = route === "settings" || route === "keys"

  if (connectError)
    return (
      <main className="fatal-state">
        <EmptyState
          title="HarnezPad could not start"
          description={connectError}
          action={
            <Button onClick={() => window.location.reload()}>Try again</Button>
          }
        />
      </main>
    )

  return (
    <SidebarProvider
      className="app-shell"
      open={sidebarOpen}
      onOpenChange={requestSidebarOpen}
      style={
        {
          "--sidebar-width": `${sidebarWidth}px`,
        } as React.CSSProperties
      }
    >
      <AppSidebar
        sidebarOpen={sidebarOpen}
        sidebarWidth={sidebarWidth}
        settingsMode={settingsMode}
        route={route}
        updateStatus={updateStatus}
        updateInstalling={updateInstalling}
        onNavigate={navigate}
        onEnterSettings={enterSettings}
        onLeaveSettings={leaveSettings}
        onInstallUpdate={() => void installPendingUpdate()}
        onSidebarResizePointerDown={onSidebarResizePointerDown}
        canGoBack={canGoBack}
        canGoForward={canGoForward}
        onHistoryBack={goBack}
        onHistoryForward={goForward}
      />
      <SidebarInset
        className={cn(
          "main-panel grid",
          settingsMode && "main-panel-settings",
        )}
      >
        {route === "agents" && (
          <LaunchShell
            launchTab={launchTab}
            onLaunchTabChange={(tab) => pushRoute("agents", tab)}
            account={account}
            loading={loading}
          />
        )}
        {route !== "agents" && (
          <div className="content-panel">
            {(route === "settings" || route === "keys") && (
              <SettingsPage
                tab={route === "keys" ? "gateway" : "general"}
                client={client}
                settings={settings}
                account={account}
                models={models}
                keys={keys}
                keyCapabilities={keyCapabilities}
                keysLoading={loading}
                keysError={keysError}
                reload={loadAll}
                reloadKeys={loadKeys}
              />
            )}
          </div>
        )}
      </SidebarInset>
      <AlertDialog
        open={updateConfirmVersion !== null}
        onOpenChange={(open) => {
          if (!open) setUpdateConfirmVersion(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>HarnezPad Updates</AlertDialogTitle>
            <AlertDialogDescription>
              {updateConfirmVersion
                ? `HarnezPad ${updateConfirmVersion} is ready to install. Restart now?`
                : null}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Not Now</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setUpdateConfirmVersion(null)
                void installPendingUpdate()
              }}
            >
              Restart
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <SetupDialog client={client} settings={settings} onComplete={loadAll} />
    </SidebarProvider>
  )
}

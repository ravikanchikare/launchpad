import { Globe02Icon, Settings02Icon } from "@hugeicons/core-free-icons"
import type { IconSvgElement } from "@hugeicons/react"
import { Send } from "lucide-react"

import { launchApps, type LaunchAppId } from "@/lib/verified-launch-commands"
import type { Route } from "@/types"

export type LaunchTabId = "agents" | LaunchAppId
export type SettingsTab = "general" | "gateway"

export const navItems: Array<{
  route: Route
  label: string
  icon?: IconSvgElement
  iconNode?: React.ReactNode
  bottom?: boolean
}> = [
  {
    route: "agents",
    label: "Agents",
    iconNode: <Send className="size-4" aria-hidden="true" />,
  },
  { route: "settings", label: "Settings", icon: Settings02Icon, bottom: true },
]

export const settingsNavItems: Array<{
  route: Extract<Route, "settings" | "keys">
  label: string
  icon: IconSvgElement
}> = [
  { route: "settings", label: "General", icon: Settings02Icon },
  { route: "keys", label: "Gateway", icon: Globe02Icon },
]

export const launchTitlebarTabs: Array<{ id: LaunchTabId; label: string }> = [
  { id: "agents", label: "Agents" },
  ...launchApps.map((app) => ({ id: app.id, label: app.name })),
]

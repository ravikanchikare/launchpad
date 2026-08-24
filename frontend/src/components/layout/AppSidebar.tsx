import {
  ArrowLeft02Icon,
  Settings02Icon,
} from "@hugeicons/core-free-icons"
import type { PointerEvent as ReactPointerEvent } from "react"

import { AppIcon } from "@/components/AppIcon"
import { NavItem } from "@/components/layout/NavItem"
import { SidebarHistoryNav } from "@/components/layout/SidebarHistoryNav"
import { SidebarUpdateButton } from "@/components/layout/SidebarUpdateButton"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { navItems, settingsNavItems } from "@/lib/nav-config"
import {
  SIDEBAR_MAX_WIDTH,
  SIDEBAR_RESIZE_MIN_WIDTH,
} from "@/lib/sidebar-width"
import type { Route, UpdateStatus } from "@/types"

export function AppSidebar({
  sidebarOpen,
  sidebarWidth,
  settingsMode,
  route,
  updateStatus,
  updateInstalling,
  onNavigate,
  onEnterSettings,
  onLeaveSettings,
  onInstallUpdate,
  onSidebarResizePointerDown,
  canGoBack,
  canGoForward,
  onHistoryBack,
  onHistoryForward,
}: {
  sidebarOpen: boolean
  sidebarWidth: number
  settingsMode: boolean
  route: Route
  updateStatus: UpdateStatus | null
  updateInstalling: boolean
  onNavigate: (route: Route) => void
  onEnterSettings: () => void
  onLeaveSettings: () => void
  onInstallUpdate: () => void
  onSidebarResizePointerDown: (event: ReactPointerEvent<HTMLDivElement>) => void
  canGoBack: boolean
  canGoForward: boolean
  onHistoryBack: () => void
  onHistoryForward: () => void
}) {
  return (
    <Sidebar
      variant="sidebar"
      collapsible="offcanvas"
      aria-label="HarnezPad sidebar"
    >
      <SidebarHeader className="app-sidebar-header !flex-row !items-center !gap-0 !p-0">
        <div className="sidebar-titlebar-space" aria-hidden="true" />
        <div className="sidebar-titlebar-drag" aria-hidden="true" />
        <SidebarHistoryNav
          canGoBack={canGoBack}
          canGoNext={canGoForward}
          onBack={onHistoryBack}
          onNext={onHistoryForward}
        />
      </SidebarHeader>
      <SidebarContent className="app-sidebar-content overflow-hidden!">
        {settingsMode ? (
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton onClick={onLeaveSettings}>
                    <AppIcon icon={ArrowLeft02Icon} />
                    <span>Back to app</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                {settingsNavItems.map((item) => (
                  <SidebarMenuItem key={item.route}>
                    <SidebarMenuButton
                      isActive={route === item.route}
                      onClick={() => onNavigate(item.route)}
                      aria-current={route === item.route ? "page" : undefined}
                    >
                      <AppIcon icon={item.icon} />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ) : (
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems
                  .filter((item) => !item.bottom)
                  .map((item) => (
                    <NavItem
                      key={item.route}
                      item={item}
                      active={route === item.route}
                      onPress={() => onNavigate(item.route)}
                    />
                  ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
        {!settingsMode && (
          <div className="app-sidebar-footer">
            <SidebarMenu>
              <SidebarMenuItem className="app-footer-actions">
                <SidebarUpdateButton
                  status={updateStatus}
                  installing={updateInstalling}
                  onInstall={onInstallUpdate}
                />
                <SidebarMenuButton
                  className="app-footer-settings"
                  onPointerDown={(event) => {
                    if (event.button === 0) onEnterSettings()
                  }}
                  onClick={() => onEnterSettings()}
                >
                  <AppIcon icon={Settings02Icon} />
                  <span>Settings</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </div>
        )}
      </SidebarContent>
      {sidebarOpen && (
        <div
          className="sidebar-resize-handle"
          role="separator"
          aria-orientation="vertical"
          aria-valuenow={sidebarWidth}
          aria-valuemin={SIDEBAR_RESIZE_MIN_WIDTH}
          aria-valuemax={SIDEBAR_MAX_WIDTH}
          onPointerDown={onSidebarResizePointerDown}
        />
      )}
    </Sidebar>
  )
}

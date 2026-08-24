import { UsageDonut, UsageDonutSkeleton } from "@/components/UsageDonut"
import { AgentsPlaceholder } from "@/components/launch/AgentsPlaceholder"
import { LaunchPage } from "@/components/launch/LaunchPage"
import { Button } from "@/components/ui/button"
import { launchTitlebarTabs, type LaunchTabId } from "@/lib/nav-config"
import { cn } from "@/lib/utils"
import type { AccountSummary } from "@/types"

export function LaunchShell({
  launchTab,
  onLaunchTabChange,
  account,
  loading,
}: {
  launchTab: LaunchTabId
  onLaunchTabChange: (tab: LaunchTabId) => void
  account: AccountSummary | null
  loading: boolean
}) {
  return (
    <div className="launch-shell">
      <header className="app-titlebar app-titlebar-launch">
        <div className="launch-titlebar-tabs">
          {launchTitlebarTabs.map((tab) => (
            <Button
              key={tab.id}
              type="button"
              variant="ghost"
              className={cn(
                "launch-titlebar-tab",
                launchTab === tab.id && "launch-titlebar-tab-active"
              )}
              aria-current={launchTab === tab.id ? "page" : undefined}
              onClick={() => onLaunchTabChange(tab.id)}
            >
              {tab.label}
            </Button>
          ))}
        </div>
        <div className="titlebar-spacer" />
        {account ? (
          <UsageDonut account={account} />
        ) : loading ? (
          <UsageDonutSkeleton />
        ) : null}
      </header>
      <div className="content-panel launch-content-panel">
        {launchTab === "agents" ? (
          <AgentsPlaceholder />
        ) : (
          <LaunchPage tab={launchTab} />
        )}
      </div>
    </div>
  )
}

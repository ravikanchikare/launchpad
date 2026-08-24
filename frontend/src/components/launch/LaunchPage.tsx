import { useState } from "react"

import { LaunchAppContent } from "@/components/launch/LaunchAppContent"
import { LaunchAppHeader } from "@/components/launch/LaunchAppHeader"
import { launchApps, type LaunchAppId, type ModelTierFilter } from "@/lib/verified-launch-commands"

export function LaunchPage({ tab }: { tab: LaunchAppId }) {
  const activeApp = launchApps.find((app) => app.id === tab) ?? launchApps[0]
  const [tierFilter, setTierFilter] = useState<ModelTierFilter>("normal")

  return (
    <div className="launch-page-layout">
      <LaunchAppHeader
        app={activeApp}
        tierFilter={tierFilter}
        onTierFilterChange={setTierFilter}
      />
      <LaunchAppContent app={activeApp} tierFilter={tierFilter} />
    </div>
  )
}

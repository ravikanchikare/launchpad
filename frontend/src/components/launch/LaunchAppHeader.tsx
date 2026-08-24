import {
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { launchApps, type ModelTierFilter } from "@/lib/verified-launch-commands"

export function LaunchAppHeader({
  app,
  tierFilter,
  onTierFilterChange,
}: {
  app: (typeof launchApps)[number]
  tierFilter: ModelTierFilter
  onTierFilterChange: (tier: ModelTierFilter) => void
}) {
  return (
    <div className="launch-app-header">
      <img
        className="application-artwork"
        data-app={app.id}
        src={app.artwork}
        alt=""
      />
      <div className="launch-app-copy">
        <CardTitle>{app.name}</CardTitle>
        <CardDescription>{app.description}</CardDescription>
      </div>
      <ToggleGroup
        className="launch-tier-filter"
        variant="outline"
        size="sm"
        spacing={0}
        value={[tierFilter]}
        onValueChange={(value) => {
          const next = value[0]
          if (next === "normal" || next === "1m") onTierFilterChange(next)
        }}
        aria-label="Model tier"
      >
        <ToggleGroupItem value="normal" aria-label="Normal models">
          Normal
        </ToggleGroupItem>
        <ToggleGroupItem value="1m" aria-label="1m tier variants">
          [1m]
        </ToggleGroupItem>
      </ToggleGroup>
    </div>
  )
}

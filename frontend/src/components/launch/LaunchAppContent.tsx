import { useMemo } from "react"

import { EmptyState } from "@/components/layout/EmptyState"
import { LaunchCommandButton } from "@/components/launch/LaunchCommandButton"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  filterModelsByTier,
  groupVerifiedModelsByProvider,
  launchCommandForModel,
  launchApps,
  type ModelTierFilter,
} from "@/lib/verified-launch-commands"

export function LaunchAppContent({
  app,
  tierFilter,
}: {
  app: (typeof launchApps)[number]
  tierFilter: ModelTierFilter
}) {
  const providerGroups = useMemo(() => {
    return groupVerifiedModelsByProvider(
      app.verifiedModels,
      app.providerOrder
    )
      .map(({ provider, models }) => ({
        provider,
        models: filterModelsByTier(models, tierFilter),
      }))
      .filter(({ models }) => models.length > 0)
  }, [app.providerOrder, app.verifiedModels, tierFilter])

  return (
    <div className="launch-app-content">
      <div className="launch-commands-area">
        {providerGroups.length > 0 ? (
          <div
            className="launch-provider-columns"
            style={
              {
                "--launch-provider-count": providerGroups.length,
              } as React.CSSProperties
            }
          >
            {providerGroups.map(({ provider, models }) => (
              <section
                key={provider}
                className="launch-provider-column"
                data-provider={provider.toLowerCase()}
              >
                <h3 className="launch-provider-heading">{provider}</h3>
                <ScrollArea className="launch-provider-commands">
                  <div className="launch-provider-commands-list">
                    {models.map((model) => {
                      const command = launchCommandForModel(app.command, model.id)
                      return (
                        <article key={model.id} className="launch-command-row">
                          <code className="launch-command-model">{model.id}</code>
                          <LaunchCommandButton command={command} label={model.id} />
                        </article>
                      )
                    })}
                  </div>
                </ScrollArea>
              </section>
            ))}
          </div>
        ) : (
          <EmptyState
            title="No verified commands"
            description="Compatibility probes have not confirmed any models for this app yet."
          />
        )}
      </div>

      <section className="launch-prerequisites">
        <h3 className="settings-section-heading">Prerequisites</h3>
        <ul className="launch-prerequisites-list">
          {app.prerequisites.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </section>
    </div>
  )
}

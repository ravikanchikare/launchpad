import {
  CardDescription,
  CardTitle,
} from "@/components/ui/card"

export function AgentsPlaceholder() {
  return (
    <div className="launch-page-layout">
      <div className="launch-app-header">
        <div className="launch-app-copy">
          <CardTitle>Agents</CardTitle>
          <CardDescription>
            Run and manage AI agents from HarnezPad. Agent workflows will appear
            here.
          </CardDescription>
        </div>
      </div>
    </div>
  )
}

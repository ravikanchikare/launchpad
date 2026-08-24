import { Copy01Icon } from "@hugeicons/core-free-icons"

import { AppIcon } from "@/components/AppIcon"
import { Button } from "@/components/ui/button"
import { copyText } from "@/lib/clipboard"

export function LaunchCommandButton({
  command,
  label,
}: {
  command: string
  label: string
}) {
  return (
    <Button
      variant="secondary"
      className="launch-command-button"
      onClick={() => copyText(command)}
      aria-label={`Copy ${label}`}
    >
      <code className="truncate text-left">{command}</code>
      <AppIcon icon={Copy01Icon} dataIcon="inline-end" />
    </Button>
  )
}

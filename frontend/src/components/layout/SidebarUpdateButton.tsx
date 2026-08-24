import { Download01Icon, Loading03Icon } from "@hugeicons/core-free-icons"
import { HugeiconsIcon } from "@hugeicons/react"

import { cn } from "@/lib/utils"
import { updateReady } from "@/lib/update-utils"
import type { UpdateStatus } from "@/types"

export function SidebarUpdateButton({
  status,
  installing,
  onInstall,
}: {
  status: UpdateStatus | null
  installing: boolean
  onInstall: () => void
}) {
  const ready = updateReady(status)
  if (!ready && !installing) return null
  const version = status?.update?.version
  return (
    <button
      type="button"
      className={cn(
        "sidebar-footer-icon-btn sidebar-update-btn",
        installing && "installing"
      )}
      disabled={installing}
      onPointerDown={(event) => {
        if (event.button === 0 && !installing) {
          event.stopPropagation()
          onInstall()
        }
      }}
      onClick={(event) => {
        event.stopPropagation()
        if (event.detail === 0 && !installing) onInstall()
      }}
      title={
        installing
          ? "Installing…"
          : version
            ? `Install HarnezPad ${version}`
            : "Install update"
      }
      aria-label={
        installing
          ? "Installing update"
          : version
            ? `Install HarnezPad ${version}`
            : "Install update"
      }
    >
      {installing ? (
        <>
          <HugeiconsIcon
            icon={Loading03Icon}
            strokeWidth={2}
            className="sidebar-update-spinner size-4"
            aria-hidden="true"
          />
          <span className="sidebar-update-label">Installing…</span>
        </>
      ) : (
        <>
          <HugeiconsIcon icon={Download01Icon} strokeWidth={2} className="size-4" aria-hidden="true" />
          <span>Update</span>
        </>
      )}
    </button>
  )
}

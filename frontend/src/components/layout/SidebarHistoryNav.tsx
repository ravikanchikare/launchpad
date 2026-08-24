import { ArrowLeft02Icon, ArrowRight02Icon } from "@hugeicons/core-free-icons"

import { AppIcon } from "@/components/AppIcon"
import { Button } from "@/components/ui/button"

export function SidebarHistoryNav({
  canGoBack,
  canGoNext,
  onBack,
  onNext,
}: {
  canGoBack: boolean
  canGoNext: boolean
  onBack: () => void
  onNext: () => void
}) {
  return (
    <div className="sidebar-history-nav">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="sidebar-history-btn"
        disabled={!canGoBack}
        onClick={onBack}
        aria-label="Back"
        title="Back"
      >
        <AppIcon icon={ArrowLeft02Icon} />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="sidebar-history-btn"
        disabled={!canGoNext}
        onClick={onNext}
        aria-label="Next"
        title="Next"
      >
        <AppIcon icon={ArrowRight02Icon} />
      </Button>
    </div>
  )
}

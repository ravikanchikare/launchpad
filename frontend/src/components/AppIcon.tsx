import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react"

export function AppIcon({
  icon,
  className,
  dataIcon,
}: {
  icon: IconSvgElement
  className?: string
  dataIcon?: "inline-start" | "inline-end"
}) {
  return (
    <HugeiconsIcon
      icon={icon}
      strokeWidth={1.8}
      className={className}
      data-icon={dataIcon}
      aria-hidden="true"
    />
  )
}

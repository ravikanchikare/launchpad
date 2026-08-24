import { AppIcon } from "@/components/AppIcon"
import {
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { navItems } from "@/lib/nav-config"

export function NavItem({
  item,
  active,
  onPress,
}: {
  item: (typeof navItems)[number]
  active: boolean
  onPress: () => void
}) {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={active}
        onClick={onPress}
        aria-current={active ? "page" : undefined}
      >
        {item.iconNode ?? (item.icon ? <AppIcon icon={item.icon} /> : null)}
        <span>{item.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

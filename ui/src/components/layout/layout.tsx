import type { ReactNode } from "react";

export function SidebarLayout({ title, sidebar, children }: { title?: string; sidebar: ReactNode; children: ReactNode }) {
  return (
    <div className="flex h-screen w-full bg-white text-neutral-950 overflow-hidden select-none">
      <aside className="flex w-[260px] shrink-0 flex-col border-r border-neutral-200 bg-white">
        <div className="h-10 shrink-0" /> {/* titlebar spacer */}
        {title && <div className="px-4 pb-2 text-[12px] font-medium uppercase tracking-wider text-neutral-400">{title}</div>}
        <div className="flex flex-1 flex-col min-h-0 overflow-y-auto">{sidebar}</div>
      </aside>
      <div className="flex flex-1 flex-col min-h-0 overflow-hidden bg-white">{children}</div>
    </div>
  );
}

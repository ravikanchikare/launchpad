export const SIDEBAR_WIDTH_KEY = "harnezpad.sidebar-width"
export const SIDEBAR_COLLAPSE_THRESHOLD_PX = 200
export const SIDEBAR_RESIZE_MIN_WIDTH = 240
export const SIDEBAR_MAX_WIDTH = 360
export const SIDEBAR_DEFAULT_WIDTH = 300

export function clampDragSidebarWidth(width: number) {
  return Math.min(
    SIDEBAR_MAX_WIDTH,
    Math.max(0, Math.round(width))
  )
}

export function clampExpandedSidebarWidth(width: number) {
  return Math.min(
    SIDEBAR_MAX_WIDTH,
    Math.max(SIDEBAR_RESIZE_MIN_WIDTH, Math.round(width))
  )
}

export function shouldCollapseSidebarWidth(width: number) {
  return width <= SIDEBAR_COLLAPSE_THRESHOLD_PX
}

export function readStoredSidebarWidth() {
  try {
    const raw = localStorage.getItem(SIDEBAR_WIDTH_KEY)
    if (!raw) return SIDEBAR_DEFAULT_WIDTH
    const parsed = Number(raw)
    if (!Number.isFinite(parsed)) return SIDEBAR_DEFAULT_WIDTH
    return clampExpandedSidebarWidth(parsed)
  } catch {
    return SIDEBAR_DEFAULT_WIDTH
  }
}

export function persistSidebarWidth(width: number) {
  try {
    localStorage.setItem(SIDEBAR_WIDTH_KEY, String(width))
  } catch {
    // Ignore quota / private-mode failures.
  }
}

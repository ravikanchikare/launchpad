export const SIDEBAR_TOGGLE_END_X = 122
export const SIDEBAR_HISTORY_NAV_WIDTH = 64
export const SIDEBAR_DRAG_START_X = SIDEBAR_TOGGLE_END_X
export const TITLEBAR_HEIGHT = 52
const MIN_DRAG_WIDTH = 4

export interface DragRegionFrame {
  x: number
  y: number
  width: number
  height: number
}

export interface TitlebarDragRegions {
  sidebar: DragRegionFrame
  content: DragRegionFrame
}

const hiddenRegion = (): DragRegionFrame => ({
  x: 0,
  y: 0,
  width: 0,
  height: 0,
})

export function computeTitlebarDragRegions(options: {
  sidebarOpen: boolean
  sidebarWidthPx: number
  route: string
}): TitlebarDragRegions {
  const dragEndX = options.sidebarWidthPx - SIDEBAR_HISTORY_NAV_WIDTH
  const sidebar =
    options.sidebarOpen && dragEndX > SIDEBAR_DRAG_START_X
      ? {
          x: SIDEBAR_DRAG_START_X,
          y: 0,
          width: dragEndX - SIDEBAR_DRAG_START_X,
          height: TITLEBAR_HEIGHT,
        }
      : hiddenRegion()

  if (options.route !== 'agents') {
    return { sidebar, content: hiddenRegion() }
  }

  const spacer = document.querySelector('.titlebar-spacer')
  if (!(spacer instanceof HTMLElement)) {
    return { sidebar, content: hiddenRegion() }
  }

  const rect = spacer.getBoundingClientRect()
  const content =
    rect.width >= MIN_DRAG_WIDTH
      ? {
          x: rect.x,
          y: rect.y,
          width: rect.width,
          height: rect.height,
        }
      : hiddenRegion()

  return { sidebar, content }
}

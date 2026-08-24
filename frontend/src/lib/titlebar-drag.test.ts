// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'

import {
  SIDEBAR_DRAG_START_X,
  SIDEBAR_HISTORY_NAV_WIDTH,
  TITLEBAR_HEIGHT,
  computeTitlebarDragRegions,
} from '@/lib/titlebar-drag'

describe('computeTitlebarDragRegions', () => {
  it('covers sidebar header space between toggle and history nav', () => {
    expect(
      computeTitlebarDragRegions({
        sidebarOpen: true,
        sidebarWidthPx: 300,
        route: 'settings',
      })
    ).toEqual({
      sidebar: {
        x: SIDEBAR_DRAG_START_X,
        y: 0,
        width: 300 - SIDEBAR_DRAG_START_X - SIDEBAR_HISTORY_NAV_WIDTH,
        height: TITLEBAR_HEIGHT,
      },
      content: { x: 0, y: 0, width: 0, height: 0 },
    })
  })

  it('hides sidebar drag when collapsed', () => {
    expect(
      computeTitlebarDragRegions({
        sidebarOpen: false,
        sidebarWidthPx: 300,
        route: 'agents',
      })
    ).toEqual({
      sidebar: { x: 0, y: 0, width: 0, height: 0 },
      content: { x: 0, y: 0, width: 0, height: 0 },
    })
  })

  it('maps the agents spacer to the content drag region', () => {
    document.body.innerHTML =
      '<header class="app-titlebar app-titlebar-launch"><div class="launch-titlebar-tabs"></div><div class="titlebar-spacer"></div><div class="titlebar-spend-meta"></div></header>'
    const spacer = document.querySelector('.titlebar-spacer') as HTMLElement
    spacer.getBoundingClientRect = () =>
      ({
        x: 360,
        y: 0,
        width: 420,
        height: 52,
        top: 0,
        left: 360,
        right: 780,
        bottom: 52,
        toJSON: () => ({}),
      }) as DOMRect

    expect(
      computeTitlebarDragRegions({
        sidebarOpen: false,
        sidebarWidthPx: 300,
        route: 'agents',
      })
    ).toEqual({
      sidebar: { x: 0, y: 0, width: 0, height: 0 },
      content: { x: 360, y: 0, width: 420, height: 52 },
    })
  })
})

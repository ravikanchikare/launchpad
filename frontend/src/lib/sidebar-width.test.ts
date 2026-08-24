import { describe, expect, it } from "vitest"

import {
  SIDEBAR_COLLAPSE_THRESHOLD_PX,
  SIDEBAR_DEFAULT_WIDTH,
  SIDEBAR_MAX_WIDTH,
  SIDEBAR_RESIZE_MIN_WIDTH,
  clampDragSidebarWidth,
  clampExpandedSidebarWidth,
  shouldCollapseSidebarWidth,
} from "@/lib/sidebar-width"

describe("sidebar-width", () => {
  it("allows drag width down to zero for collapse zone", () => {
    expect(clampDragSidebarWidth(50)).toBe(50)
    expect(clampDragSidebarWidth(0)).toBe(0)
    expect(clampDragSidebarWidth(-10)).toBe(0)
  })

  it("clamps expanded width to resize min and max", () => {
    expect(clampExpandedSidebarWidth(200)).toBe(SIDEBAR_RESIZE_MIN_WIDTH)
    expect(clampExpandedSidebarWidth(248)).toBe(248)
    expect(clampExpandedSidebarWidth(999)).toBe(SIDEBAR_MAX_WIDTH)
  })

  it("collapses when width is at or below threshold", () => {
    expect(shouldCollapseSidebarWidth(SIDEBAR_COLLAPSE_THRESHOLD_PX)).toBe(true)
    expect(shouldCollapseSidebarWidth(SIDEBAR_COLLAPSE_THRESHOLD_PX + 1)).toBe(
      false
    )
  })

  it("defaults expanded width within normal bounds", () => {
    expect(SIDEBAR_DEFAULT_WIDTH).toBeGreaterThanOrEqual(
      SIDEBAR_RESIZE_MIN_WIDTH
    )
    expect(SIDEBAR_DEFAULT_WIDTH).toBeLessThanOrEqual(SIDEBAR_MAX_WIDTH)
  })
})

import { describe, expect, it } from "vitest"

import {
  defaultAppLocation,
  locationsEqual,
  reduceRouteHistory,
  reduceRouteHistoryBack,
  reduceRouteHistoryForward,
} from "@/lib/route-history"

describe("route history", () => {
  it("treats launch tab changes on agents as distinct locations", () => {
    const initial = { entries: [defaultAppLocation()], index: 0 }
    const next = reduceRouteHistory(initial, {
      route: "agents",
      launchTab: "codex",
    })

    expect(next.entries).toHaveLength(2)
    expect(next.index).toBe(1)
  })

  it("ignores duplicate pushes", () => {
    const initial = { entries: [defaultAppLocation()], index: 0 }
    expect(reduceRouteHistory(initial, defaultAppLocation())).toBe(initial)
  })

  it("drops forward history when pushing a new location", () => {
    let state = { entries: [defaultAppLocation()], index: 0 }
    state = reduceRouteHistory(state, { route: "keys", launchTab: "agents" })
    state = reduceRouteHistoryBack(state)
    state = reduceRouteHistory(state, { route: "settings", launchTab: "agents" })

    expect(state.entries.map((entry) => entry.route)).toEqual([
      "agents",
      "settings",
    ])
    expect(state.index).toBe(1)
  })

  it("moves back and forward through entries", () => {
    let state = { entries: [defaultAppLocation()], index: 0 }
    state = reduceRouteHistory(state, { route: "keys", launchTab: "agents" })
    state = reduceRouteHistory(state, { route: "settings", launchTab: "agents" })
    state = reduceRouteHistoryBack(state)
    expect(state.entries[state.index].route).toBe("keys")

    state = reduceRouteHistoryForward(state)
    expect(state.entries[state.index].route).toBe("settings")
  })

  it("compares agent locations by launch tab", () => {
    expect(
      locationsEqual(
        { route: "agents", launchTab: "agents" },
        { route: "agents", launchTab: "codex" }
      )
    ).toBe(false)
    expect(
      locationsEqual(
        { route: "settings", launchTab: "agents" },
        { route: "settings", launchTab: "codex" }
      )
    ).toBe(true)
  })
})

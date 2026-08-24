import { useCallback, useMemo, useState } from "react"

import type { LaunchTabId } from "@/lib/nav-config"
import type { Route } from "@/types"

export interface AppLocation {
  route: Route
  launchTab: LaunchTabId
}

interface RouteHistoryState {
  entries: AppLocation[]
  index: number
}

export const defaultAppLocation = (): AppLocation => ({
  route: "agents",
  launchTab: "agents",
})

export function locationsEqual(a: AppLocation, b: AppLocation) {
  return (
    a.route === b.route &&
    (a.route !== "agents" || a.launchTab === b.launchTab)
  )
}

export function reduceRouteHistory(
  state: RouteHistoryState,
  next: AppLocation
): RouteHistoryState {
  const current = state.entries[state.index]
  if (current && locationsEqual(current, next)) return state
  return {
    entries: [...state.entries.slice(0, state.index + 1), next],
    index: state.index + 1,
  }
}

export function reduceRouteHistoryBack(
  state: RouteHistoryState
): RouteHistoryState {
  if (state.index <= 0) return state
  return { ...state, index: state.index - 1 }
}

export function reduceRouteHistoryForward(
  state: RouteHistoryState
): RouteHistoryState {
  if (state.index >= state.entries.length - 1) return state
  return { ...state, index: state.index + 1 }
}

export function useRouteHistory(initialLocation: AppLocation = defaultAppLocation()) {
  const [history, setHistory] = useState<RouteHistoryState>({
    entries: [initialLocation],
    index: 0,
  })

  const location = history.entries[history.index]
  const canGoBack = history.index > 0
  const canGoForward = history.index < history.entries.length - 1

  const pushLocation = useCallback((next: AppLocation) => {
    setHistory((state) => reduceRouteHistory(state, next))
  }, [])

  const goBack = useCallback(() => {
    setHistory(reduceRouteHistoryBack)
  }, [])

  const goForward = useCallback(() => {
    setHistory(reduceRouteHistoryForward)
  }, [])

  return useMemo(
    () => ({
      location,
      canGoBack,
      canGoForward,
      pushLocation,
      goBack,
      goForward,
    }),
    [location, canGoBack, canGoForward, pushLocation, goBack, goForward]
  )
}

import type { NativeBootstrap, NativeRoute } from '@/types'
import type { TitlebarDragRegions } from '@/lib/titlebar-drag'

type ZeroListener = (detail: unknown) => void

interface ZeroApi {
  invoke<T>(command: string, payload?: unknown): Promise<T>
  on?(name: string, listener: ZeroListener): (() => void) | void
}

declare global {
  interface Window {
    zero?: ZeroApi
  }
}

const invoke = async <T>(command: string, payload: unknown = {}): Promise<T> => {
  if (!window.zero) throw new Error('Native bridge is unavailable')
  return window.zero.invoke<T>(command, payload)
}

export const nativeBridge = {
  bootstrap: () => invoke<NativeBootstrap>('harnezpad.bootstrap'),
  toggleSidebar: () => invoke<{ sidebarOpen: boolean }>('harnezpad.sidebar.toggle'),
  quit: () => invoke('harnezpad.app.quit'),
  updateComplete: () => invoke('harnezpad.update.installComplete'),
  presentUpdateAlert: (title: string, message: string) =>
    invoke<{ accepted: boolean }>('harnezpad.update.alert', { title, message }),
  presentUpdateConfirm: (title: string, message: string) =>
    invoke<{ confirmed: boolean }>('harnezpad.update.confirm', { title, message }),
  setTitlebarDragRegions: (regions: TitlebarDragRegions) =>
    invoke<{ accepted: boolean }>('harnezpad.titlebar.setDragRegions', regions),
}

export const hasNativeUpdateDialogs = () => Boolean(window.zero)

export function onNativeEvent<T>(name: string, listener: (detail: T) => void) {
  const domName = `native-sdk:${name}`
  const domListener = (event: Event) => listener((event as CustomEvent<T>).detail)
  window.addEventListener(domName, domListener)
  return () => window.removeEventListener(domName, domListener)
}

export const onNativeNavigation = (listener: (route: NativeRoute) => void) =>
  onNativeEvent<{ route: NativeRoute }>('harnezpad:navigate', ({ route }) => listener(route))

// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'

import { nativeBridge, onNativeEvent } from '@/lib/native'

describe('native bridge', () => {
  afterEach(() => { delete window.zero })

  it('uses the typed HarnezPad bootstrap command', async () => {
    const invoke = vi.fn().mockResolvedValue({ ready: true, helperUrl: 'http://127.0.0.1:1', helperToken: 'token', sidebarOpen: true, appearance: 'light' })
    window.zero = { invoke }
    await expect(nativeBridge.bootstrap()).resolves.toMatchObject({ ready: true, sidebarOpen: true })
    expect(invoke).toHaveBeenCalledWith('harnezpad.bootstrap', {})
  })

  it('signals the native host after an update is installed', async () => {
    const invoke = vi.fn().mockResolvedValue(undefined)
    window.zero = { invoke }
    await nativeBridge.updateComplete()
    expect(invoke).toHaveBeenCalledWith('harnezpad.update.installComplete', {})
  })

  it('receives native window events', () => {
    const listener = vi.fn()
    const cleanup = onNativeEvent<{ open: boolean }>('harnezpad:sidebar', listener)
    window.dispatchEvent(new CustomEvent('native-sdk:harnezpad:sidebar', { detail: { open: false } }))
    expect(listener).toHaveBeenCalledWith({ open: false })
    cleanup()
  })
})

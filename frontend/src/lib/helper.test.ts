// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'

import { HelperClient } from '@/lib/helper'

describe('HelperClient', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('uses the ephemeral bearer token and decodes settings', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      gatewayUrl: 'https://gateway.example',
      tokenConfigured: true,
      tokenValid: true,
      defaultKeySlug: 'management-key',
    }), { status: 200, headers: { 'content-type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    const client = new HelperClient('http://127.0.0.1:49152', 'session-token')
    await expect(client.settings()).resolves.toMatchObject({ tokenValid: true })
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:49152/api/settings',
      expect.objectContaining({ headers: expect.objectContaining({ authorization: 'Bearer session-token' }) }),
    )
  })

  it('surfaces structured helper errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'management key is invalid' }), { status: 400 })))
    const client = new HelperClient('http://127.0.0.1:49152', 'session-token')
    await expect(client.validateToken('bad')).rejects.toThrow('management key is invalid')
  })
})

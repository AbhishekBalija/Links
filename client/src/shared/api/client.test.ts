import { afterEach, describe, expect, it, vi } from 'vitest'
import { REFRESH_LOCK_NAME, apiDownload, apiRequest, attemptRefresh, setAccessToken } from './client'

// In dev builds the client hangs helpers on window; Vitest runs in Node.
vi.hoisted(() => {
  globalThis.window = {} as Window & typeof globalThis
})

afterEach(() => vi.unstubAllGlobals())

describe('apiRequest', () => {
  it('resolves with nothing for a 204 No Content response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    await expect(apiRequest<void>('/api/v1/events/e1', { method: 'DELETE' })).resolves.toBeUndefined()
  })
})

describe('apiRequest with a file', () => {
  it('lets the browser set the multipart type for a FormData body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{"data":{"ok":true}}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const form = new FormData()
    form.append('file', new Blob(['email,full_name,usn\n']), 'class.csv')
    await apiRequest('/api/v1/admin/users/import', { method: 'POST', body: form })
    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>
    expect(headers['Content-Type']).toBeUndefined()
  })
})

describe('apiDownload', () => {
  it('returns the file with its name from the server', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response('full_name\nAsha', { status: 200, headers: { 'Content-Disposition': 'attachment; filename="event-participants.csv"' } }),
    )
    vi.stubGlobal('fetch', fetchMock)
    setAccessToken('token-1')
    const file = await apiDownload('/api/v1/events/e1/export', 'answers.csv')
    expect(file.name).toBe('event-participants.csv')
    expect(await file.blob.text()).toBe('full_name\nAsha')
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer token-1')
  })

  it('uses the name it was given when the server name is hidden', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('csv', { status: 200 })))
    setAccessToken('token-1')
    expect((await apiDownload('/api/v1/events/e1/export', 'guest-talk-answers.csv')).name).toBe('guest-talk-answers.csv')
  })

  it('refreshes an expired session once and tries again', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('{"error":{"code":"UNAUTHENTICATED","message":"expired"}}', { status: 401 }))
      .mockResolvedValueOnce(new Response('{"data":{"access_token":"token-2","expires_in":900}}', { status: 200 }))
      .mockResolvedValueOnce(new Response('csv', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    setAccessToken('token-1')
    const file = await apiDownload('/api/v1/events/e1/export', 'answers.csv')
    expect(await file.blob.text()).toBe('csv')
    expect(fetchMock.mock.calls[2][1].headers.Authorization).toBe('Bearer token-2')
  })

  it("throws the server's error", async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":{"code":"FORBIDDEN","message":"only organisers can export"}}', { status: 403 })))
    setAccessToken('token-1')
    await expect(apiDownload('/api/v1/events/e1/export', 'answers.csv')).rejects.toMatchObject({ status: 403, message: 'only organisers can export' })
  })
})

describe('attemptRefresh across tabs', () => {
  const refreshed = () => new Response('{"data":{"access_token":"token-2","expires_in":900}}', { status: 200 })

  it('refreshes while holding the shared lock, so other tabs wait their turn', async () => {
    let lockHeld = false
    let heldDuringFetch = false
    const request = vi.fn(async (_name: string, callback: () => Promise<string>) => {
      lockHeld = true
      try {
        return await callback()
      } finally {
        lockHeld = false
      }
    })
    vi.stubGlobal('navigator', { locks: { request } })
    vi.stubGlobal('fetch', vi.fn(async () => {
      heldDuringFetch = lockHeld
      return refreshed()
    }))

    await expect(attemptRefresh()).resolves.toBe('token-2')
    expect(request.mock.calls[0][0]).toBe(REFRESH_LOCK_NAME)
    expect(heldDuringFetch).toBe(true)
  })

  it('still refreshes in a browser without Web Locks', async () => {
    vi.stubGlobal('navigator', {})
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(refreshed()))
    await expect(attemptRefresh()).resolves.toBe('token-2')
  })
})

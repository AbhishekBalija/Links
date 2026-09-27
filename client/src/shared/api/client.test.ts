import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiRequest } from './client'

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

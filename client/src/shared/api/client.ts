import type { ApiSuccess, ApiError } from './types'
import { ApiRequestError } from './types'

const BASE_URL = import.meta.env.VITE_API_URL ?? ''

/** Safely parse JSON from a Response, throwing ApiRequestError on parse failure */
async function safeJson<R>(res: Response): Promise<R> {
  try {
    return await res.json() as R
  } catch {
    throw new ApiRequestError(res.status, {
      code: 'PARSE_ERROR',
      message: `Non-JSON response (${res.status})`,
    })
  }
}

let accessToken: string | null = null
let isRefreshing = false
let refreshQueue: Array<{
  resolve: (token: string) => void
  reject: (err: unknown) => void
}> = []

// Callback registration to break circular dependency with auth/store.
// store.ts calls registerLogoutHandler(clearAuth) once at init,
// so client.ts never needs to import the store directly.
let onLogout: (() => void) | null = null

export function registerLogoutHandler(handler: () => void) {
  onLogout = handler
}

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function getAccessToken(): string | null {
  return accessToken
}

export async function attemptRefresh(): Promise<string> {
  const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
  })

  if (!res.ok) {
    throw new Error('refresh_failed')
  }

  const body = (await res.json()) as ApiSuccess<{
    access_token: string
    expires_in: number
  }>

  if ('error' in body) {
    throw new Error('refresh_failed')
  }

  accessToken = body.data.access_token
  return accessToken
}

function processRefreshQueue(token: string) {
  for (const item of refreshQueue) {
    item.resolve(token)
  }
  refreshQueue = []
}

function failRefreshQueue(err: unknown) {
  for (const item of refreshQueue) {
    item.reject(err)
  }
  refreshQueue = []
}

// Dev-only: expose api internals for e2e silent-refresh testing.
// The test needs to trigger the 401-interceptor path (apiRequest → 401 →
// attemptRefresh → retry), which requires calling through the app's own
// client to hit the `accessToken` closure. No alternative via Playwright's
// request context or SPA navigation can exercise this specific code path.
declare global {
  interface Window {
    __getAccessToken: typeof getAccessToken
    __apiRequest: typeof apiRequest
  }
}
if (import.meta.env.DEV) {
  window.__getAccessToken = getAccessToken
  window.__apiRequest = apiRequest
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> | undefined),
  }

  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
    credentials: 'include',
  })

  // Silent refresh on 401: only if we had a token (skip unauthenticated requests like login)
  if (res.status === 401 && accessToken !== null) {
    if (!isRefreshing) {
      isRefreshing = true
      try {
        const newToken = await attemptRefresh()
        isRefreshing = false
        processRefreshQueue(newToken)

        // Retry the original request with the new token
        headers['Authorization'] = `Bearer ${newToken}`
        const retryRes = await fetch(`${BASE_URL}${path}`, {
          ...options,
          headers,
          credentials: 'include',
        })

        const retryBody = await safeJson<ApiSuccess<T> | ApiError>(retryRes)

        if (!retryRes.ok || 'error' in retryBody) {
          const errPayload = 'error' in retryBody
            ? retryBody.error
            : { code: 'UNKNOWN', message: 'Unknown error' }
          if (retryRes.status === 401) {
            accessToken = null
            onLogout?.()
          }
          throw new ApiRequestError(retryRes.status, errPayload)
        }

        return retryBody.data
      } catch (err) {
        isRefreshing = false
        failRefreshQueue(err)

        if (err instanceof ApiRequestError) {
          throw err
        }

        // Refresh failed — clear auth state
        accessToken = null

        if (onLogout) {
          onLogout()
        }

        throw new ApiRequestError(401, {
          code: 'UNAUTHENTICATED',
          message: 'Session expired. Please log in again.',
        })
      }
    }

    // Another refresh is in flight — queue this request
    return new Promise<T>((resolve, reject) => {
      refreshQueue.push({
        resolve: async (newToken: string) => {
          headers['Authorization'] = `Bearer ${newToken}`
          try {
            const retryRes = await fetch(`${BASE_URL}${path}`, {
              ...options,
              headers,
              credentials: 'include',
            })
            const retryBody = await safeJson<ApiSuccess<T> | ApiError>(retryRes)
            if (!retryRes.ok || 'error' in retryBody) {
              const errPayload = 'error' in retryBody
                ? retryBody.error
                : { code: 'UNKNOWN', message: 'Unknown error' }
              reject(new ApiRequestError(retryRes.status, errPayload))
              return
            }
            resolve(retryBody.data)
          } catch (err) {
            reject(err)
          }
        },
        reject,
      })
    })
  }

  const body = await safeJson<ApiSuccess<T> | ApiError>(res)

  if (!res.ok || 'error' in body) {
    const errPayload = 'error' in body
      ? body.error
      : { code: 'UNKNOWN', message: 'Unknown error' }
    throw new ApiRequestError(res.status, errPayload)
  }

  return body.data
}

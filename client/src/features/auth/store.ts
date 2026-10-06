import { create } from 'zustand'
import { setAccessToken, getAccessToken, attemptRefresh, registerLogoutHandler } from '../../shared/api/client'
import type { CurrentUser, FirstSignIn, SignInResponse } from './types'
import { fetchCurrentUser, logoutUser } from './api'
import { ApiRequestError } from '../../shared/api/types'

interface AuthState {
  user: CurrentUser | null
  isAuthenticated: boolean
  isLoading: boolean
  // firstSignIn is who a first sign-in signed in as, until they confirm it.
  firstSignIn: FirstSignIn | null
  // leftBecause is why the last session ended, for the sign-in screen to say:
  // signed out by hand, "Not you?", or the session ending on its own.
  leftBecause: 'signed-out' | 'reported' | 'expired' | null
  signedIn: (response: SignInResponse) => Promise<void>
  confirmFirstSignIn: () => void
  signedOutAfterReport: () => void
  forgetLeftBecause: () => void
  logout: () => Promise<void>
  clearAuth: () => void
  refreshUser: () => Promise<void>
}

// The first sign-in step is kept for this tab, so reloading "You're signed
// in as" doesn't lose it while "Not you?" still works (#212).
const firstSignInKey = 'links.firstSignIn'

function keepFirstSignIn(first: FirstSignIn | null) {
  try {
    if (first) sessionStorage.setItem(firstSignInKey, JSON.stringify(first))
    else sessionStorage.removeItem(firstSignInKey)
  } catch {
    // Nothing to keep it in: a reload goes to Home, as before.
  }
}

function keptFirstSignIn(): FirstSignIn | null {
  try {
    const kept = sessionStorage.getItem(firstSignInKey)
    return kept ? (JSON.parse(kept) as FirstSignIn) : null
  } catch {
    return null
  }
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  firstSignIn: null,
  leftBecause: null,
  signedIn: async (response: SignInResponse) => {
    setAccessToken(response.access_token)
    const user = await fetchCurrentUser()
    keepFirstSignIn(response.first_sign_in ?? null)
    set({ user, isAuthenticated: true, firstSignIn: response.first_sign_in ?? null, leftBecause: null })
  },
  confirmFirstSignIn: () => {
    keepFirstSignIn(null)
    set({ firstSignIn: null })
  },
  // "Not you?" has already signed the account out everywhere on the server.
  signedOutAfterReport: () => {
    setAccessToken(null)
    keepFirstSignIn(null)
    set({ user: null, isAuthenticated: false, firstSignIn: null, leftBecause: 'reported' })
  },
  forgetLeftBecause: () => set({ leftBecause: null }),
  logout: async () => {
    try {
      await logoutUser()
    } catch (error) {
      if (!(error instanceof ApiRequestError) || error.status !== 401) {
        throw error
      }
    }
    setAccessToken(null)
    keepFirstSignIn(null)
    set({ user: null, isAuthenticated: false, firstSignIn: null, leftBecause: 'signed-out' })
  },
  // clearAuth runs when the session can't be refreshed: it ended on its own
  // (a role ended, the account was paused, or days went by), so the sign-in
  // screen says so instead of appearing out of nowhere.
  clearAuth: () => {
    setAccessToken(null)
    keepFirstSignIn(null)
    set({ user: null, isAuthenticated: false, firstSignIn: null, leftBecause: get().isAuthenticated ? 'expired' : null })
  },
  refreshUser: async () => {
    const user = await fetchCurrentUser()
    set({ user })
  },
}))

// Register clearAuth with the API client so it can call it on refresh
// failure without importing this module (breaks circular dependency).
registerLogoutHandler(() => useAuthStore.getState().clearAuth())
export async function initializeAuth() {
  try {
    const token = getAccessToken() ?? await attemptRefresh().catch(() => null)
    if (!token) {
      useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
      return
    }
    const user = await fetchCurrentUser()
    useAuthStore.setState({ user, isAuthenticated: true, isLoading: false, firstSignIn: keptFirstSignIn() })
  } catch {
    useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
  }
}

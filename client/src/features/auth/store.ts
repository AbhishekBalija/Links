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
  // leftBecause is why the last session ended, for the sign-in screen to say.
  leftBecause: 'signed-out' | 'reported' | null
  signedIn: (response: SignInResponse) => Promise<void>
  confirmFirstSignIn: () => void
  signedOutAfterReport: () => void
  forgetLeftBecause: () => void
  logout: () => Promise<void>
  clearAuth: () => void
  refreshUser: () => Promise<void>
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  firstSignIn: null,
  leftBecause: null,
  signedIn: async (response: SignInResponse) => {
    setAccessToken(response.access_token)
    const user = await fetchCurrentUser()
    set({ user, isAuthenticated: true, firstSignIn: response.first_sign_in ?? null, leftBecause: null })
  },
  confirmFirstSignIn: () => set({ firstSignIn: null }),
  // "Not you?" has already signed the account out everywhere on the server.
  signedOutAfterReport: () => {
    setAccessToken(null)
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
    set({ user: null, isAuthenticated: false, firstSignIn: null, leftBecause: 'signed-out' })
  },
  clearAuth: () => {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false, firstSignIn: null })
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
    useAuthStore.setState({ user, isAuthenticated: true, isLoading: false })
  } catch {
    useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
  }
}

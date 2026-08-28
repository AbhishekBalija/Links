import { create } from 'zustand'
import { setAccessToken, getAccessToken, attemptRefresh, registerLogoutHandler } from '../../shared/api/client'
import type { CurrentUser, LoginInput } from './types'
import { loginUser, fetchCurrentUser, logoutUser } from './api'
import { ApiRequestError } from '../../shared/api/types'

interface AuthState {
  user: CurrentUser | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (input: LoginInput) => Promise<void>
  logout: () => Promise<void>
  clearAuth: () => void
  refreshUser: () => Promise<void>
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  login: async (input: LoginInput) => {
    const res = await loginUser(input)
    setAccessToken(res.access_token)
    const user = await fetchCurrentUser()
    set({ user, isAuthenticated: true })
  },
  logout: async () => {
    try {
      await logoutUser()
    } catch (error) {
      if (!(error instanceof ApiRequestError) || error.status !== 401) {
        throw error
      }
    }
    setAccessToken(null)
    set({ user: null, isAuthenticated: false })
  },
  clearAuth: () => {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false })
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

import { apiRequest } from '../../shared/api/client'
import type {
  LoginInput,
  LoginResponse,
  RequestAccessInput,
  RequestAccessResponse,
  CurrentUser,
} from './types'

export function loginUser(input: LoginInput) {
  return apiRequest<LoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function requestAccess(input: RequestAccessInput) {
  return apiRequest<RequestAccessResponse>('/api/v1/auth/request-access', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function fetchCurrentUser() {
  return apiRequest<CurrentUser>('/api/v1/me')
}

export function logoutUser() {
  return apiRequest<{ message: string }>('/api/v1/auth/logout', {
    method: 'POST',
  })
}

export function resendActivation(email: string) {
  return apiRequest<{ message: string }>('/api/v1/auth/resend-activation', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function refreshUserToken() {
  return apiRequest<{ access_token: string; expires_in: number }>('/api/v1/auth/refresh', {
    method: 'POST',
  })
}

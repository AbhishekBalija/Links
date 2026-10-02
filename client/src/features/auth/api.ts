import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { AccessRequestInput, CodeChallenge, CurrentUser, PublicDepartment, SignInResponse } from './types'

// The sign-in API (spec #129, ADR 0026): an email code or Google, with the
// class list deciding who gets in.

export function requestCode(email: string) {
  return apiRequest<CodeChallenge>('/api/v1/auth/code', {
    method: 'POST',
    body: JSON.stringify({ email: email.trim() }),
  })
}

export function verifyCode(input: { challengeId: string; email: string; code: string }) {
  return apiRequest<SignInResponse>('/api/v1/auth/code/verify', {
    method: 'POST',
    body: JSON.stringify({ challenge_id: input.challengeId, email: input.email.trim(), code: input.code }),
  })
}

// googleNonce also sets the httpOnly cookie the sign-in is checked against.
export function googleNonce() {
  return apiRequest<{ nonce: string }>('/api/v1/auth/google/nonce')
}

export function googleSignIn(credential: string) {
  return apiRequest<SignInResponse>('/api/v1/auth/google', {
    method: 'POST',
    body: JSON.stringify({ credential }),
  })
}

export function sendAccessRequest(input: AccessRequestInput) {
  return apiRequest<{ user_id: string; status: string }>('/api/v1/auth/access-request', {
    method: 'POST',
    body: JSON.stringify({ request_token: input.requestToken, usn: input.usn, full_name: input.fullName.trim() }),
  })
}

export function reportNotMe() {
  return apiRequest<{ message: string }>('/api/v1/auth/not-me', { method: 'POST' })
}

export function fetchCurrentUser() {
  return apiRequest<CurrentUser>('/api/v1/me')
}

export function logoutUser() {
  return apiRequest<{ message: string }>('/api/v1/auth/logout', {
    method: 'POST',
  })
}

// usePublicDepartments loads the Department list for reading a USN on the
// request form, which is used before sign-in.
export function usePublicDepartments() {
  return useQuery({
    queryKey: ['public-departments'],
    staleTime: 5 * 60_000,
    queryFn: ({ signal }) => apiRequest<PublicDepartment[]>('/api/v1/public/departments', { signal }),
  })
}

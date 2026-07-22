import { apiRequest } from '../../shared/api/client'
import type { UpdateProfileInput } from '../auth/types'

export type ProfileResponse = {
  username: string
  full_name: string
  headline: string | null
  bio: string | null
  avatar_url: string | null
  public_profile_enabled: boolean
  show_email: boolean
  show_phone: boolean
  linkedin_url: string | null
  github_url: string | null
  portfolio_url: string | null
}

export function updateMyProfile(input: UpdateProfileInput) {
  return apiRequest<ProfileResponse>('/api/v1/me/profile', {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
}


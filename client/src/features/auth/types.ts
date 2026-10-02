export type Profile = {
  full_name: string | null
  username: string | null
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

// CurrentUser is GET /api/v1/me.
export type CurrentUser = {
  user_id: string
  email: string
  roles: string[]
  profile: Profile
}

export type CodeChallenge = {
  challenge_id: string
  message: string
}

// FirstSignIn says who a first sign-in signed in as, for "Not you?".
export type FirstSignIn = {
  full_name: string
  email: string
  usn?: string
  department_code?: string
  department_name?: string
  batch_year?: number
  roles: string[]
}

export type SignInResponse = {
  access_token: string
  expires_in: number
  first_sign_in?: FirstSignIn
}

export type AccessRequestInput = {
  requestToken: string
  usn: string
  fullName: string
}

export type UpdateProfileInput = {
  headline?: string | null
  bio?: string | null
  avatar_url?: string | null
  show_email?: boolean
  show_phone?: boolean
  linkedin_url?: string | null
  github_url?: string | null
  portfolio_url?: string | null
}

export type PublicDepartment = { code: string; name: string }

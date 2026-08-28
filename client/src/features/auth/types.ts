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

export type CurrentUser = {
  id: string
  email: string
  roles: string[]
  profile: Profile
}

export type LoginInput = {
  email: string
  password: string
}

export type LoginResponse = {
  access_token: string
  expires_in: number
}

export type RequestAccessInput = {
  email: string
  password: string
  full_name: string
  usn?: string
  department_code?: string
}

export type RequestAccessResponse = {
  user_id: string
  status: string
}

export type ActivateInput = {
  token: string
  password: string
}

export type AuthState = {
  user: CurrentUser | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (input: LoginInput) => Promise<void>
  logout: () => Promise<void>
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

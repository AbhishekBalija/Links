export type DepartmentRef = { code: string; name: string }

// Entry is one member as the directory lists them.
export type Entry = {
  username: string
  full_name: string
  headline: string | null
  avatar_url: string | null
  // In effect now, most senior first.
  roles: string[]
  department: DepartmentRef | null
  batch_year?: number
  email?: string
  phone?: string
}

export type DirectoryMeta = { next_cursor?: string; total: number }

// Filters are the directory's query, as text from the address bar.
export type Filters = {
  department?: string
  role?: string
  batch?: string
  q?: string
}

// PublicProfile is a member's profile page. Signed-in viewers get roles,
// Department and Batch; email and phone only when the member shares them.
export type PublicProfile = {
  user_id: string
  username: string
  full_name: string
  headline?: string
  bio?: string
  avatar_url?: string
  public_profile_enabled: boolean
  show_email: boolean
  show_phone: boolean
  email?: string
  phone?: string
  linkedin_url?: string
  github_url?: string
  portfolio_url?: string
  roles?: string[]
  department?: DepartmentRef
  batch_year?: number
}

export type BatchCount = { batch_year: number; count: number }

export type Overview = {
  department: { code: string; name: string; description: string | null }
  hod: Entry | null
  counts: { students: number; faculty: number; students_by_batch: BatchCount[] }
  // HODs, placement officers and faculty of the Department, most senior first.
  staff: Entry[]
}

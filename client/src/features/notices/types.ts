export type Category = 'official' | 'department' | 'placement'

export type AudienceRule = {
  department_id: string | null
  department_code: string | null
  batch_year: number | null
  role: string | null
}

// A published Announcement as a reader sees it. Authors and approvers get
// more fields (status notes, edits); the composer and queue add those types.
export type Notice = {
  id: string
  title: string
  body: string
  category: Category
  status: string
  publisher_id: string
  publisher_name: string
  audience: AudienceRule[]
  published_at: string | null
  expires_at: string | null
  created_at: string
}

export type FeedMeta = {
  next_cursor?: string
}

export const categories: { value: Category; label: string; short: string }[] = [
  { value: 'official', label: 'Official', short: 'Official' },
  { value: 'department', label: 'Department', short: 'Dept' },
  { value: 'placement', label: 'Placement', short: 'Placement' },
]

export function isCategory(value: string | null): value is Category {
  return value === 'official' || value === 'department' || value === 'placement'
}

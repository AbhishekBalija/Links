import type { Category, Notice } from '../notices/types'

// A rule as the composer sends it. A missing field matches anyone.
export type RuleInput = {
  department_id?: string
  batch_year?: number
  role?: string
}

export type EditInfo = {
  status: 'pending' | 'rejected'
  review_note?: string
  approver?: string
  title: string
  body: string
  category: Category
  audience: RuleInput[]
  expires_at: string | null
}

// One of the author's own Announcements, in any status.
export type Authored = Notice & {
  status: 'draft' | 'pending' | 'published' | 'rejected' | 'withdrawn'
  review_note?: string
  approver?: string
  edit?: EditInfo
}

export type MineFilter = 'attention' | 'draft' | 'waiting' | 'live' | 'ended'

export type Preview = {
  publishes_directly: boolean
  approver: string | null
  reach: number
}

export type Department = { id: string; code: string; name: string }

export type Draft = {
  title: string
  body: string
  category: Category
  audience: RuleInput[]
  expires_at: string | null
}

// Something waiting in the approver's queue.
export type QueueItem = Notice & {
  kind: 'new' | 'edit'
  submitted_at: string
  approver: string
  // For an edit: the text readers see now.
  live?: { title: string; body: string }
}

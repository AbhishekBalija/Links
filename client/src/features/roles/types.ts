// A Role assignment as GET /api/v1/admin/users/:id/roles lists it.
export type RoleState = 'active' | 'scheduled' | 'ended'

export type RoleAssignment = {
  id: string
  role: string
  scope_type: 'global' | 'department'
  scope_id: string | null
  department: { id: string; code: string; name: string } | null
  starts_at: string
  ends_at: string | null
  state: RoleState
}

export type PersonRef = { user_id: string; full_name: string }
export type WorkItem = { id: string; title: string }

// Handover is what ending a role does to the person's unfinished work
// (ADR 0028), from the preview or the end itself.
export type Handover = {
  withdrawn_announcements: WorkItem[]
  closed_edits: WorkItem[]
  returned_events: WorkItem[]
  moved_events: { id: string; title: string; starts_at: string; organiser: PersonRef | null }[]
  organiser_needed: boolean
  organiser_options: PersonRef[]
}

export type GrantInput = {
  role: string
  scope_type: 'global' | 'department'
  scope_id?: string
  starts_at?: string
  ends_at?: string
}

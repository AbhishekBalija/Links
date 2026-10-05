import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { Notice } from '../notices/types'
import type { Opportunity } from '../jobs/types'
import type { PlacementSummary } from './placement'
import type { MyWork } from './author'
import type { NewRole } from './welcome'

export type Dashboard = {
  user: {
    full_name: string
    roles: string[]
    department: { id: string; code: string; name: string } | null
  }
  notices: { items: Notice[]; has_more: boolean }
  // Only for HODs, the principal and admins. Announcements and Event
  // proposals are counted apart.
  approvals?: {
    pending_count: number
    oldest_submitted_at: string | null
    events_pending_count: number
    oldest_event_submitted_at: string | null
  }
  // Only for users who can post.
  my_announcements?: { draft: number; pending: number; rejected: number; edits_waiting: number }
  // An author's own posts and events: sent back to them, or waiting.
  my_work?: MyWork
  // A role to welcome the person to, once (student coordinators for now).
  new_role?: NewRole
  // The next open Opportunities the user is eligible for, when there are any.
  opportunities?: { items: Opportunity[]; has_more: boolean }
  // Only for placement staff.
  placement?: PlacementSummary
  // An HOD's own Department.
  department?: DepartmentPanel
  // The principal's and admins' view of every Department.
  college?: { departments: CollegeDepartment[]; departments_without_hod: number }
  // For whoever decides Access requests (HODs for their Department).
  access_requests?: { pending_count: number; oldest_requested_at: string | null }
  // Who class lists and staff invites let in, and the latest imports.
  lists?: Lists
}

export type DepartmentPanel = {
  code: string
  name: string
  students: number
  staff: number
  students_by_batch: { batch_year: number; count: number }[]
  upcoming_events: { id: string; title: string; event_type: string; location: string; starts_at: string }[]
}

export type CollegeDepartment = {
  code: string
  name: string
  students: number
  staff: number
  hod: { full_name: string; username: string } | null
}

export type Lists = {
  waiting_count: number
  has_more: boolean
  recent_imports: {
    imported_at: string
    imported_by: { full_name: string }
    rows: number
    created: number
    failed: number
    batches: { department_code: string; batch_year: number; created: number }[] | null
  }[]
}

export function useDashboard(enabled = true) {
  return useQuery({
    queryKey: ['dashboard'],
    enabled,
    queryFn: ({ signal }) => apiRequest<Dashboard>('/api/v1/dashboard', { signal }),
  })
}

// useWelcomed records that the welcome was closed. Home drops it at once,
// so it never flashes back while the request is on its way.
export function useWelcomed() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiRequest(`/api/v1/me/roles/${encodeURIComponent(id)}/welcomed`, { method: 'POST' }),
    onMutate: (id) => {
      queryClient.setQueryData<Dashboard>(['dashboard'], (data) => (data?.new_role?.id === id ? { ...data, new_role: undefined } : data))
    },
  })
}

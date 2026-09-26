import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { Notice } from '../notices/types'

export type Dashboard = {
  user: {
    full_name: string
    roles: string[]
    department: { id: string; code: string; name: string } | null
  }
  notices: { items: Notice[]; has_more: boolean }
  // Only for HODs, the principal and admins.
  approvals?: { pending_count: number; oldest_submitted_at: string | null }
  // Only for users who can post.
  my_announcements?: { draft: number; pending: number; rejected: number; edits_waiting: number }
}

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: ({ signal }) => apiRequest<Dashboard>('/api/v1/dashboard', { signal }),
  })
}

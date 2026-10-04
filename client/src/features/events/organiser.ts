import { dayMonth, reviewerAt } from './standing'
import type { CampusEvent } from './types'

type Viewer = { user_id: string; roles: string[] }

// isOrganiser mirrors the server's rule for who runs a published event: its
// Organiser (the proposer, unless it was handed to someone else), the HOD
// of its Department, the principal and admins. The server checks again on
// every organiser action; this only decides what to show.
export function isOrganiser(event: Pick<CampusEvent, 'proposer_id' | 'organiser' | 'department'>, viewer: Viewer, viewerDepartment: { id: string } | null) {
  if ((event.organiser?.user_id ?? event.proposer_id) === viewer.user_id) return true
  if (viewer.roles.includes('principal') || viewer.roles.includes('admin')) return true
  return viewer.roles.includes('hod') && event.department !== null && event.department.id === viewerDepartment?.id
}

// publishedLine says when an event went live and who approved it:
// "Published 26 Sep after the CS HOD and the principal approved it".
export function publishedLine(event: Pick<CampusEvent, 'published_at' | 'reviews' | 'department'>) {
  if (!event.published_at) return 'Published'
  const approvers = (event.reviews ?? []).filter((r) => r.decision === 'approve').map((r) => reviewerAt(r.stage, event))
  const date = `Published ${dayMonth(event.published_at)}`
  return approvers.length > 0 ? `${date} after ${approvers.join(' and ')} approved it` : date
}

// organiserName is who runs the event, for "Organised by".
export function organiserName(event: Pick<CampusEvent, 'proposer_name' | 'organiser'>): string {
  return event.organiser?.full_name ?? event.proposer_name
}

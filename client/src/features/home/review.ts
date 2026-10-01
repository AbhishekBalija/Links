import type { Dashboard } from './api'

// waitingForReview is everything waiting for the reviewer: Announcements
// (and edits) plus Event proposals. Home and the sidebar badge both use it,
// so the two numbers always agree.
export function waitingForReview(approvals: Dashboard['approvals']): number {
  if (!approvals) return 0
  return approvals.pending_count + approvals.events_pending_count
}

import type { ApplicantCounts, Opportunity } from '../jobs/types'

export type StaffTone = 'plain' | 'going'

// staffStanding is where an Opportunity stands for the office. A published
// one whose deadline has passed takes no more applications, so it says so
// rather than "Open".
export function staffStanding(item: Pick<Opportunity, 'status' | 'apply_by' | 'closed_at'>, now = new Date()): { label: string; tone: StaffTone } {
  if (item.status === 'draft') return { label: 'Draft', tone: 'plain' }
  if (item.status === 'published') {
    return new Date(item.apply_by) > now ? { label: 'Open', tone: 'going' } : { label: 'Deadline passed', tone: 'plain' }
  }
  const early = item.closed_at && new Date(item.closed_at) < new Date(item.apply_by)
  return { label: early ? 'Closed early' : 'Closed', tone: 'plain' }
}

// pipelineWords puts applicant counts into words beside the bar, so no
// status is told by colour alone.
export function pipelineWords(counts: ApplicantCounts): string {
  if (counts.total === 0) return 'No applicants yet'
  const parts = [`${counts.applied} to review`]
  if (counts.shortlisted) parts.push(`${counts.shortlisted} shortlisted`)
  if (counts.selected) parts.push(`${counts.selected} selected`)
  if (counts.rejected) parts.push(`${counts.rejected} rejected`)
  return parts.join(' · ')
}

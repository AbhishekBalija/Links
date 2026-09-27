import type { CampusEvent, Review } from './types'

export type HistoryEntry = {
  at: string
  who: string
  what: string
  // The reviewer's note or the cancel reason, quoted under the line.
  note?: string
  tone?: 'warning' | 'danger'
}

const decisions: Record<Review['stage'], Record<Review['decision'], string>> = {
  hod: { approve: 'approved it', request_changes: 'asked for changes', reject: 'rejected it' },
  final: {
    approve: 'gave final approval, so it was published',
    request_changes: 'asked for changes at final approval',
    reject: 'rejected it at final approval',
  },
}

const tones: Partial<Record<Review['decision'], HistoryEntry['tone']>> = { request_changes: 'warning', reject: 'danger' }

// A proposal submitted within a minute of being created never sat as a draft.
const MINUTE = 60_000

// proposalHistory is what happened to one of the author's proposals, newest
// first: saving, submitting, each review, publishing and cancelling. Only the
// latest submission is known, so a resubmission shows once.
export function proposalHistory(event: CampusEvent): HistoryEntry[] {
  const entries: HistoryEntry[] = []
  const submitted = event.submitted_at ? new Date(event.submitted_at).getTime() : null
  if (submitted === null || submitted - new Date(event.created_at).getTime() > MINUTE) {
    entries.push({ at: event.created_at, who: 'You', what: 'saved a draft' })
  }
  if (event.submitted_at) entries.push({ at: event.submitted_at, who: 'You', what: 'submitted it' })

  const reviews = event.reviews ?? []
  for (const review of reviews) {
    const who = review.stage === 'hod' && event.department ? `${review.reviewer_name}, ${event.department.code} HOD` : review.reviewer_name
    const entry: HistoryEntry = { at: review.decided_at, who, what: decisions[review.stage][review.decision] }
    if (review.note) entry.note = review.note
    const tone = tones[review.decision]
    if (tone) entry.tone = tone
    entries.push(entry)
  }
  // The principal and admins publish their own proposals with no review.
  const finalApproval = reviews.some((r) => r.stage === 'final' && r.decision === 'approve')
  if (event.published_at && !finalApproval) entries.push({ at: event.published_at, who: 'You', what: 'published it' })
  if (event.cancelled_at) {
    const entry: HistoryEntry = { at: event.cancelled_at, who: 'Cancelled', what: '' }
    if (event.cancel_reason) entry.note = event.cancel_reason
    entries.push(entry)
  }
  return entries.sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
}

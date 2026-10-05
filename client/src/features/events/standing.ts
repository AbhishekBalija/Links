import { timeAgo } from '../notices/format'
import type { CampusEvent, Review } from './types'
import { monthsShort, wallClock, weekdaysShort } from '../../shared/time/college'

// The tabs of My posts. "needs" is what waits for the author's changes.
export type PostTab = 'needs' | 'draft' | 'waiting' | 'live' | 'ended'

export type ProposalTone = 'neutral' | 'warning' | 'live' | 'danger'

export type ProposalStanding = {
  tab: PostTab
  // The short status tag: "Draft", "With the CS HOD", "Live".
  tag: string
  tone: ProposalTone
  // What the date means for this status: "asked 2 hours ago".
  when: string
}

// "2 Oct". Months keep three letters, as on the date tiles.
export function dayMonth(iso: string) {
  const c = wallClock(iso)
  return `${c.day} ${monthsShort[c.month - 1]}`
}

// "Fri 2 Oct".
export function dayAndDate(iso: string) {
  return `${weekdaysShort[wallClock(iso).weekday]} ${dayMonth(iso)}`
}

// reviewerAt names who reviews a stage: the Department's HOD, or the
// principal for final approval and for college-wide events.
export function reviewerAt(stage: Review['stage'], event: Pick<CampusEvent, 'department'>) {
  return stage === 'hod' && event.department ? `the ${event.department.code} HOD` : 'the principal'
}

// latestReview is the most recent decision, which is what a sent-back or
// rejected proposal is waiting on.
export function latestReview(event: Pick<CampusEvent, 'reviews'>): Review | undefined {
  const reviews = event.reviews ?? []
  return reviews[reviews.length - 1]
}

// proposalStanding is where one of the author's own events stands: which tab
// of My posts it belongs to and how its row describes it.
export function proposalStanding(event: CampusEvent, now = new Date()): ProposalStanding {
  const decided = latestReview(event)?.decided_at ?? event.updated_at
  switch (event.status) {
    case 'draft':
      return { tab: 'draft', tag: 'Draft', tone: 'neutral', when: `saved ${timeAgo(event.updated_at, now)}` }
    case 'submitted':
    case 'hod_approved': {
      const stage = event.status === 'submitted' ? 'hod' : 'final'
      return { tab: 'waiting', tag: `With ${reviewerAt(stage, event)}`, tone: 'neutral', when: `sent ${timeAgo(event.submitted_at ?? event.updated_at, now)}` }
    }
    case 'hod_changes_requested':
    case 'final_changes_requested': {
      const stage = event.status === 'hod_changes_requested' ? 'hod' : 'final'
      return { tab: 'needs', tag: `Changes asked by ${reviewerAt(stage, event)}`, tone: 'warning', when: `asked ${timeAgo(decided, now)}` }
    }
    case 'hod_rejected':
    case 'final_rejected':
      return { tab: 'ended', tag: 'Rejected', tone: 'danger', when: `rejected ${timeAgo(decided, now)}` }
    case 'published':
      if (new Date(event.ends_at) <= now) {
        return { tab: 'ended', tag: 'Over', tone: 'neutral', when: `was on ${dayAndDate(event.starts_at)}` }
      }
      return { tab: 'live', tag: 'Live', tone: 'live', when: `on ${dayAndDate(event.starts_at)}` }
    case 'cancelled':
      return { tab: 'ended', tag: 'Cancelled', tone: 'neutral', when: `cancelled ${timeAgo(event.cancelled_at ?? event.updated_at, now)}` }
  }
}

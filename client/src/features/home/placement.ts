import { daysLeft } from '../jobs/format'
import type { ApplicantCounts, OpportunityType } from '../jobs/types'

// Drive is one open Opportunity in the placement summary on Home.
export type Drive = {
  id: string
  opportunity_type: OpportunityType
  title: string
  company: string
  apply_by: string
  applicant_counts: ApplicantCounts
}

export type PlacementSummary = {
  open_count: number
  awaiting_review_count: number
  // The open drives, soonest deadline first (up to five).
  drives: Drive[]
}

// showOpenJobs keeps "Open jobs for you" to students: the dashboard sends
// open Opportunities to anyone eligible, and staff can't apply.
export function showOpenJobs(roles: string[], section: { items: unknown[]; has_more: boolean } | undefined): boolean {
  return roles.includes('student') && Boolean(section && section.items.length > 0)
}

// closing turns the countdown into a phrase: "closes in 2 days".
function closing(applyBy: string, now: Date): string {
  const left = daysLeft(applyBy, now)
  if (left === 'Closes today') return 'closes today'
  if (left === 'Closes tomorrow') return 'closes tomorrow'
  return `closes in ${left.replace(' left', '')}`
}

// placementLine is the sentence under the placement officer's greeting:
// what waits for review, and which drive closes first.
export function placementLine(summary: PlacementSummary, now = new Date()): string {
  const waiting = summary.awaiting_review_count
  const first = summary.drives[0]
  if (!first && waiting === 0) return 'No drives are open and nothing is waiting for review.'
  const review =
    waiting === 0 ? 'Nothing is waiting for review.' : `${waiting} ${waiting === 1 ? 'application is' : 'applications are'} waiting for review.`
  return first ? `${review} ${first.company} ${closing(first.apply_by, now)}.` : review
}

// reviewFirst is the open drive with the most applications waiting, the
// best place to start reviewing.
export function reviewFirst(drives: Drive[]): Drive | undefined {
  const waiting = drives.filter((d) => d.applicant_counts.applied > 0)
  return waiting.sort((a, b) => b.applicant_counts.applied - a.applicant_counts.applied)[0]
}

import { describe, expect, it } from 'vitest'
import { proposalStanding } from './standing'
import type { CampusEvent, Review } from './types'
import { collegeInstant } from '../../shared/time/college'

// Sunday 27 September 2026, evening.
const now = collegeInstant(2026, 9, 27, 18, 0)
const hoursAgo = (hours: number) => new Date(now.getTime() - hours * 3600_000).toISOString()
const at = (month: number, day: number, hour: number) => collegeInstant(2026, month, day, hour).toISOString()

function event(overrides: Partial<CampusEvent>): CampusEvent {
  return {
    id: 'e1',
    title: 'Guest talk',
    description: '',
    event_type: 'talk',
    status: 'draft',
    proposer_id: 'u1',
    proposer_name: 'Meera N',
    organiser: null,
    department: { id: 'd1', code: 'CS' },
    faculty_mentor: null,
    location: 'CS Seminar Hall',
    starts_at: at(10, 2, 14),
    ends_at: at(10, 2, 16),
    capacity: null,
    audience: [],
    created_at: hoursAgo(72),
    updated_at: hoursAgo(3),
    reviews: [],
    ...overrides,
  }
}

const review = (stage: Review['stage'], decision: Review['decision'], hours: number): Review => ({
  stage,
  decision,
  note: 'Please move it to 3 pm.',
  reviewer_name: 'Asha Rao',
  decided_at: hoursAgo(hours),
})

describe('proposalStanding', () => {
  it('keeps a draft under Drafts', () => {
    expect(proposalStanding(event({ status: 'draft' }), now)).toEqual({
      tab: 'draft',
      tag: 'Draft',
      tone: 'neutral',
      when: 'saved 3 hours ago',
    })
  })

  it('says who has a proposal while it waits', () => {
    const submitted = event({ status: 'submitted', submitted_at: hoursAgo(26) })
    expect(proposalStanding(submitted, now)).toEqual({ tab: 'waiting', tag: 'With the CS HOD', tone: 'neutral', when: 'sent yesterday' })
    const passed = event({ status: 'hod_approved', submitted_at: hoursAgo(26) })
    expect(proposalStanding(passed, now).tag).toBe('With the principal')
    const collegeWide = event({ status: 'submitted', department: null, submitted_at: hoursAgo(26) })
    expect(proposalStanding(collegeWide, now).tag).toBe('With the principal')
  })

  it('puts changes asked under Needs you, naming who asked', () => {
    const hod = event({ status: 'hod_changes_requested', reviews: [review('hod', 'request_changes', 2)] })
    expect(proposalStanding(hod, now)).toEqual({ tab: 'needs', tag: 'Changes asked by the CS HOD', tone: 'warning', when: 'asked 2 hours ago' })
    const final = event({ status: 'final_changes_requested', reviews: [review('hod', 'approve', 30), review('final', 'request_changes', 5)] })
    expect(proposalStanding(final, now).tag).toBe('Changes asked by the principal')
  })

  it('puts a rejected proposal under Ended, since it cannot be sent again', () => {
    const rejected = event({ status: 'final_rejected', reviews: [review('final', 'reject', 5)] })
    expect(proposalStanding(rejected, now)).toEqual({ tab: 'ended', tag: 'Rejected', tone: 'danger', when: 'rejected 5 hours ago' })
  })

  it('shows a published event as Live until it ends', () => {
    const live = event({ status: 'published', published_at: hoursAgo(24) })
    expect(proposalStanding(live, now)).toEqual({ tab: 'live', tag: 'Live', tone: 'live', when: 'on Fri 2 Oct' })
    const over = event({ status: 'published', starts_at: at(9, 20, 10), ends_at: at(9, 20, 12) })
    expect(proposalStanding(over, now)).toEqual({ tab: 'ended', tag: 'Over', tone: 'neutral', when: 'was on Sun 20 Sep' })
  })

  it('ends a cancelled event', () => {
    const cancelled = event({ status: 'cancelled', cancelled_at: hoursAgo(50) })
    expect(proposalStanding(cancelled, now)).toEqual({ tab: 'ended', tag: 'Cancelled', tone: 'neutral', when: 'cancelled 2 days ago' })
  })
})

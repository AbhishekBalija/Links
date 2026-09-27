import { describe, expect, it } from 'vitest'
import { proposalHistory } from './history'
import type { CampusEvent } from './types'

const on = (day: number, hour = 10) => new Date(2026, 8, day, hour).toISOString()

const base = {
  department: { id: 'd1', code: 'CS' },
  created_at: on(22),
  updated_at: on(24),
  reviews: [],
} as unknown as CampusEvent

describe('proposalHistory', () => {
  it('tells the story newest first, with the reviewer’s note', () => {
    const event = {
      ...base,
      status: 'hod_changes_requested',
      submitted_at: on(23),
      reviews: [{ stage: 'hod', decision: 'request_changes', note: 'Move it to 3 pm.', reviewer_name: 'Asha Rao', decided_at: on(24) }],
    } as CampusEvent
    expect(proposalHistory(event)).toEqual([
      { at: on(24), who: 'Asha Rao, CS HOD', what: 'asked for changes', note: 'Move it to 3 pm.', tone: 'warning' },
      { at: on(23), who: 'You', what: 'submitted it' },
      { at: on(22), who: 'You', what: 'saved a draft' },
    ])
  })

  it('skips the draft step for a proposal submitted straight away, and shows the outcome', () => {
    const event = {
      ...base,
      status: 'cancelled',
      created_at: on(22),
      submitted_at: on(22),
      published_at: on(25),
      cancelled_at: on(26),
      cancel_reason: 'The speaker cannot travel.',
      reviews: [
        { stage: 'hod', decision: 'approve', reviewer_name: 'Asha Rao', decided_at: on(23) },
        { stage: 'final', decision: 'approve', reviewer_name: 'Dr Rao', decided_at: on(25) },
      ],
    } as CampusEvent
    expect(proposalHistory(event)).toEqual([
      { at: on(26), who: 'Cancelled', what: '', note: 'The speaker cannot travel.' },
      { at: on(25), who: 'Dr Rao', what: 'gave final approval, so it was published' },
      { at: on(23), who: 'Asha Rao, CS HOD', what: 'approved it' },
      { at: on(22), who: 'You', what: 'submitted it' },
    ])
  })

  it('marks a rejection', () => {
    const event = {
      ...base,
      status: 'final_rejected',
      submitted_at: on(22),
      created_at: on(22),
      reviews: [{ stage: 'final', decision: 'reject', note: 'Clashes with exams.', reviewer_name: 'Dr Rao', decided_at: on(24) }],
    } as CampusEvent
    expect(proposalHistory(event)[0]).toEqual({ at: on(24), who: 'Dr Rao', what: 'rejected it at final approval', note: 'Clashes with exams.', tone: 'danger' })
  })
})

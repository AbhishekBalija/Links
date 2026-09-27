import { describe, expect, it } from 'vitest'
import { approvalCopy } from './review'
import type { EventQueueItem } from './types'

const on = (day: number) => new Date(2026, 8, day, 10).toISOString()
const item = (overrides: Partial<EventQueueItem>) =>
  ({ stage: 'hod', department: { id: 'd-cs', code: 'CS' }, audience: [{ department_id: 'd-cs', department_code: 'CS', role: 'student' }], reviews: [], ...overrides }) as EventQueueItem

describe('approvalCopy', () => {
  it('tells an HOD that the principal decides next', () => {
    expect(approvalCopy(item({}), ['hod'])).toEqual({
      sentence: 'Approving sends it to the principal for final approval.',
      button: 'Approve',
      publishes: false,
      earlier: null,
    })
  })

  it('tells the principal reviewing for a Department without an HOD that final approval is theirs too', () => {
    expect(approvalCopy(item({}), ['principal']).sentence).toBe('No CS HOD is listed, so you review it here and again at final approval.')
  })

  it('publishes at final approval, naming the HOD who approved it', () => {
    const final = item({
      stage: 'final',
      reviews: [{ stage: 'hod', decision: 'approve', reviewer_name: 'Asha Rao', decided_at: on(25) }],
    })
    expect(approvalCopy(final, ['principal'])).toEqual({
      sentence: 'Approving publishes it to CS students now.',
      button: 'Approve and publish',
      publishes: true,
      earlier: 'Asha Rao, CS HOD approved it on 25 Sep. Yours is the final approval.',
    })
  })

  it('reads naturally for a college-wide event', () => {
    expect(approvalCopy(item({ stage: 'final', audience: [] }), ['principal']).sentence).toBe('Approving publishes it to the whole college now.')
  })

  it('says so when the HOD stage was skipped', () => {
    expect(approvalCopy(item({ stage: 'final' }), ['admin']).earlier).toBe('It skipped the HOD review, so yours is the only approval.')
  })
})

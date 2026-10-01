import { describe, expect, it } from 'vitest'
import { pipelineWords, staffStanding } from './standing'

const now = new Date('2026-09-30T04:30:00Z')
const base = { status: 'published' as const, apply_by: '2026-10-02T18:29:00Z', closed_at: null }

describe('staffStanding', () => {
  it('names each state the office cares about', () => {
    expect(staffStanding({ ...base, status: 'draft' }, now)).toMatchObject({ label: 'Draft', tone: 'plain' })
    expect(staffStanding(base, now)).toMatchObject({ label: 'Open', tone: 'going' })
    expect(staffStanding({ ...base, apply_by: '2026-09-29T18:29:00Z' }, now)).toMatchObject({ label: 'Deadline passed', tone: 'plain' })
    expect(staffStanding({ ...base, status: 'closed', closed_at: '2026-09-30T04:00:00Z' }, now)).toMatchObject({ label: 'Closed early', tone: 'plain' })
  })

  it('says a closed one closed at its deadline when it was closed after it', () => {
    expect(staffStanding({ ...base, status: 'closed', apply_by: '2026-09-29T18:29:00Z', closed_at: '2026-09-30T04:00:00Z' }, now).label).toBe('Closed')
  })
})

describe('pipelineWords', () => {
  it('lists the statuses that have anyone, in words', () => {
    expect(pipelineWords({ total: 42, applied: 30, shortlisted: 8, selected: 2, rejected: 2, withdrawn: 1 })).toBe('30 to review · 8 shortlisted · 2 selected · 2 rejected')
    expect(pipelineWords({ total: 3, applied: 3, shortlisted: 0, selected: 0, rejected: 0, withdrawn: 0 })).toBe('3 to review')
    expect(pipelineWords({ total: 0, applied: 0, shortlisted: 0, selected: 0, rejected: 0, withdrawn: 0 })).toBe('No applicants yet')
  })
})

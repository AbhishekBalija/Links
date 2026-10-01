import { describe, expect, it } from 'vitest'
import { placementLine, reviewFirst, showOpenJobs, type Drive } from './placement'

const now = new Date('2026-09-30T04:30:00Z')
const counts = (applied: number) => ({ total: applied, applied, shortlisted: 0, selected: 0, rejected: 0, withdrawn: 0 })
const drive = (company: string, applyBy: string, applied: number): Drive => ({
  id: company,
  opportunity_type: 'job',
  title: 'Trainee',
  company,
  apply_by: applyBy,
  applicant_counts: counts(applied),
})

describe('showOpenJobs', () => {
  it('shows open jobs to students who have some, and to nobody else', () => {
    const section = { items: [{ id: 'o1' }], has_more: false }
    expect(showOpenJobs(['student'], section)).toBe(true)
    expect(showOpenJobs(['student'], { items: [], has_more: false })).toBe(false)
    expect(showOpenJobs(['student'], undefined)).toBe(false)
    expect(showOpenJobs(['faculty'], section)).toBe(false)
  })
})

describe('placementLine', () => {
  it('says what waits and which drive closes first', () => {
    expect(placementLine({ open_count: 2, awaiting_review_count: 56, drives: [drive('Acme Systems', '2026-10-02T12:00:00Z', 30)] }, now)).toBe(
      '56 applications are waiting for review. Acme Systems closes in 2 days.',
    )
    expect(placementLine({ open_count: 1, awaiting_review_count: 1, drives: [drive('Kaveri', '2026-09-30T15:00:00Z', 1)] }, now)).toBe(
      '1 application is waiting for review. Kaveri closes today.',
    )
  })

  it('stays calm when nothing is open or waiting', () => {
    expect(placementLine({ open_count: 0, awaiting_review_count: 0, drives: [] }, now)).toBe('No drives are open and nothing is waiting for review.')
    expect(placementLine({ open_count: 1, awaiting_review_count: 0, drives: [drive('Acme', '2026-10-01T12:00:00Z', 0)] }, now)).toBe(
      'Nothing is waiting for review. Acme closes tomorrow.',
    )
  })
})

describe('reviewFirst', () => {
  it('picks the open drive with the most applications waiting', () => {
    expect(reviewFirst([drive('A', '2026-10-02T12:00:00Z', 3), drive('B', '2026-10-05T12:00:00Z', 30)])?.company).toBe('B')
    expect(reviewFirst([drive('A', '2026-10-02T12:00:00Z', 0)])).toBeUndefined()
  })
})

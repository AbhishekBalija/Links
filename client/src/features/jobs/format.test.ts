import { describe, expect, it } from 'vitest'
import { deadlineLine, deadlineParts, daysLeft, isUrgent, jobGroup, jobMeta } from './format'

// Wednesday 30 September 2026, 10 am in India.
const now = new Date('2026-09-30T04:30:00Z')

describe('deadline wording', () => {
  it('counts calendar days, and says today and tomorrow in words', () => {
    expect(daysLeft('2026-09-30T18:29:00Z', now)).toBe('Closes today')
    expect(daysLeft('2026-10-01T12:00:00Z', now)).toBe('Closes tomorrow')
    expect(daysLeft('2026-10-02T18:29:00Z', now)).toBe('2 days left')
    expect(daysLeft('2026-10-15T12:00:00Z', now)).toBe('15 days left')
  })

  it('says Closed once the deadline has passed', () => {
    expect(daysLeft('2026-09-30T04:00:00Z', now)).toBe('Closed')
  })

  it('treats three days or fewer as urgent, and a passed deadline as not', () => {
    expect(isUrgent('2026-10-03T12:00:00Z', now)).toBe(true)
    expect(isUrgent('2026-10-04T12:00:00Z', now)).toBe(false)
    expect(isUrgent('2026-09-29T12:00:00Z', now)).toBe(false)
  })

  it('groups the next seven days as closing this week', () => {
    expect(jobGroup('2026-10-07T12:00:00Z', now)).toBe('Closing this week')
    expect(jobGroup('2026-10-08T12:00:00Z', now)).toBe('Later')
  })

  it('splits the deadline into the tile parts', () => {
    expect(deadlineParts('2026-10-02T18:29:00Z')).toEqual({ day: '02', month: 'OCT' })
  })

  it('writes the full deadline without a comma, as the rest of LINKS does', () => {
    expect(deadlineLine('2026-10-02T12:00:00Z').date).toBe('Friday 2 October')
  })
})

describe('jobMeta', () => {
  it('joins company, place and pay, skipping what is missing', () => {
    expect(jobMeta({ company: 'Acme Systems', location: 'Mysuru', compensation: '4.5 LPA' })).toBe('Acme Systems · Mysuru · 4.5 LPA')
    expect(jobMeta({ company: 'Placement cell', location: null, compensation: null })).toBe('Placement cell')
  })
})

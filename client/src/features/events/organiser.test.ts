import { describe, expect, it } from 'vitest'
import { isOrganiser, publishedLine } from './organiser'
import type { CampusEvent } from './types'

const on = (day: number) => new Date(2026, 8, day, 10).toISOString()
const event = {
  proposer_id: 'u-proposer',
  department: { id: 'd-cs', code: 'CS' },
  published_at: on(26),
  reviews: [],
} as unknown as CampusEvent

describe('isOrganiser', () => {
  it('counts the proposer, the principal and admins', () => {
    expect(isOrganiser(event, { user_id: 'u-proposer', roles: ['student_coordinator'] }, null)).toBe(true)
    expect(isOrganiser(event, { user_id: 'u-p', roles: ['principal'] }, null)).toBe(true)
    expect(isOrganiser(event, { user_id: 'u-a', roles: ['admin'] }, null)).toBe(true)
  })

  it("counts the HOD of the event's Department only", () => {
    expect(isOrganiser(event, { user_id: 'u-h', roles: ['hod'] }, { id: 'd-cs' })).toBe(true)
    expect(isOrganiser(event, { user_id: 'u-h', roles: ['hod'] }, { id: 'd-me' })).toBe(false)
    expect(isOrganiser({ ...event, department: null }, { user_id: 'u-h', roles: ['hod'] }, { id: 'd-cs' })).toBe(false)
  })

  it('leaves out everyone else, faculty of the same Department included', () => {
    expect(isOrganiser(event, { user_id: 'u-f', roles: ['faculty'] }, { id: 'd-cs' })).toBe(false)
  })
})

describe('publishedLine', () => {
  it('names who approved it', () => {
    const reviewed = {
      ...event,
      reviews: [
        { stage: 'hod', decision: 'approve', reviewer_name: 'Asha Rao', decided_at: on(25) },
        { stage: 'final', decision: 'approve', reviewer_name: 'Dr Rao', decided_at: on(26) },
      ],
    } as CampusEvent
    expect(publishedLine(reviewed)).toBe('Published 26 Sep after the CS HOD and the principal approved it')
  })

  it('names only the principal when the HOD stage was skipped', () => {
    const final = { ...event, reviews: [{ stage: 'final', decision: 'approve', reviewer_name: 'Dr Rao', decided_at: on(26) }] } as CampusEvent
    expect(publishedLine(final)).toBe('Published 26 Sep after the principal approved it')
  })

  it('says only the date when the principal or an admin published it', () => {
    expect(publishedLine(event)).toBe('Published 26 Sep')
  })
})

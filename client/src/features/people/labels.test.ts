import { describe, expect, it } from 'vitest'
import { byLetter, countLine, highlight, roleLine } from './labels'
import type { Entry } from './types'

const cs = { code: 'CS', name: 'Computer Science and Engineering' }

function entry(overrides: Partial<Entry>): Entry {
  return {
    username: 'someone',
    full_name: 'Someone',
    headline: null,
    avatar_url: null,
    roles: [],
    department: null,
    ...overrides,
  }
}

describe('roleLine', () => {
  it('names a student by Department code and Batch', () => {
    const student = entry({ roles: ['student'], department: cs, batch_year: 2023 })
    expect(roleLine(student)).toBe('Student · CS · batch 2023')
    expect(roleLine(student, { short: true })).toBe('Student · CS · 2023')
  })

  it('uses the most senior role and the Department name for staff', () => {
    expect(roleLine(entry({ roles: ['hod', 'faculty'], department: cs }))).toBe('HOD · Computer Science and Engineering')
  })

  it('keeps the Batch for a coordinator who is also a student', () => {
    const coordinator = entry({ roles: ['student_coordinator', 'student'], department: cs, batch_year: 2023 })
    expect(roleLine(coordinator)).toBe('Student coordinator · CS · batch 2023')
  })

  it('works without a Department or roles', () => {
    expect(roleLine(entry({ roles: ['principal'] }))).toBe('Principal')
    expect(roleLine(entry({}))).toBe('')
  })
})

describe('byLetter', () => {
  it('groups people under the first letter of their name, in list order', () => {
    const groups = byLetter([entry({ full_name: 'asha' }), entry({ full_name: 'Aditi' }), entry({ full_name: 'Bala' }), entry({ full_name: '9 Lives' })])
    expect(groups.map((group) => [group.letter, group.people.length])).toEqual([
      ['A', 2],
      ['B', 1],
      ['#', 1],
    ])
  })
})

describe('countLine', () => {
  it('says how many people and where', () => {
    expect(countLine(412, { department: 'CS' })).toBe('412 people in CS')
    expect(countLine(1, { department: 'CS' })).toBe('1 person in CS')
    expect(countLine(1204, {})).toBe('1,204 people')
  })

  it('names the role and Batch when they are filtered', () => {
    expect(countLine(104, { department: 'CS', role: 'student', batch: '2023' })).toBe('104 students in CS, batch 2023')
    expect(countLine(36, { department: 'CS', role: 'faculty' })).toBe('36 faculty in CS')
    expect(countLine(1, { role: 'hod' })).toBe('1 HOD')
    expect(countLine(3, { role: 'hod' })).toBe('3 HODs')
  })
})

describe('highlight', () => {
  it('splits a name around the first case-insensitive match', () => {
    expect(highlight('Asha Rao', 'ASH')).toEqual({ before: '', match: 'Ash', after: 'a Rao' })
  })

  it('returns null when the name does not contain the search', () => {
    expect(highlight('Akash S', 'asha')).toBeNull()
    expect(highlight('Akash S', '')).toBeNull()
  })
})

import { describe, expect, it } from 'vitest'
import { addedOn, fixRefusal, listParams, mayFixRows, whoLine } from './logic'
import type { WaitingPerson } from './types'

const person = (over: Partial<WaitingPerson> = {}): WaitingPerson => ({
  user_id: 'u1',
  full_name: 'Kavya Rao',
  email: 'kavya.rao@gmail.com',
  kind: 'student',
  role: '',
  usn: '4MN25CS001',
  batch_year: 2025,
  department_code: 'CS',
  added_at: '2026-09-28T10:00:00Z',
  added_by: { full_name: 'Asha Rao' },
  ...over,
})

describe('whoLine', () => {
  it('shows a student by USN and a staff member by role', () => {
    expect(whoLine(person())).toBe('4MN25CS001')
    expect(whoLine(person({ kind: 'staff', role: 'faculty', usn: '' }))).toBe('Faculty')
  })
})

describe('addedOn', () => {
  it('says Today for today in India, otherwise the day and month', () => {
    const now = new Date('2026-10-04T12:00:00+05:30')
    expect(addedOn('2026-10-04T03:00:00Z', now)).toBe('Today')
    expect(addedOn('2026-09-28T10:00:00Z', now)).toBe('28 Sep')
  })
})

describe('listParams', () => {
  it('sends only the filters that are set', () => {
    expect(listParams({ department: '', kind: '' }).toString()).toBe('')
    expect(listParams({ department: 'CS', kind: 'student' }).toString()).toBe('department=CS&kind=student')
  })
})

describe('mayFixRows', () => {
  it('lets admins and HODs fix rows, not the principal', () => {
    expect(mayFixRows(['admin'])).toBe(true)
    expect(mayFixRows(['hod'])).toBe(true)
    expect(mayFixRows(['principal'])).toBe(false)
  })
})

describe('fixRefusal', () => {
  it("says an email is someone else's", () => {
    expect(fixRefusal({ email: 'student' }, 'kavya.m@gmail.com')).toBe('kavya.m@gmail.com is already someone else\'s on LINKS. Each person signs in with their own email.')
  })
  it('says when the person signed in meanwhile', () => {
    expect(fixRefusal(undefined, 'x@gmail.com')).toBeNull()
  })
})

import { describe, expect, it } from 'vitest'
import { welcomeLine, type NewRole } from './welcome'

const role = (over: Partial<NewRole> = {}): NewRole => ({
  id: 'r1',
  role: 'student_coordinator',
  department: { code: 'CS', name: 'Computer Science & Engineering' },
  assigned_by: 'Asha Rao',
  started_at: '2026-10-05T09:00:00Z',
  ...over,
})
const now = new Date('2026-10-05T15:00:00Z')

describe('welcomeLine', () => {
  it('says who made them a coordinator, for which department, and when', () => {
    expect(welcomeLine(role(), now)).toBe('Asha Rao made you a student coordinator for Computer Science & Engineering today.')
    expect(welcomeLine(role({ started_at: '2026-10-01T09:00:00Z' }), now)).toBe(
      'Asha Rao made you a student coordinator for Computer Science & Engineering on 1 October.',
    )
  })

  it('leaves out who when LINKS does not know', () => {
    expect(welcomeLine(role({ assigned_by: null }), now)).toBe('You are now a student coordinator for Computer Science & Engineering.')
  })
})

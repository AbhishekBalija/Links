import { describe, expect, it } from 'vitest'
import { assignmentLine, consequences, defaultOrganiser, firstName, grantableRoles, grantPayload, grantProblems, mayManage, needsDepartment, sortAssignments } from './roles'
import type { Handover, RoleAssignment } from './types'

const cs = { id: 'd1', code: 'CS', name: 'Computer Science' }
const assignment = (over: Partial<RoleAssignment> = {}): RoleAssignment => ({
  id: 'r1',
  role: 'faculty',
  scope_type: 'department',
  scope_id: 'd1',
  department: cs,
  starts_at: '2024-06-01T00:00:00Z',
  ends_at: null,
  state: 'active',
  ...over,
})

const empty: Handover = {
  withdrawn_announcements: [],
  closed_edits: [],
  returned_events: [],
  moved_events: [],
  organiser_needed: false,
  organiser_options: [],
}

describe('assignmentLine', () => {
  it('says where an active role applies and since when', () => {
    expect(assignmentLine(assignment())).toBe('Computer Science · Since 1 Jun 2024')
  })

  it('says when a scheduled role starts', () => {
    expect(assignmentLine(assignment({ state: 'scheduled', starts_at: '2026-11-01T00:00:00Z' }))).toBe('Computer Science · Starts 1 Nov 2026')
  })

  it('gives an ended role its dates', () => {
    expect(assignmentLine(assignment({ state: 'ended', starts_at: '2022-06-01T00:00:00Z', ends_at: '2024-05-31T00:00:00Z' }))).toBe(
      'Computer Science · 1 Jun 2022 to 31 May 2024',
    )
  })

  it('calls a global role the whole college', () => {
    expect(assignmentLine(assignment({ role: 'principal', scope_type: 'global', scope_id: null, department: null }))).toBe('Whole college · Since 1 Jun 2024')
  })
})

describe('who manages which roles (ADR 0027)', () => {
  it('lets an HOD appoint student coordinators only', () => {
    expect(grantableRoles(['hod'])).toEqual(['student_coordinator'])
    expect(mayManage(['hod'], 'faculty')).toBe(false)
  })

  it('lets the principal appoint senior staff but not coordinators or admins', () => {
    expect(grantableRoles(['principal'])).toEqual(['faculty', 'hod', 'placement_officer'])
    expect(mayManage(['principal'], 'student_coordinator')).toBe(false)
    expect(mayManage(['principal'], 'admin')).toBe(false)
  })

  it('lets an admin manage every role', () => {
    expect(grantableRoles(['admin'])).toEqual(['faculty', 'hod', 'placement_officer', 'student_coordinator', 'principal', 'admin'])
  })

  it('gives everyone else nothing to manage', () => {
    expect(grantableRoles(['faculty', 'student'])).toEqual([])
  })

  it('knows which roles need a department', () => {
    expect(needsDepartment('hod')).toBe(true)
    expect(needsDepartment('placement_officer')).toBe(false)
  })
})

describe('consequences', () => {
  it('names each piece of work and what happens to it', () => {
    const handover: Handover = {
      ...empty,
      withdrawn_announcements: [{ id: 'a1', title: 'Coding club meet' }],
      closed_edits: [{ id: 'a2', title: 'Library hours' }],
      returned_events: [{ id: 'e1', title: 'Open source sprint' }],
      moved_events: [{ id: 'e2', title: 'Hackathon prep', starts_at: '2026-10-20T10:00:00Z', organiser: null }],
    }
    expect(consequences(handover, 'Rohan')).toEqual([
      { title: 'Coding club meet', text: "isn't published yet. It will be withdrawn." },
      { title: 'Library hours', text: 'has an edit waiting. The edit is dropped; the published notice stays.' },
      { title: 'Open source sprint', text: "goes back to Rohan's drafts, where they can read it but not send it." },
      { title: 'Hackathon prep', text: 'is coming up. Students who are going keep their place.' },
    ])
  })

  it('is empty when nothing is waiting or coming up', () => {
    expect(consequences(empty, 'Kiran')).toEqual([])
  })
})

describe('defaultOrganiser', () => {
  it('is whoever the server would hand the events to', () => {
    const hod = { user_id: 'u9', full_name: 'Asha Rao' }
    const handover = { ...empty, moved_events: [{ id: 'e2', title: 'Hackathon prep', starts_at: '2026-10-20T10:00:00Z', organiser: hod }] }
    expect(defaultOrganiser(handover)).toEqual(hod)
  })

  it('is nobody when the server needs someone picked', () => {
    const handover = { ...empty, organiser_needed: true, moved_events: [{ id: 'e2', title: 'Day', starts_at: '2026-10-20T10:00:00Z', organiser: null }] }
    expect(defaultOrganiser(handover)).toBeNull()
  })
})

describe('grantPayload', () => {
  const today = '2026-10-03'

  it('leaves out a start of today so the role starts now', () => {
    expect(grantPayload({ role: 'placement_officer', departmentId: '', starts: today, ends: '' }, today)).toEqual({ role: 'placement_officer', scope_type: 'global' })
  })

  it('starts a later role at midnight in India and ends it at the end of its last day', () => {
    expect(grantPayload({ role: 'hod', departmentId: 'd1', starts: '2026-11-01', ends: '2027-05-31' }, today)).toEqual({
      role: 'hod',
      scope_type: 'department',
      scope_id: 'd1',
      starts_at: '2026-11-01T00:00:00+05:30',
      ends_at: '2027-05-31T23:59:59+05:30',
    })
  })
})

describe('grantProblems', () => {
  const today = '2026-10-03'

  it('asks for a department for a department role', () => {
    expect(grantProblems({ role: 'hod', departmentId: '', starts: today, ends: '' }, today)).toEqual({ department: 'An HOD role needs a department.' })
  })

  it('refuses a start in the past and an end before the start', () => {
    expect(grantProblems({ role: 'faculty', departmentId: 'd1', starts: '2026-10-01', ends: '2026-09-30' }, today)).toEqual({
      starts: "A role can't start in the past.",
      ends: 'The end has to be on or after the start.',
    })
  })

  it('has nothing to say about a valid grant', () => {
    expect(grantProblems({ role: 'faculty', departmentId: 'd1', starts: today, ends: '' }, today)).toEqual({})
  })
})

describe('firstName', () => {
  it('skips a title so copy reads "End Kiran\'s role"', () => {
    expect(firstName('Prof. Kiran Hegde')).toBe('Kiran')
    expect(firstName('Dr Asha Rao')).toBe('Asha')
  })

  it('keeps a plain first name, and a lone title as it is', () => {
    expect(firstName('Rohan Shetty')).toBe('Rohan')
    expect(firstName('Dr.')).toBe('Dr.')
  })
})

describe('sortAssignments', () => {
  it('puts roles in effect first, then scheduled, then ended', () => {
    const list = [
      assignment({ id: 'scheduled', state: 'scheduled' }),
      assignment({ id: 'ended', state: 'ended' }),
      assignment({ id: 'active', state: 'active' }),
    ]
    expect(sortAssignments(list).map((a) => a.id)).toEqual(['active', 'scheduled', 'ended'])
  })
})

describe('assignmentLine for a student', () => {
  it("names the student's department and batch instead of the whole college", () => {
    const studentRole = assignment({ role: 'student', scope_type: 'global', scope_id: null, department: null })
    expect(assignmentLine(studentRole, { department: 'Computer Science', batch: 2025 })).toBe('Computer Science · Batch 2025 · Since 1 Jun 2024')
  })
})

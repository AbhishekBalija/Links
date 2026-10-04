import { describe, expect, it } from 'vitest'
import { addedMessage, isCollegeWide, nextWithoutHOD, refusal, staffPayload, staffProblems, staffRoles } from './logic'

const departments = [
  { id: 'd-cs', code: 'CS', name: 'Computer Science and Engineering' },
  { id: 'd-ec', code: 'EC', name: 'Electronics and Communication Engineering' },
  { id: 'd-me', code: 'ME', name: 'Mechanical Engineering' },
]

describe('staffRoles', () => {
  it('lists roles in the order an admin adds people, and only faculty for an HOD', () => {
    expect(staffRoles(['admin'])).toEqual(['hod', 'faculty', 'placement_officer', 'principal', 'admin'])
    expect(staffRoles(['hod'])).toEqual(['faculty'])
    expect(staffRoles(['principal'])).toEqual([])
  })

  it('knows which roles need no department', () => {
    expect(isCollegeWide('principal')).toBe(true)
    expect(isCollegeWide('hod')).toBe(false)
  })
})

describe('staffProblems', () => {
  it('asks for a name, a whole email and a department where one is needed', () => {
    expect(staffProblems({ fullName: ' ', email: 'kiran@', role: 'hod', departmentId: '' })).toEqual({
      full_name: 'Enter their full name.',
      email: 'Enter a full email, like name@college.edu.',
      department: 'Choose their department.',
    })
    expect(staffProblems({ fullName: 'Dr. Shalini Rao', email: 'principal@mitm.ac.in', role: 'principal', departmentId: '' })).toEqual({})
  })
})

describe('staffPayload', () => {
  it('sends a department role with its department and a college-wide role without one', () => {
    expect(staffPayload({ fullName: ' Prof. Kiran Hegde ', email: ' kiran@college.edu ', role: 'faculty', departmentId: 'd-cs' })).toEqual({
      full_name: 'Prof. Kiran Hegde',
      email: 'kiran@college.edu',
      role: 'faculty',
      scope_type: 'department',
      scope_id: 'd-cs',
    })
    expect(staffPayload({ fullName: 'Dr. Shalini Rao', email: 'principal@mitm.ac.in', role: 'principal', departmentId: 'd-cs' })).toEqual({
      full_name: 'Dr. Shalini Rao',
      email: 'principal@mitm.ac.in',
      role: 'principal',
      scope_type: 'global',
    })
  })
})

describe('refusal', () => {
  it('says why an email is refused, under the email', () => {
    expect(refusal({ email: 'student' }, 'asha@gmail.com', '')).toEqual({ field: 'email', text: "asha@gmail.com is a current student's email. Staff need their own email." })
    expect(refusal({ email: 'request' }, 'kiran@gmail.com', '')).toEqual({
      field: 'email',
      text: 'kiran@gmail.com asked to join as a student. If they work here, remove that request first, then add them as staff.',
      link: { to: '/admin/requests', label: 'Go to the request' },
    })
    expect(refusal({ email: 'member', username: 'kiran' }, 'kiran@college.edu', '')).toEqual({
      field: 'email',
      text: 'kiran@college.edu is already on LINKS. Give them this role on their profile.',
      link: { to: '/people/kiran', label: 'Open their profile' },
    })
  })

  it("sends an HOD to their own Access requests", () => {
    expect(refusal({ email: 'request' }, 'kiran@gmail.com', '', '/approvals/access')?.link).toEqual({ to: '/approvals/access', label: 'Go to the request' })
  })

  it('names the HOD a department already has, under the department', () => {
    expect(refusal({ scope_id: 'has_hod', hod_name: 'Dr. Meera Iyer' }, 'x@college.edu', 'Computer Science and Engineering')).toEqual({
      field: 'department',
      text: 'Computer Science and Engineering already has an HOD, Dr. Meera Iyer. End that role on their profile first.',
    })
  })

  it('has nothing to say about other refusals', () => {
    expect(refusal(undefined, 'x@college.edu', '')).toBeNull()
  })
})

describe('addedMessage', () => {
  it('says who was added, as what, and whether they were emailed', () => {
    expect(addedMessage({ fullName: 'Prof. Kiran Hegde', email: 'kiran@college.edu', role: 'faculty', departmentName: 'Computer Science and Engineering', emailed: true })).toBe(
      'Prof. Kiran Hegde was added as Faculty, Computer Science and Engineering. We emailed kiran@college.edu how to sign in.',
    )
    expect(addedMessage({ fullName: 'Dr. Shalini Rao', email: 'principal@mitm.ac.in', role: 'principal', departmentName: '', emailed: false })).toBe(
      "Dr. Shalini Rao was added as Principal. The email to principal@mitm.ac.in couldn't be sent, so let them know they can sign in.",
    )
  })
})

describe('nextWithoutHOD', () => {
  it('picks the next department that has no HOD, after the one just filled', () => {
    expect(nextWithoutHOD(departments, new Set(['CS']), 'd-cs')).toBe('d-ec')
    expect(nextWithoutHOD(departments, new Set(['CS', 'EC']), 'd-ec')).toBe('d-me')
    expect(nextWithoutHOD(departments, new Set(['CS', 'EC', 'ME']), 'd-me')).toBe('')
  })
})

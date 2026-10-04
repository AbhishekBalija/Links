import { describe, expect, it } from 'vitest'
import { accessRequestsPlace, whyItsHere } from './reason'
import type { AccessRequest } from './types'

const request = (over: Partial<AccessRequest> = {}): AccessRequest => ({
  id: 'u1',
  email: 'asha.rao@gmail.com',
  profile: { full_name: 'Asha Rao', username: 'asha.rao1' },
  student_identity: { usn: '4MN23CS042', department_code: 'CS', department_name: 'Computer Science and Engineering', department_has_hod: true, batch_year: 2023 },
  created_at: '2026-10-01T08:00:00Z',
  reported_at: null,
  ...over,
})

describe('whyItsHere', () => {
  it('says a request comes from someone on no class list', () => {
    expect(whyItsHere(request())).toBe('Not on the Computer Science and Engineering class list. They proved this email and sent a request.')
  })

  it('says a reported row needs its details checked', () => {
    expect(whyItsHere(request({ reported_at: '2026-10-02T08:00:00Z' }))).toMatch(/^This row came from the class list\. On first sign-in the person said it wasn't them/)
  })

  it('says a reported staff invite is about who it was added for, not a class list', () => {
    expect(whyItsHere(request({ reported_at: '2026-10-02T08:00:00Z', student_identity: undefined }))).toBe(
      "This staff member was added by an admin. On first sign-in the person said it wasn't them, so check who it was added for before letting anyone in. Approving gives back the role they were added with.",
    )
  })

  it("explains that a department with no HOD sends its requests to admins", () => {
    const noHOD = request({ student_identity: { ...request().student_identity!, department_code: 'EC', department_name: 'Electronics and Communication Engineering', department_has_hod: false } })
    expect(whyItsHere(noHOD)).toBe('Not on any class list. Electronics and Communication Engineering has no HOD, so the request comes to admins.')
  })
})

describe('accessRequestsPlace', () => {
  it('puts them in the Approval queue for an HOD', () => {
    expect(accessRequestsPlace(['hod'])).toBe('approvals')
  })

  it('puts them in the Admin workspace for an admin, even one who is also an HOD', () => {
    expect(accessRequestsPlace(['admin'])).toBe('admin')
    expect(accessRequestsPlace(['hod', 'admin'])).toBe('admin')
  })

  it('leaves them off the screens for the principal and everyone else', () => {
    expect(accessRequestsPlace(['principal'])).toBeNull()
    expect(accessRequestsPlace(['faculty'])).toBeNull()
  })
})

import type { AccessRequest } from './types'

// whyItsHere is the "Why it's here" line on a request, so the reviewer knows
// what to check before deciding.
export function whyItsHere(request: AccessRequest): string {
  if (request.reported_at && !request.student_identity) {
    return "This staff member was added by an admin. On first sign-in the person said it wasn't them, so check who it was added for before letting anyone in. Approving gives back the role they were added with."
  }
  if (request.reported_at) {
    return "This row came from the class list. On first sign-in the person said it wasn't them, so check the name, USN and email before letting anyone in."
  }
  const department = request.student_identity?.department_name ?? 'their department'
  if (request.student_identity && !request.student_identity.department_has_hod) {
    return `Not on any class list. ${department} has no HOD, so the request comes to admins.`
  }
  return `Not on the ${department} class list. They proved this email and sent a request.`
}

// accessRequestsPlace says where someone decides Access requests: an HOD in
// the Approval queue, an admin in the Admin workspace. The principal can, but
// it isn't their job, so their screens don't show them.
export function accessRequestsPlace(roles: string[]): 'approvals' | 'admin' | null {
  if (roles.includes('admin')) return 'admin'
  if (roles.includes('hod')) return 'approvals'
  return null
}

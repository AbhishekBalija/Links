import { roleLabel } from '../../app/shell/nav'

// Adding staff (#189): what the form offers, checks and says.

export type StaffDraft = {
  fullName: string
  email: string
  role: string
  departmentId: string
}

export type Department = { id: string; code: string; name: string }

// The order an admin adds people on day 0: HODs first, then faculty and the
// college-wide roles. An HOD adds only faculty, to their own department.
const adminRoles = ['hod', 'faculty', 'placement_officer', 'principal', 'admin']

export function staffRoles(viewerRoles: string[]): string[] {
  if (viewerRoles.includes('admin')) return adminRoles
  if (viewerRoles.includes('hod')) return ['faculty']
  return []
}

// College-wide roles belong to no department.
export function isCollegeWide(role: string): boolean {
  return role === 'placement_officer' || role === 'principal' || role === 'admin'
}

// The principal and admins sign in with Google only, so their email must be
// a Google account.
export function isGoogleOnly(role: string): boolean {
  return role === 'principal' || role === 'admin'
}

export function staffProblems(draft: StaffDraft): Record<string, string> {
  const problems: Record<string, string> = {}
  if (draft.fullName.trim() === '') problems.full_name = 'Enter their full name.'
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(draft.email.trim())) problems.email = 'Enter a full email, like name@college.edu.'
  if (!isCollegeWide(draft.role) && draft.departmentId === '') problems.department = 'Choose their department.'
  return problems
}

export function staffPayload(draft: StaffDraft) {
  const base = { full_name: draft.fullName.trim(), email: draft.email.trim(), role: draft.role }
  if (isCollegeWide(draft.role)) return { ...base, scope_type: 'global' }
  return { ...base, scope_type: 'department', scope_id: draft.departmentId }
}

export type Refusal = { field: 'email' | 'department'; text: string; link?: { to: string; label: string } }

// refusal turns the server's reason for a 409 into the line under the field
// it is about, with a link when there is somewhere to go to fix it.
export function refusal(details: Record<string, unknown> | undefined, email: string, departmentName: string, requestsPath = '/admin/requests'): Refusal | null {
  if (!details) return null
  switch (details.email) {
    case 'student':
      return { field: 'email', text: `${email} is a current student's email. Staff need their own email.` }
    case 'request':
      return {
        field: 'email',
        text: `${email} asked to join as a student. If they work here, remove that request first, then add them as staff.`,
        link: { to: requestsPath, label: 'Go to the request' },
      }
    case 'member':
      return {
        field: 'email',
        text: `${email} is already on LINKS. Give them this role on their profile.`,
        link: { to: `/people/${String(details.username ?? '')}`, label: 'Open their profile' },
      }
  }
  if (details.scope_id === 'has_hod') {
    return { field: 'department', text: `${departmentName} already has an HOD, ${String(details.hod_name ?? '')}. End that role on their profile first.` }
  }
  return null
}

export function addedMessage(added: { fullName: string; email: string; role: string; departmentName: string; emailed: boolean }): string {
  const as = added.departmentName ? `${roleLabel(added.role)}, ${added.departmentName}` : roleLabel(added.role)
  const told = added.emailed
    ? `We emailed ${added.email} how to sign in.`
    : `The email to ${added.email} couldn't be sent, so let them know they can sign in.`
  return `${added.fullName} was added as ${as}. ${told}`
}

// nextWithoutHOD is the department to offer after adding an HOD: the next
// one in the list still without an HOD, or none when every one has one.
export function nextWithoutHOD(departments: Department[], withHOD: Set<string>, justFilledId: string): string {
  const start = departments.findIndex((d) => d.id === justFilledId)
  for (let step = 1; step <= departments.length; step++) {
    const department = departments[(start + step) % departments.length]
    if (!withHOD.has(department.code)) return department.id
  }
  return ''
}

import { roleLabel } from '../../app/shell/nav'
import type { GrantInput, Handover, PersonRef, RoleAssignment } from './types'
import { COLLEGE_TIME_ZONE, fromCollegeTime } from '../../shared/time/college'

// The order roles are offered in a grant form: staff first, then the roles
// only an admin grants.
const allGrantable = ['faculty', 'hod', 'placement_officer', 'student_coordinator', 'principal', 'admin']

// Who normally appoints each role (ADR 0027). The server checks the same
// rules; the screen only uses them to show the right buttons.
const managedBy: Record<string, string[]> = {
  admin: allGrantable,
  principal: ['faculty', 'hod', 'placement_officer'],
  hod: ['student_coordinator'],
}

export function grantableRoles(viewerRoles: string[]): string[] {
  const allowed = new Set(viewerRoles.flatMap((role) => managedBy[role] ?? []))
  return allGrantable.filter((role) => allowed.has(role))
}

export function mayManage(viewerRoles: string[], role: string): boolean {
  return grantableRoles(viewerRoles).includes(role)
}

// Department roles need a Department; the rest are for the whole college.
export function needsDepartment(role: string): boolean {
  return role === 'faculty' || role === 'hod' || role === 'student_coordinator'
}

function day(iso: string): string {
  return new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: COLLEGE_TIME_ZONE })
}

// grantedLine is what the profile says once a role is given: "Asha is a
// student coordinator now", or "... from 1 Nov 2026" for a later start.
export function grantedLine(firstName: string, role: string, label: string, starts: string, today: string): string {
  const when = starts === today ? 'now' : `from ${day(fromCollegeTime(starts, '00:00').toISOString())}`
  return role === 'student_coordinator' ? `${firstName} is a student coordinator ${when}.` : `${firstName} has the ${label} role ${when}.`
}

// assignmentLine reads "Computer Science · Since 1 Jun 2024", "… · Starts 1
// Nov 2026" or "… · 1 Jun 2022 to 31 May 2024".
// A student's role is college-wide, so it names their own Department and
// Batch from the profile instead.
export function assignmentLine(assignment: RoleAssignment, student?: { department?: string; batch?: number }): string {
  let where = assignment.department?.name ?? 'Whole college'
  if (assignment.role === 'student' && student?.department) {
    where = student.batch ? `${student.department} · Batch ${student.batch}` : student.department
  }
  if (assignment.state === 'scheduled') return `${where} · Starts ${day(assignment.starts_at)}`
  if (assignment.state === 'ended' && assignment.ends_at) return `${where} · ${day(assignment.starts_at)} to ${day(assignment.ends_at)}`
  return `${where} · Since ${day(assignment.starts_at)}`
}

export type Consequence = { title: string; text: string }

// consequences lists what ending the role does to each piece of the
// person's work, in the order the confirmation shows them.
export function consequences(handover: Handover, firstName: string): Consequence[] {
  return [
    ...handover.withdrawn_announcements.map((item) => ({ title: item.title, text: "isn't published yet. It will be withdrawn." })),
    ...handover.closed_edits.map((item) => ({ title: item.title, text: 'has an edit waiting. The edit is dropped; the published notice stays.' })),
    ...handover.returned_events.map((item) => ({ title: item.title, text: `goes back to ${firstName}'s drafts, where they can read it but not send it.` })),
    ...handover.moved_events.map((event) => ({ title: event.title, text: 'is coming up. Students who are going keep their place.' })),
  ]
}

// defaultOrganiser is who the server hands the upcoming Events to when
// nobody else is picked, or null when someone must be picked.
export function defaultOrganiser(handover: Handover): PersonRef | null {
  if (handover.organiser_needed) return null
  return handover.moved_events.find((event) => event.organiser)?.organiser ?? null
}

// GrantDraft is the grant form as typed: the start is yyyy-mm-dd from the
// date input, read as a day in India. A role has no end date: it ends by
// hand, which hands over the person's work (#204).
export type GrantDraft = { role: string; departmentId: string; starts: string }

// todayInIndia is today's date as a date input holds it.
export function todayInIndia(now = new Date()): string {
  return now.toLocaleDateString('en-CA', { timeZone: COLLEGE_TIME_ZONE })
}

// grantProblems checks the form before sending it, keyed by field. The
// server checks the same rules and more.
export function grantProblems(draft: GrantDraft, today: string): Record<string, string> {
  const problems: Record<string, string> = {}
  if (needsDepartment(draft.role) && !draft.departmentId) {
    const label = roleLabel(draft.role)
    problems.department = `${/^[AEIOU]|^HOD/.test(label) ? 'An' : 'A'} ${label} role needs a department.`
  }
  if (draft.starts < today) problems.starts = "A role can't start in the past."
  return problems
}

// grantPayload turns the form into the API's body. Starting today is left
// out so the role starts now; a later start begins at midnight.
export function grantPayload(draft: GrantDraft, today: string): GrantInput {
  const input: GrantInput = needsDepartment(draft.role)
    ? { role: draft.role, scope_type: 'department', scope_id: draft.departmentId }
    : { role: draft.role, scope_type: 'global' }
  if (draft.starts && draft.starts !== today) input.starts_at = `${draft.starts}T00:00:00+05:30`
  return input
}

const titles = new Set(['dr', 'prof', 'mr', 'mrs', 'ms', 'shri', 'smt'])

// firstName is how the copy addresses someone: "Prof. Kiran Hegde" is Kiran.
export function firstName(fullName: string): string {
  const words = fullName.trim().split(/\s+/)
  const first = words.find((word) => !titles.has(word.replace(/\.$/, '').toLowerCase()))
  return first ?? words[0] ?? ''
}

const stateOrder = { active: 0, scheduled: 1, ended: 2 }

// sortAssignments lists roles in effect first, then scheduled, then ended,
// keeping the server's order (newest start first) within each.
export function sortAssignments(list: RoleAssignment[]): RoleAssignment[] {
  return [...list].sort((a, b) => stateOrder[a.state] - stateOrder[b.state])
}

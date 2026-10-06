import { collegeDay, monthsLong, wallClock } from '../../shared/time/college'

// NewRole is a role the person hasn't been welcomed to yet. Home shows the
// welcome once; closing it is remembered on the account.
export type NewRole = {
  id: string
  role: 'student_coordinator'
  department: { code: string; name: string }
  assigned_by: string | null
  started_at: string
}


// welcomeLine says who made them a coordinator, where, and when.
export function welcomeLine(role: NewRole, now = new Date()): string {
  if (!role.assigned_by) return `You are now a student coordinator for ${role.department.name}.`
  const started = wallClock(role.started_at)
  const when = collegeDay(role.started_at) === collegeDay(now) ? 'today' : `on ${started.day} ${monthsLong[started.month - 1]}`
  return `${role.assigned_by} made you a student coordinator for ${role.department.name} ${when}.`
}

// NewRole is a role the person hasn't been welcomed to yet. Home shows the
// welcome once; closing it is remembered on the account.
export type NewRole = {
  id: string
  role: 'student_coordinator'
  department: { code: string; name: string }
  assigned_by: string | null
  started_at: string
}

const months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']

function sameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

// welcomeLine says who made them a coordinator, where, and when.
export function welcomeLine(role: NewRole, now = new Date()): string {
  if (!role.assigned_by) return `You are now a student coordinator for ${role.department.name}.`
  const started = new Date(role.started_at)
  const when = sameDay(started, now) ? 'today' : `on ${started.getDate()} ${months[started.getMonth()]}`
  return `${role.assigned_by} made you a student coordinator for ${role.department.name} ${when}.`
}

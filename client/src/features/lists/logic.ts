import { roleLabel } from '../../app/shell/nav'
import type { ListFilter, WaitingPerson } from './types'
import { COLLEGE_TIME_ZONE } from '../../shared/time/college'

// The college's calendar day as YYYY-MM-DD.
const dateKey = new Intl.DateTimeFormat('en-CA', { timeZone: COLLEGE_TIME_ZONE })
const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

// whoLine is a row's second column: a student's USN, a staff member's role.
export function whoLine(person: WaitingPerson): string {
  return person.kind === 'student' ? person.usn : roleLabel(person.role)
}

// addedOn is when they were added, by the college's calendar.
export function addedOn(iso: string, now = new Date()): string {
  const added = new Date(iso)
  const [, month, day] = dateKey.format(added).split('-')
  return dateKey.format(added) === dateKey.format(now) ? 'Today' : `${Number(day)} ${months[Number(month) - 1]}`
}

export function listParams(filter: ListFilter): URLSearchParams {
  const params = new URLSearchParams()
  if (filter.department) params.set('department', filter.department)
  if (filter.kind) params.set('kind', filter.kind)
  return params
}

// Admins and HODs fix or remove rows: the people who import (ADR 0029). The
// principal sees the list but isn't offered changes.
export function mayFixRows(roles: string[]): boolean {
  return roles.includes('admin') || roles.includes('hod')
}

// fixRefusal is the line under the email when the new one is taken.
export function fixRefusal(details: Record<string, unknown> | undefined, email: string): string | null {
  if (details && typeof details.email === 'string' && details.email !== 'unchanged') {
    return `${email} is already someone else's on LINKS. Each person signs in with their own email.`
  }
  return null
}

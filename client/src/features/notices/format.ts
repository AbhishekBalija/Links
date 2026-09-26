import type { AudienceRule } from './types'

const DAY = 24 * 60 * 60 * 1000

const rolePlurals: Record<string, string> = {
  student: 'students',
  student_coordinator: 'student coordinators',
  faculty: 'faculty',
  hod: 'HODs',
  placement_officer: 'placement officers',
  principal: 'principal',
  alumni: 'alumni',
  club_organizer: 'club organisers',
  admin: 'admins',
}

function capitalize(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

// "2 hours ago", "yesterday", "5 days ago"; older than a week shows the date.
export function timeAgo(iso: string, now = new Date()): string {
  const then = new Date(iso)
  const diff = now.getTime() - then.getTime()
  const minutes = Math.round(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return hours === 1 ? '1 hour ago' : `${hours} hours ago`
  const days = Math.round(diff / DAY)
  if (days <= 1) return 'yesterday'
  if (days < 7) return `${days} days ago`
  return then.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })
}

// "26 Sep 2026, 13:10" for the detail view.
export function fullDate(iso: string): string {
  return new Date(iso).toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

export type Expiry = { at: string; text: string; soon: boolean }

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

// Expiry says how long a notice stays up, in calendar days: a deadline at
// 11:00 tomorrow is "tomorrow" however late it is today. It counts as soon
// within a week, so deadlines stand out without every notice looking urgent.
export function expiry(iso: string | null, now = new Date()): Expiry | null {
  if (!iso) return null
  const ends = new Date(iso)
  // Math.round absorbs the hour a daylight-saving change adds or removes.
  const days = Math.round((startOfDay(ends).getTime() - startOfDay(now).getTime()) / DAY)
  let text: string
  if (days <= 0) text = 'today'
  else if (days === 1) text = 'tomorrow'
  else text = `in ${days} days`
  return { at: iso, text, soon: days <= 7 }
}

// audienceLabel turns Audience rules into words: "CS students, batch 2023".
export function audienceLabel(rules: AudienceRule[]): string {
  if (rules.length === 0) return 'Whole college'
  return rules.map(ruleLabel).join('; ')
}

function ruleLabel(rule: AudienceRule): string {
  const people = rule.role ? (rolePlurals[rule.role] ?? rule.role) : null
  const code = rule.department_code
  let label: string
  if (code && people) label = `${code} ${people}`
  else if (code) label = `Everyone in ${code}`
  else if (people) label = capitalize(`all ${people}`)
  else label = ''
  if (rule.batch_year) {
    label = label ? `${label}, batch ${rule.batch_year}` : `Batch ${rule.batch_year}`
  }
  return label
}

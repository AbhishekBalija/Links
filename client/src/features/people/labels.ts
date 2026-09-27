import { roleLabel } from '../../app/shell/nav'
import type { Entry, Filters } from './types'

// roleLine says who someone is in one line: "Student · CS · batch 2023" for
// students, "HOD · Computer Science and Engineering" for staff. Phones use the
// short form, "Student · CS · 2023".
export function roleLine(entry: Pick<Entry, 'roles' | 'department' | 'batch_year'>, { short = false } = {}) {
  const top = entry.roles[0]
  if (!top) return ''
  const parts = [roleLabel(top)]
  if (entry.batch_year !== undefined) {
    if (entry.department) parts.push(entry.department.code)
    parts.push(short ? String(entry.batch_year) : `batch ${entry.batch_year}`)
  } else if (entry.department) {
    parts.push(entry.department.name)
  }
  return parts.join(' · ')
}

export type LetterGroup = { letter: string; people: Entry[] }

// byLetter groups an alphabetical list under each name's first letter, so a
// long list reads like an index. Names that don't start with a letter go
// under "#".
export function byLetter(people: Entry[]): LetterGroup[] {
  const groups: LetterGroup[] = []
  for (const person of people) {
    const first = person.full_name.trim().charAt(0).toUpperCase()
    const letter = /[A-Z]/.test(first) ? first : '#'
    const last = groups[groups.length - 1]
    if (last && last.letter === letter) last.people.push(person)
    else groups.push({ letter, people: [person] })
  }
  return groups
}

const plurals: Record<string, [string, string]> = {
  student: ['student', 'students'],
  faculty: ['faculty', 'faculty'],
  hod: ['HOD', 'HODs'],
  student_coordinator: ['coordinator', 'coordinators'],
  placement_officer: ['placement officer', 'placement officers'],
  principal: ['principal', 'principals'],
  admin: ['admin', 'admins'],
  alumni: ['alumnus', 'alumni'],
  club_organizer: ['club organiser', 'club organisers'],
}

// peopleWord counts people by the role they are filtered to: "412 people",
// "104 students", "1 HOD".
export function peopleWord(total: number, role?: string) {
  const [one, many] = (role && plurals[role]) || ['person', 'people']
  return `${total.toLocaleString('en-IN')} ${total === 1 ? one : many}`
}

// countLine says how many people a list holds: "412 people in CS",
// "104 students in CS, batch 2023".
export function countLine(total: number, filters: Filters) {
  let line = peopleWord(total, filters.role)
  if (filters.department) line += ` in ${filters.department}`
  if (filters.batch) line += `${filters.department ? ',' : ''} batch ${filters.batch}`
  return line
}

// highlight finds the search inside a name, ignoring case, so the matching
// part can be marked. It returns null when the name doesn't contain it (a
// match on the username, the headline or a similar spelling).
export function highlight(name: string, q: string) {
  if (!q) return null
  const at = name.toLowerCase().indexOf(q.toLowerCase())
  if (at < 0) return null
  return { before: name.slice(0, at), match: name.slice(at, at + q.length), after: name.slice(at + q.length) }
}

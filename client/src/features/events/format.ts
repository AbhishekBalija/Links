// How Events read on screen: the date tile, the time span, the week groups
// and the seats line.

const monthShort = (d: Date) => d.toLocaleDateString('en-IN', { month: 'short' }).toUpperCase().slice(0, 3)
const weekdayShort = (d: Date) => d.toLocaleDateString('en-IN', { weekday: 'short' })

// dateParts is what the date tile shows: OCT, 02, THU.
export function dateParts(iso: string) {
  const d = new Date(iso)
  return { month: monthShort(d), day: String(d.getDate()).padStart(2, '0'), weekday: weekdayShort(d).toUpperCase().slice(0, 3) }
}

function clock(d: Date, { minutes }: { minutes: 'always' | 'when-needed' }) {
  const hour = d.getHours() % 12 || 12
  const mins = d.getMinutes()
  if (minutes === 'when-needed' && mins === 0) return String(hour)
  return `${hour}:${String(mins).padStart(2, '0')}`
}
const meridiem = (d: Date) => (d.getHours() < 12 ? 'am' : 'pm')
const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString()

// timeRange reads "2:30 to 4:00 pm" for a same-day event and names both days
// when it runs overnight: "Fri 9 am to Sat 9 am".
export function timeRange(startsIso: string, endsIso: string) {
  const starts = new Date(startsIso)
  const ends = new Date(endsIso)
  if (sameDay(starts, ends)) {
    const first = clock(starts, { minutes: 'always' })
    const last = `${clock(ends, { minutes: 'always' })} ${meridiem(ends)}`
    return meridiem(starts) === meridiem(ends) ? `${first} to ${last}` : `${first} ${meridiem(starts)} to ${last}`
  }
  const part = (d: Date) => `${weekdayShort(d)} ${clock(d, { minutes: 'when-needed' })} ${meridiem(d)}`
  return `${part(starts)} to ${part(ends)}`
}

// whenLine is the detail page's "When": the date and the times for a
// same-day Event, and both dates in one line for an overnight one.
export function whenLine(startsIso: string, endsIso: string): { date: string | null; time: string } {
  const starts = new Date(startsIso)
  const ends = new Date(endsIso)
  if (sameDay(starts, ends)) {
    return { date: starts.toLocaleDateString('en-IN', { weekday: 'long', day: 'numeric', month: 'long' }), time: timeRange(startsIso, endsIso) }
  }
  const part = (d: Date) =>
    `${weekdayShort(d)} ${d.getDate()} ${d.toLocaleDateString('en-IN', { month: 'short' })}, ${clock(d, { minutes: 'when-needed' })} ${meridiem(d)}`
  return { date: null, time: `${part(starts)} to ${part(ends)}` }
}

// happeningNow is true while an Event is on.
export function happeningNow(event: { starts_at: string; ends_at: string }, now = new Date()) {
  return new Date(event.starts_at) <= now && now < new Date(event.ends_at)
}

// startTime is when an Event begins: "2:30 pm", "9 am".
export function startTime(iso: string) {
  const d = new Date(iso)
  return `${clock(d, { minutes: 'when-needed' })} ${meridiem(d)}`
}

// Weeks start on Monday, as a college timetable does.
function weekStart(d: Date) {
  const start = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  start.setDate(start.getDate() - ((start.getDay() + 6) % 7))
  return start.getTime()
}
const WEEK = 7 * 24 * 60 * 60 * 1000
const monthName = (d: Date) => d.toLocaleDateString('en-IN', { month: 'long' })

// groupLabel puts an Event under "This week", "Next week" or "Later in
// October"; past Events read backwards: "Last week", "Earlier in September".
export function groupLabel(iso: string, now = new Date(), past = false) {
  const d = new Date(iso)
  const weeks = Math.round((weekStart(d) - weekStart(now)) / WEEK)
  if (weeks === 0) return 'This week'
  if (!past && weeks === 1) return 'Next week'
  if (past && weeks === -1) return 'Last week'
  const year = d.getFullYear() === now.getFullYear() ? '' : ` ${d.getFullYear()}`
  return `${past ? 'Earlier in' : 'Later in'} ${monthName(d)}${year}`
}

// seatsLine: "45 going" without a limit, "34 of 120 seats left" with one,
// and "60 of 60 going" once it is full.
export function seatsLine(capacity: number | null, going: number) {
  if (capacity === null) return `${going} going`
  if (going >= capacity) return `${capacity} of ${capacity} going`
  return `${capacity - going} of ${capacity} seats left`
}

export function isFull(capacity: number | null, going: number) {
  return capacity !== null && going >= capacity
}

// answersClosed is true once an Event starts or is cancelled; the server
// refuses answers then too.
export function answersClosed(event: { status: string; starts_at: string }, now = new Date()) {
  return event.status === 'cancelled' || new Date(event.starts_at) <= now
}

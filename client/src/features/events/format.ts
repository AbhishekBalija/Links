// How Events read on screen: the date tile, the time span, the week groups
// and the seats line.

import { clockTime, collegeDay, monthsLong, monthsShort, wallClock, weekdaysLong, weekdaysShort } from '../../shared/time/college'

// Every date and time here is the college's (#211), whatever the device's
// zone.

// dateParts is what the date tile shows: OCT, 02, THU.
export function dateParts(iso: string) {
  const c = wallClock(iso)
  return { month: monthsShort[c.month - 1].toUpperCase(), day: String(c.day).padStart(2, '0'), weekday: weekdaysShort[c.weekday].toUpperCase() }
}

function clock(iso: string, { minutes }: { minutes: 'always' | 'when-needed' }) {
  const c = wallClock(iso)
  const hour = c.hour % 12 || 12
  if (minutes === 'when-needed' && c.minute === 0) return String(hour)
  return `${hour}:${String(c.minute).padStart(2, '0')}`
}
const meridiem = (iso: string) => (wallClock(iso).hour < 12 ? 'am' : 'pm')
const sameDay = (a: string, b: string) => collegeDay(a) === collegeDay(b)
const weekdayOf = (iso: string) => weekdaysShort[wallClock(iso).weekday]

// timeRange reads "2:30 to 4:00 pm" for a same-day event and names both days
// when it runs overnight: "Fri 9 am to Sat 9 am".
export function timeRange(startsIso: string, endsIso: string) {
  if (sameDay(startsIso, endsIso)) {
    const first = clock(startsIso, { minutes: 'always' })
    const last = `${clock(endsIso, { minutes: 'always' })} ${meridiem(endsIso)}`
    return meridiem(startsIso) === meridiem(endsIso) ? `${first} to ${last}` : `${first} ${meridiem(startsIso)} to ${last}`
  }
  const part = (iso: string) => `${weekdayOf(iso)} ${clock(iso, { minutes: 'when-needed' })} ${meridiem(iso)}`
  return `${part(startsIso)} to ${part(endsIso)}`
}

// whenLine is the detail page's "When": the date and the times for a
// same-day Event, and both dates in one line for an overnight one.
export function whenLine(startsIso: string, endsIso: string): { date: string | null; time: string } {
  if (sameDay(startsIso, endsIso)) {
    const c = wallClock(startsIso)
    return { date: `${weekdaysLong[c.weekday]}, ${c.day} ${monthsLong[c.month - 1]}`, time: timeRange(startsIso, endsIso) }
  }
  const part = (iso: string) => {
    const c = wallClock(iso)
    return `${weekdaysShort[c.weekday]} ${c.day} ${monthsShort[c.month - 1]}, ${clock(iso, { minutes: 'when-needed' })} ${meridiem(iso)}`
  }
  return { date: null, time: `${part(startsIso)} to ${part(endsIso)}` }
}

// happeningNow is true while an Event is on.
export function happeningNow(event: { starts_at: string; ends_at: string }, now = new Date()) {
  return new Date(event.starts_at) <= now && now < new Date(event.ends_at)
}

// startTime is when an Event begins: "2:30 pm", "9 am".
export function startTime(iso: string) {
  return clockTime(iso)
}

// Weeks start on Monday, as a college timetable does: the college day
// number of that Monday.
function weekStart(at: Date | string) {
  return collegeDay(at) - ((wallClock(at).weekday + 6) % 7)
}

// groupLabel puts an Event under "This week", "Next week" or "Later in
// October"; past Events read backwards: "Last week", "Earlier in September".
export function groupLabel(iso: string, now = new Date(), past = false) {
  const weeks = Math.round((weekStart(iso) - weekStart(now)) / 7)
  if (weeks === 0) return 'This week'
  if (!past && weeks === 1) return 'Next week'
  if (past && weeks === -1) return 'Last week'
  const at = wallClock(iso)
  const year = at.year === wallClock(now).year ? '' : ` ${at.year}`
  return `${past ? 'Earlier in' : 'Later in'} ${monthsLong[at.month - 1]}${year}`
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

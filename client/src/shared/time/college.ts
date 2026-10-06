// College time (#211). Every date and time LINKS shows or takes in is the
// college's, not the device's: a coordinator proposing from a phone set to
// another zone must still create the event at the right time in India.
// The Home greeting is the one exception: it follows the reader's own day.

export const COLLEGE_TIME_ZONE = 'Asia/Kolkata'

const DAY = 24 * 60 * 60 * 1000

// WallClock is what a clock on the college wall shows. month is 1 to 12;
// weekday is 0 (Sunday) to 6.
export type WallClock = { year: number; month: number; day: number; hour: number; minute: number; weekday: number }

const parts = new Intl.DateTimeFormat('en-US', {
  timeZone: COLLEGE_TIME_ZONE,
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
  hour: 'numeric',
  minute: 'numeric',
  weekday: 'short',
  hourCycle: 'h23',
})
const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

export function wallClock(at: Date | string): WallClock {
  const found: Record<string, string> = {}
  for (const part of parts.formatToParts(new Date(at))) found[part.type] = part.value
  return {
    year: Number(found.year),
    month: Number(found.month),
    day: Number(found.day),
    hour: Number(found.hour),
    minute: Number(found.minute),
    weekday: weekdays.indexOf(found.weekday),
  }
}

// fromCollegeTime is the instant a date input ("2026-10-09") and a time
// input ("14:30") mean in the college.
export function fromCollegeTime(date: string, time: string, seconds = 0): Date {
  const [year, month, day] = date.split('-').map(Number)
  const [hour, minute] = time.split(':').map(Number)
  const asIfUTC = Date.UTC(year, month - 1, day, hour, minute, seconds)
  // The college's offset at that moment, read back from the clock itself.
  const shown = wallClock(new Date(asIfUTC))
  const offset = Date.UTC(shown.year, shown.month - 1, shown.day, shown.hour, shown.minute) - Math.floor(asIfUTC / 60000) * 60000
  return new Date(asIfUTC - offset)
}

// collegeDay numbers the college calendar's days, so two instants on the
// same college date get the same number and "tomorrow" is one more.
export function collegeDay(at: Date | string): number {
  const c = wallClock(at)
  return Date.UTC(c.year, c.month - 1, c.day) / DAY
}

export const monthsLong = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
export const monthsShort = monthsLong.map((m) => m.slice(0, 3))
export const weekdaysLong = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
export const weekdaysShort = weekdays

const pad = (n: number) => String(n).padStart(2, '0')

// collegeDate writes the college date: "2026-10-06" for a date input,
// "Tue 6 Oct" or "Tuesday 6 October".
export function collegeDate(at: Date | string, style: 'input' | 'short' | 'long'): string {
  const c = wallClock(at)
  if (style === 'input') return `${c.year}-${pad(c.month)}-${pad(c.day)}`
  if (style === 'short') return `${weekdaysShort[c.weekday]} ${c.day} ${monthsShort[c.month - 1]}`
  return `${weekdaysLong[c.weekday]} ${c.day} ${monthsLong[c.month - 1]}`
}

// collegeTimeInput is the college time for a time input: "14:30".
export function collegeTimeInput(at: Date | string): string {
  const c = wallClock(at)
  return `${pad(c.hour)}:${pad(c.minute)}`
}

// clockTime reads "2:30 pm", or "9 am" on the hour.
export function clockTime(at: Date | string): string {
  const c = wallClock(at)
  const hour = c.hour % 12 || 12
  const meridiem = c.hour < 12 ? 'am' : 'pm'
  return c.minute === 0 ? `${hour} ${meridiem}` : `${hour}:${pad(c.minute)} ${meridiem}`
}

// collegeInstant is the instant a college clock shows these numbers (month 1
// to 12), for building times in code and tests.
export function collegeInstant(year: number, month: number, day: number, hour = 0, minute = 0): Date {
  return fromCollegeTime(`${year}-${pad(month)}-${pad(day)}`, `${pad(hour)}:${pad(minute)}`)
}

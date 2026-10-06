import { COLLEGE_TIME_ZONE, collegeDate, fromCollegeTime } from '../../../../shared/time/college'

// Expiry is a date; the notice leaves the feed at the end of that day in the
// college (#211), whatever the author's device zone.
export function toEndOfDay(date: string): string {
  return fromCollegeTime(date, '23:59', 59).toISOString()
}

export function toDateInput(iso: string | null): string {
  if (!iso) return ''
  return collegeDate(iso, 'input')
}

export function friendlyDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-IN', { weekday: 'short', day: 'numeric', month: 'short', timeZone: COLLEGE_TIME_ZONE })
}

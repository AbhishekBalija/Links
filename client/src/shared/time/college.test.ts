import { describe, expect, it } from 'vitest'
import { collegeDay, collegeDate, fromCollegeTime, wallClock } from './college'

// The suite runs with the device in Los Angeles (see package.json), so these
// only pass if college time ignores the device's own zone.
describe('college time', () => {
  it('reads an instant as the college clock shows it', () => {
    // 20:00 UTC is 1:30 am the next day in India.
    expect(wallClock('2026-10-05T20:00:00Z')).toEqual({ year: 2026, month: 10, day: 6, hour: 1, minute: 30, weekday: 2 })
  })

  it('turns a date and time typed in the college into the right instant', () => {
    expect(fromCollegeTime('2026-10-09', '14:30').toISOString()).toBe('2026-10-09T09:00:00.000Z')
    expect(fromCollegeTime('2026-10-09', '23:59', 59).toISOString()).toBe('2026-10-09T18:29:59.000Z')
  })

  it('counts days on the college calendar', () => {
    // 11 pm and 1 am IST are a calendar day apart, though two hours apart.
    const late = '2026-10-05T17:30:00Z'
    const early = '2026-10-05T19:30:00Z'
    expect(collegeDay(early) - collegeDay(late)).toBe(1)
  })

  it('formats dates for the college', () => {
    expect(collegeDate('2026-10-05T20:00:00Z', 'input')).toBe('2026-10-06')
    expect(collegeDate('2026-10-05T20:00:00Z', 'short')).toBe('Tue 6 Oct')
    expect(collegeDate('2026-10-05T20:00:00Z', 'long')).toBe('Tuesday 6 October')
  })
})

import { describe, expect, it } from 'vitest'
import { answersClosed, dateParts, groupLabel, happeningNow, seatsLine, startTime, timeRange, whenLine } from './format'
import { fromCollegeTime } from '../../shared/time/college'

// College times (India), whatever zone the tests run in.
const pad = (n: number) => String(n).padStart(2, '0')
const at = (month: number, day: number, hour: number, minute = 0) => fromCollegeTime(`2026-${pad(month)}-${pad(day)}`, `${pad(hour)}:${pad(minute)}`).toISOString()
// Sunday 27 September 2026, evening.
const now = new Date(at(9, 27, 18))

describe('dateParts', () => {
  it('gives the tile its month, day and weekday', () => {
    expect(dateParts(at(10, 2, 14, 30))).toEqual({ month: 'OCT', day: '02', weekday: 'FRI' })
  })
})

describe('timeRange', () => {
  it('reads a same-day event as one span', () => {
    expect(timeRange(at(10, 2, 14, 30), at(10, 2, 16))).toBe('2:30 to 4:00 pm')
    expect(timeRange(at(10, 6, 10), at(10, 6, 13))).toBe('10:00 am to 1:00 pm')
  })

  it('names both days when an event runs overnight', () => {
    expect(timeRange(at(10, 3, 9), at(10, 4, 9))).toBe('Sat 9 am to Sun 9 am')
    expect(timeRange(at(10, 3, 9, 30), at(10, 4, 17))).toBe('Sat 9:30 am to Sun 5 pm')
  })
})

describe('whenLine', () => {
  it('gives a same-day event its date and times', () => {
    expect(whenLine(at(10, 2, 14, 30), at(10, 2, 16))).toEqual({ date: 'Friday, 2 October', time: '2:30 to 4:00 pm' })
  })

  it('names both dates once for an overnight event', () => {
    expect(whenLine(at(10, 3, 9), at(10, 4, 9))).toEqual({ date: null, time: 'Sat 3 Oct, 9 am to Sun 4 Oct, 9 am' })
  })
})

describe('happeningNow', () => {
  it('is true between the start and the end', () => {
    expect(happeningNow({ starts_at: at(9, 27, 17), ends_at: at(9, 27, 20) }, now)).toBe(true)
    expect(happeningNow({ starts_at: at(9, 27, 19), ends_at: at(9, 27, 20) }, now)).toBe(false)
    expect(happeningNow({ starts_at: at(9, 27, 14), ends_at: at(9, 27, 16) }, now)).toBe(false)
  })
})

describe('startTime', () => {
  it('drops the minutes on the hour', () => {
    expect(startTime(at(10, 2, 14, 30))).toBe('2:30 pm')
    expect(startTime(at(10, 3, 9))).toBe('9 am')
  })
})

describe('groupLabel', () => {
  it('groups upcoming events by week, then by month', () => {
    expect(groupLabel(at(9, 28, 10), now)).toBe('Next week')
    expect(groupLabel(at(9, 27, 20), now)).toBe('This week')
    expect(groupLabel(at(10, 5, 10), now)).toBe('Later in October')
    expect(groupLabel(at(11, 1, 17), now)).toBe('Later in November')
  })

  it('groups past events the other way', () => {
    expect(groupLabel(at(9, 24, 10), now, true)).toBe('This week')
    expect(groupLabel(at(9, 19, 14), now, true)).toBe('Last week')
    expect(groupLabel(at(9, 2, 14), now, true)).toBe('Earlier in September')
  })
})

describe('seatsLine', () => {
  it('says how many are going when there is no limit', () => {
    expect(seatsLine(null, 45)).toBe('45 going')
  })

  it('counts the seats left, or says it is full', () => {
    expect(seatsLine(120, 86)).toBe('34 of 120 seats left')
    expect(seatsLine(120, 119)).toBe('1 of 120 seats left')
    expect(seatsLine(60, 60)).toBe('60 of 60 going')
  })
})

describe('answersClosed', () => {
  it('closes answers once an event starts or is cancelled', () => {
    expect(answersClosed({ status: 'published', starts_at: at(10, 2, 14) }, now)).toBe(false)
    expect(answersClosed({ status: 'published', starts_at: at(9, 27, 17) }, now)).toBe(true)
    expect(answersClosed({ status: 'cancelled', starts_at: at(10, 2, 14) }, now)).toBe(true)
  })
})

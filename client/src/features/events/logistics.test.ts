import { describe, expect, it } from 'vitest'
import { checkLogistics, logisticsChanges } from './logistics'
import { fromEvent } from './proposal'
import type { CampusEvent } from './types'
import { collegeInstant } from '../../shared/time/college'

// Sunday 27 September 2026, evening.
const now = collegeInstant(2026, 9, 27, 18, 0)
const local = (month: number, day: number, hour: number, minute = 0) => collegeInstant(2026, month, day, hour, minute).toISOString()

const published = {
  event_type: 'talk',
  title: 'Guest talk',
  description: 'How search works.',
  location: 'CS Seminar Hall',
  starts_at: local(10, 2, 14, 30),
  ends_at: local(10, 2, 16),
  capacity: 120,
  audience: [],
} as unknown as CampusEvent

describe('logisticsChanges', () => {
  it('sends only what changed', () => {
    const form = { ...fromEvent(published), location: 'Main auditorium', startTime: '15:00' }
    expect(logisticsChanges(form, published)).toEqual({ location: 'Main auditorium', starts_at: local(10, 2, 15) })
  })

  it('removes the seat limit with null', () => {
    expect(logisticsChanges({ ...fromEvent(published), limitSeats: false }, published)).toEqual({ capacity: null })
  })

  it('sends nothing when nothing changed', () => {
    expect(logisticsChanges(fromEvent(published), published)).toEqual({})
  })
})

describe('checkLogistics', () => {
  it('keeps the limit at or above the people already going', () => {
    const form = { ...fromEvent(published), capacity: '40' }
    expect(checkLogistics(form, published, now, 45)).toEqual({ capacity: "45 are going. A limit can't be lower than that." })
    expect(checkLogistics({ ...form, capacity: '45' }, published, now, 45)).toEqual({})
  })

  it('needs a new start to be ahead, but leaves an unchanged start alone', () => {
    const past = { ...published, starts_at: local(9, 20, 10), ends_at: local(10, 20, 12) }
    expect(checkLogistics(fromEvent(past), past, now, 0)).toEqual({})
    const moved = { ...fromEvent(published), startDate: '2026-09-25' }
    expect(checkLogistics(moved, published, now, 0).starts).toBe("That's in the past.")
  })

  it('needs a place and an end after the start', () => {
    const form = { ...fromEvent(published), location: ' ', endTime: '14:00' }
    expect(checkLogistics(form, published, now, 0)).toEqual({ location: 'Say where it happens.', ends: 'It has to end after it starts.' })
  })
})

import { describe, expect, it } from 'vitest'
import { checkProposal, durationLabel, emptyProposal, fromEvent, toInput, type ProposalForm } from './proposal'
import type { CampusEvent } from './types'

// Sunday 27 September 2026, evening.
const now = new Date(2026, 8, 27, 18, 0)
const local = (month: number, day: number, hour: number, minute = 0) => new Date(2026, month - 1, day, hour, minute).toISOString()

const talk: ProposalForm = {
  ...emptyProposal([{ department_id: 'd1', role: 'student' }]),
  event_type: 'talk',
  title: '  Guest talk: building search at scale ',
  description: 'How a query becomes a ranked page.',
  location: 'CS Seminar Hall',
  startDate: '2026-10-02',
  startTime: '14:30',
  endDate: '2026-10-02',
  endTime: '16:00',
}

describe('toInput', () => {
  it('reads the dates and times in the proposer’s time zone', () => {
    expect(toInput(talk, { id: 'd1', code: 'CS' })).toEqual({
      title: 'Guest talk: building search at scale',
      description: 'How a query becomes a ranked page.',
      event_type: 'talk',
      department_id: 'd1',
      location: 'CS Seminar Hall',
      starts_at: local(10, 2, 14, 30),
      ends_at: local(10, 2, 16),
      capacity: null,
      audience: [{ department_id: 'd1', role: 'student' }],
    })
  })

  it('sends a seat limit only when one is set', () => {
    expect(toInput({ ...talk, limitSeats: true, capacity: '120' }, null).capacity).toBe(120)
    expect(toInput({ ...talk, limitSeats: false, capacity: '120' }, null).capacity).toBeNull()
  })
})

describe('fromEvent', () => {
  it('fills the form from a saved proposal', () => {
    const saved = {
      event_type: 'competition',
      title: 'Hack the Campus',
      description: '',
      location: 'Main auditorium',
      starts_at: local(10, 3, 9),
      ends_at: local(10, 4, 9),
      capacity: 120,
      audience: [{ department_id: 'd1', department_code: 'CS', role: 'student' }],
    } as CampusEvent
    expect(fromEvent(saved)).toEqual({
      event_type: 'competition',
      title: 'Hack the Campus',
      description: '',
      location: 'Main auditorium',
      startDate: '2026-10-03',
      startTime: '09:00',
      endDate: '2026-10-04',
      endTime: '09:00',
      limitSeats: true,
      capacity: '120',
      audience: [{ department_id: 'd1', role: 'student' }],
    })
  })
})

describe('checkProposal', () => {
  it('accepts a complete proposal', () => {
    expect(checkProposal(talk, now, { submitting: true })).toEqual({})
  })

  it('names every field that needs fixing', () => {
    const blank = emptyProposal([])
    expect(checkProposal(blank, now, { submitting: true })).toEqual({
      event_type: 'Choose what kind of event it is.',
      title: 'Give the event a title.',
      starts: 'Choose when it starts.',
      ends: 'Choose when it ends.',
      location: 'Say where it happens.',
    })
  })

  it('refuses a past start only when submitting, since drafts may be dated anything', () => {
    const past = { ...talk, startDate: '2026-09-22', endDate: '2026-09-22' }
    expect(checkProposal(past, now, { submitting: true })).toEqual({ starts: "That's in the past." })
    expect(checkProposal(past, now, { submitting: false })).toEqual({})
  })

  it('needs the end after the start, a real title and a real seat limit', () => {
    expect(checkProposal({ ...talk, endTime: '14:00' }, now, { submitting: false }).ends).toBe('It has to end after it starts.')
    expect(checkProposal({ ...talk, title: 'Hi' }, now, { submitting: false }).title).toBe('Use at least 3 characters.')
    expect(checkProposal({ ...talk, limitSeats: true, capacity: '0' }, now, { submitting: false }).capacity).toBe('Use a whole number from 1 up.')
    expect(checkProposal({ ...talk, limitSeats: true, capacity: '12.5' }, now, { submitting: false }).capacity).toBe('Use a whole number from 1 up.')
  })
})

describe('durationLabel', () => {
  it('says how long the event runs', () => {
    expect(durationLabel(talk)).toBe('1 h 30 min')
    expect(durationLabel({ ...talk, endTime: '16:30' })).toBe('2 h')
    expect(durationLabel({ ...talk, endDate: '2026-10-03', endTime: '14:30' })).toBe('24 hours, over two days')
    expect(durationLabel({ ...talk, endTime: '' })).toBe('')
  })
})

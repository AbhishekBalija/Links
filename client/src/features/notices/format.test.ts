import { describe, expect, it } from 'vitest'
import { audienceLabel, expiry, timeAgo } from './format'

const now = new Date(2026, 8, 28, 10, 0) // Monday 28 September, 10:00
const hoursAgo = (h: number) => new Date(now.getTime() - h * 3600_000).toISOString()

describe('timeAgo', () => {
  it.each([
    [0.005, 'just now'],
    [0.5, '30 min ago'],
    [1, '1 hour ago'],
    [5, '5 hours ago'],
    [30, 'yesterday'],
    [72, '3 days ago'],
  ])('%s hours ago reads "%s"', (hours, text) => {
    expect(timeAgo(hoursAgo(hours), now)).toBe(text)
  })

  it('shows the date once it is older than a week', () => {
    expect(timeAgo(new Date(2026, 8, 10).toISOString(), now)).toMatch(/10 Sept?/)
  })
})

describe('expiry', () => {
  const at = (days: number, hour: number) => new Date(2026, 8, 28 + days, hour).toISOString()

  it('is null without an expiry date', () => {
    expect(expiry(null, now)).toBeNull()
  })

  it.each([
    ['later today', at(0, 23), 'today', true],
    ['just after midnight', at(1, 0), 'tomorrow', true],
    ['25 hours away', at(1, 11), 'tomorrow', true],
    ['two days away', at(2, 9), 'in 2 days', true],
    ['a week away', at(7, 9), 'in 7 days', true],
    ['eight days away', at(8, 9), 'in 8 days', false],
  ])('counts calendar days: %s', (_, iso, text, soon) => {
    expect(expiry(iso, now)).toMatchObject({ text, soon })
  })
})

describe('audienceLabel', () => {
  const rule = (fields: Partial<{ department_code: string; batch_year: number; role: string }>) => ({
    department_id: fields.department_code ? 'id' : null,
    department_code: fields.department_code ?? null,
    batch_year: fields.batch_year ?? null,
    role: fields.role ?? null,
  })

  it.each([
    [[], 'Whole college'],
    [[rule({ department_code: 'CS', role: 'student' })], 'CS students'],
    [[rule({ department_code: 'CS', role: 'student', batch_year: 2023 })], 'CS students, batch 2023'],
    [[rule({ department_code: 'EC' })], 'Everyone in EC'],
    [[rule({ role: 'faculty' })], 'All faculty'],
    [[rule({ batch_year: 2024 })], 'Batch 2024'],
    [[rule({ department_code: 'CS', role: 'hod' }), rule({ role: 'placement_officer' })], 'CS HODs; All placement officers'],
  ])('%j reads "%s"', (rules, text) => {
    expect(audienceLabel(rules)).toBe(text)
  })
})

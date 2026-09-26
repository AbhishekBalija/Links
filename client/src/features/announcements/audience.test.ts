import { describe, expect, it } from 'vitest'
import { batchApplies, describe as describeAudience, isEmptyRule, matchPreset, presetsFor, toRules } from './audience'

const cs = { id: 'cs-id', code: 'CS' }

describe('presetsFor', () => {
  it('offers the author their own department first', () => {
    expect(presetsFor(cs).map((p) => p.label)).toEqual(['Whole college', 'Everyone in CS', 'CS students', 'CS faculty'])
  })

  it('falls back to college-wide groups without a department', () => {
    expect(presetsFor(null).map((p) => p.label)).toEqual(['Whole college', 'All students', 'All faculty'])
  })
})

describe('matchPreset', () => {
  const presets = presetsFor(cs)

  it('recognises a saved audience as its quick pick', () => {
    expect(matchPreset([{ department_id: 'cs-id', role: 'student' }], presets)).toBe('dept-students')
    expect(matchPreset([], presets)).toBe('college')
  })

  it('treats anything else as custom groups', () => {
    expect(matchPreset([{ department_id: 'cs-id', role: 'student', batch_year: 2023 }], presets)).toBeNull()
  })
})

describe('batchApplies', () => {
  it('only lets students have a batch', () => {
    expect(batchApplies('student')).toBe(true)
    expect(batchApplies('student_coordinator')).toBe(true)
    expect(batchApplies(undefined)).toBe(true)
    expect(batchApplies('faculty')).toBe(false)
    expect(batchApplies('hod')).toBe(false)
  })
})

describe('toRules and isEmptyRule', () => {
  it('drops the nulls the server sends for "anyone"', () => {
    expect(toRules([{ department_id: 'cs-id', department_code: 'CS', batch_year: null, role: 'student' }])).toEqual([
      { department_id: 'cs-id', role: 'student' },
    ])
  })

  it('spots a group with nothing chosen', () => {
    expect(isEmptyRule({})).toBe(true)
    expect(isEmptyRule({ role: 'faculty' })).toBe(false)
  })
})

describe('describe', () => {
  it('names departments by code', () => {
    const codes = new Map([['cs-id', 'CS']])
    expect(describeAudience([{ department_id: 'cs-id', role: 'student', batch_year: 2023 }], codes)).toBe('CS students, batch 2023')
  })
})

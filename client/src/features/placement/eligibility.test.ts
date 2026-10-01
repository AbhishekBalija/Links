import { describe, expect, it } from 'vitest'
import { describeEligibility, eligibilitySummary, fromRules, MAX_RULES, toRules } from './eligibility'

const departments = [
  { id: 'cs', code: 'CS', name: 'Computer Science' },
  { id: 'is', code: 'IS', name: 'Information Science' },
  { id: 'ec', code: 'EC', name: 'Electronics' },
]

describe('toRules', () => {
  it('gives every student one rule for the student role', () => {
    expect(toRules({ everyone: true, departments: [], batches: [] })).toEqual([{ role: 'student' }])
  })

  it('pairs each chosen department with each chosen batch, students only', () => {
    expect(toRules({ everyone: false, departments: ['cs', 'is'], batches: [2023] })).toEqual([
      { department_id: 'cs', batch_year: 2023, role: 'student' },
      { department_id: 'is', batch_year: 2023, role: 'student' },
    ])
  })

  it('leaves the department out when none is chosen, and the batch the same way', () => {
    expect(toRules({ everyone: false, departments: [], batches: [2023, 2024] })).toEqual([
      { batch_year: 2023, role: 'student' },
      { batch_year: 2024, role: 'student' },
    ])
    expect(toRules({ everyone: false, departments: ['ec'], batches: [] })).toEqual([{ department_id: 'ec', role: 'student' }])
  })
})

describe('fromRules', () => {
  it('reads back what toRules wrote', () => {
    const choice = { everyone: false, departments: ['cs', 'is'], batches: [2023, 2024] }
    expect(fromRules(toRules(choice))).toEqual(choice)
    expect(fromRules([{ role: 'student' }])).toEqual({ everyone: true, departments: [], batches: [] })
  })

  it('reads the server shape, where empty fields are null or missing', () => {
    expect(fromRules([{ department_id: 'cs', batch_year: null, role: 'student' }])).toEqual({ everyone: false, departments: ['cs'], batches: [] })
  })

  it('says it cannot show rules the chips could not have made', () => {
    expect(fromRules([{ department_id: 'cs', role: 'faculty' }])).toBeNull()
    expect(fromRules([{ department_id: 'cs', batch_year: 2023, role: 'student' }, { department_id: 'is', batch_year: 2024, role: 'student' }])).toBeNull()
  })
})

describe('eligibilitySummary', () => {
  it('puts the choice into plain words', () => {
    expect(eligibilitySummary({ everyone: true, departments: [], batches: [] }, departments)).toBe('every student')
    expect(eligibilitySummary({ everyone: false, departments: ['cs', 'is'], batches: [2023] }, departments)).toBe('CS and IS students, batch 2023')
    expect(eligibilitySummary({ everyone: false, departments: ['cs', 'is', 'ec'], batches: [] }, departments)).toBe('CS, EC and IS students, every batch')
    expect(eligibilitySummary({ everyone: false, departments: [], batches: [2023, 2024] }, departments)).toBe('students of every department, batches 2023 and 2024')
  })

  it('caps the combinations at what the server takes', () => {
    expect(MAX_RULES).toBe(20)
  })
})

describe('describeEligibility', () => {
  it('reads saved rules as one sentence, using the codes the server sends', () => {
    expect(
      describeEligibility([
        { department_id: 'cs', department_code: 'CS', batch_year: 2023, role: 'student' },
        { department_id: 'ad', department_code: 'AD', batch_year: 2023, role: 'student' },
      ]),
    ).toBe('AD and CS students, batch 2023')
    expect(describeEligibility([{ role: 'student', batch_year: 2023 }, { role: 'student', batch_year: 2024 }])).toBe(
      'Students of every department, batches 2023 and 2024',
    )
    expect(describeEligibility([{ role: 'student' }])).toBe('Every student')
    expect(describeEligibility([])).toBe('Everyone')
  })

  it('falls back to listing the rules when the chips could not have made them', () => {
    expect(describeEligibility([{ department_id: 'cs', department_code: 'CS', role: 'faculty' }])).toBe('CS faculty')
  })
})

import { describe, expect, it } from 'vitest'
import { applicantParams, applicantTabs, exportName, staffStatusLabel } from './applicants'

const counts = { total: 42, applied: 30, shortlisted: 8, selected: 2, rejected: 2, withdrawn: 1 }

describe('applicantTabs', () => {
  it('starts with All, without a number, then each status with its count', () => {
    expect(applicantTabs(counts)).toEqual([
      { value: null, label: 'All' },
      { value: 'applied', label: 'To review', count: 30 },
      { value: 'shortlisted', label: 'Shortlisted', count: 8 },
      { value: 'selected', label: 'Selected', count: 2 },
      { value: 'rejected', label: 'Rejected', count: 2 },
      { value: 'withdrawn', label: 'Withdrawn', count: 1 },
    ])
  })

  it('leaves the counts out while they load', () => {
    expect(applicantTabs(undefined)[1]).toEqual({ value: 'applied', label: 'To review' })
  })
})

describe('applicantParams', () => {
  it('sends only the filters that are set, trimming the search', () => {
    expect(applicantParams({ status: 'shortlisted', department: 'CS', batch: '2023', q: '  asha ' }, 'c1').toString()).toBe(
      'limit=50&status=shortlisted&department=CS&batch=2023&q=asha&cursor=c1',
    )
    expect(applicantParams({ status: null, department: '', batch: '', q: '' }, '').toString()).toBe('limit=50')
  })
})

describe('exportName', () => {
  it('names the file after the opportunity and the status exported', () => {
    expect(exportName('Graduate Engineer Trainee', 'Acme Systems', null)).toBe('acme-systems-graduate-engineer-trainee-applicants.csv')
    expect(exportName('Graduate Engineer Trainee', 'Acme Systems', 'shortlisted')).toBe('acme-systems-graduate-engineer-trainee-shortlisted.csv')
  })
})

describe('staffStatusLabel', () => {
  it('calls an applied one To review for the office', () => {
    expect(staffStatusLabel('applied')).toBe('To review')
    expect(staffStatusLabel('rejected')).toBe('Rejected')
  })
})

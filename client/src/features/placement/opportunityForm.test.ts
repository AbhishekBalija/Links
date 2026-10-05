import { describe, expect, it } from 'vitest'
import { checkOpportunity, emptyOpportunity, fromOpportunity, toInput, type OpportunityForm } from './opportunityForm'
import type { Opportunity } from '../jobs/types'
import { collegeInstant } from '../../shared/time/college'

const now = collegeInstant(2026, 9, 30, 10, 0)

const filled = (overrides: Partial<OpportunityForm> = {}): OpportunityForm => ({
  ...emptyOpportunity(),
  opportunity_type: 'job',
  title: 'Graduate Engineer Trainee',
  company: 'Acme Systems',
  applyDate: '2026-10-02',
  applyTime: '23:59',
  eligibility: { everyone: false, departments: ['cs'], batches: [2023] },
  ...overrides,
})

describe('checkOpportunity', () => {
  it('passes a complete opportunity', () => {
    expect(checkOpportunity(filled(), now, { publishing: true })).toEqual({})
  })

  it('asks for the type, role, company and deadline in words', () => {
    const errors = checkOpportunity(emptyOpportunity(), now, { publishing: false })
    expect(errors).toMatchObject({
      opportunity_type: 'Choose a job, internship or training.',
      title: 'Write the role.',
      company: "Add the company's name.",
      apply_by: 'Choose the last day to apply.',
    })
  })

  it('lets a draft keep a past deadline but not a published one', () => {
    const past = filled({ applyDate: '2026-09-28' })
    expect(checkOpportunity(past, now, { publishing: false }).apply_by).toBeUndefined()
    expect(checkOpportunity(past, now, { publishing: true }).apply_by).toBe('Pick a time after now.')
  })

  it('needs a full https or http link when students apply on the company site', () => {
    expect(checkOpportunity(filled({ application_mode: 'external', external_url: '' }), now, { publishing: false }).external_url).toBe(
      'Add the full link students apply at.',
    )
    expect(checkOpportunity(filled({ application_mode: 'external', external_url: 'careers.acme.in' }), now, { publishing: false }).external_url).toBe(
      'Start the link with https://.',
    )
    expect(checkOpportunity(filled({ application_mode: 'external', external_url: 'https://careers.acme.in' }), now, { publishing: false })).toEqual({})
  })

  it('refuses more department and batch combinations than the server takes', () => {
    const many = filled({ eligibility: { everyone: false, departments: ['a', 'b', 'c', 'd', 'e', 'f'], batches: [2023, 2024, 2025, 2026] } })
    expect(checkOpportunity(many, now, { publishing: false }).eligibility).toBe('That makes 24 groups; pick at most 20, or choose All departments.')
  })
})

describe('toInput and fromOpportunity', () => {
  it('sends the deadline in UTC, clears empty optional fields, and drops the link for In LINKS', () => {
    const input = toInput(filled({ location: '  ', external_url: 'https://left.over' }))
    expect(input).toMatchObject({
      opportunity_type: 'job',
      title: 'Graduate Engineer Trainee',
      company: 'Acme Systems',
      location: null,
      compensation: null,
      application_mode: 'internal',
      external_url: null,
      apply_by: collegeInstant(2026, 10, 2, 23, 59).toISOString(),
      eligibility: [{ department_id: 'cs', batch_year: 2023, role: 'student' }],
    })
  })

  it('opens a saved opportunity in the form', () => {
    const saved = {
      opportunity_type: 'internship',
      title: 'Data analyst intern',
      company: 'Kaveri Analytics',
      description: 'Ten weeks.',
      location: 'Bengaluru',
      compensation: null,
      apply_by: collegeInstant(2026, 10, 3, 17, 0).toISOString(),
      application_mode: 'external',
      external_url: 'https://kaveri.in/apply',
      eligibility: [{ batch_year: 2024, role: 'student' }],
    } as unknown as Opportunity
    expect(fromOpportunity(saved)).toMatchObject({
      opportunity_type: 'internship',
      location: 'Bengaluru',
      compensation: '',
      applyDate: '2026-10-03',
      applyTime: '17:00',
      application_mode: 'external',
      external_url: 'https://kaveri.in/apply',
      eligibility: { everyone: false, departments: [], batches: [2024] },
    })
  })
})

import { describe, expect, it } from 'vitest'
import { jobAction, studentStatus } from './standing'
import type { MyApplication, Opportunity } from './types'

const job = (overrides: Partial<Opportunity> = {}): Pick<Opportunity, 'open' | 'application_mode' | 'my_application'> => ({
  open: true,
  application_mode: 'internal',
  my_application: null,
  ...overrides,
})

const application = (overrides: Partial<MyApplication> = {}): MyApplication => ({
  id: 'a1',
  opportunity_id: 'o1',
  mode: 'internal',
  status: 'applied',
  applied_at: '2026-09-30T05:12:00Z',
  withdrawn_at: null,
  ...overrides,
})

describe('jobAction', () => {
  it('offers to apply while it is open and the student has not applied', () => {
    expect(jobAction(job())).toBe('apply')
    expect(jobAction(job({ application_mode: 'external' }))).toBe('apply-external')
  })

  it('says it is closed when nobody applied before it closed', () => {
    expect(jobAction(job({ open: false }))).toBe('closed')
  })

  it("follows the student's own application once there is one, open or closed", () => {
    expect(jobAction(job({ my_application: application() }))).toBe('applied')
    expect(jobAction(job({ application_mode: 'external', my_application: application({ mode: 'external' }) }))).toBe('applied-external')
    expect(jobAction(job({ open: false, my_application: application({ status: 'shortlisted' }) }))).toBe('shortlisted')
    expect(jobAction(job({ my_application: application({ status: 'selected' }) }))).toBe('selected')
    expect(jobAction(job({ my_application: application({ status: 'rejected' }) }))).toBe('rejected')
    expect(jobAction(job({ my_application: application({ status: 'withdrawn', withdrawn_at: '2026-09-29T10:00:00Z' }) }))).toBe('withdrawn')
  })
})

describe('studentStatus', () => {
  it('words each status for the student, gently for a rejection', () => {
    expect(studentStatus(application())).toEqual({ label: 'Applied', tone: 'plain' })
    expect(studentStatus(application({ mode: 'external' }))).toEqual({ label: 'Applied on their site', tone: 'plain' })
    expect(studentStatus(application({ status: 'shortlisted' }))).toEqual({ label: 'Shortlisted', tone: 'warning' })
    expect(studentStatus(application({ status: 'selected' }))).toEqual({ label: 'Selected', tone: 'going' })
    expect(studentStatus(application({ status: 'rejected' }))).toEqual({ label: 'Not selected', tone: 'cancelled' })
    expect(studentStatus(application({ status: 'withdrawn' }))).toEqual({ label: 'Withdrawn', tone: 'outline' })
  })
})

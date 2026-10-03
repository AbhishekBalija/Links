import { describe, expect, it } from 'vitest'
import type { Dashboard } from './api'
import { homeKind, summaryLine } from './roleHome'

const base: Dashboard = {
  user: { full_name: 'Asha Rao', roles: ['hod'], department: { id: 'd1', code: 'CS', name: 'Computer Science' } },
  notices: { items: [], has_more: false },
}
const approvals = (announcements: number, events: number) => ({
  pending_count: announcements,
  oldest_submitted_at: null,
  events_pending_count: events,
  oldest_event_submitted_at: null,
})
const department = (code: string, name: string, hod: boolean) => ({
  code,
  name,
  students: 10,
  staff: 2,
  hod: hod ? { full_name: 'X', username: 'x' } : null,
})

describe('homeKind', () => {
  it('gives each role its own Home, the most senior first', () => {
    expect(homeKind(['admin', 'principal'])).toBe('admin')
    expect(homeKind(['principal'])).toBe('principal')
    expect(homeKind(['faculty', 'hod'])).toBe('hod')
    expect(homeKind(['student'])).toBe('everyone')
  })
})

describe('summaryLine', () => {
  it("tells an HOD who is waiting to get in and what needs their approval", () => {
    const data = { ...base, access_requests: { pending_count: 2, oldest_requested_at: null }, approvals: approvals(2, 1) }
    expect(summaryLine('hod', data)).toBe('Two students are waiting to get in, and three posts need your approval.')
  })

  it('reads one of each in the singular', () => {
    const data = { ...base, access_requests: { pending_count: 1, oldest_requested_at: null }, approvals: approvals(1, 0) }
    expect(summaryLine('hod', data)).toBe('One student is waiting to get in, and one post needs your approval.')
  })

  it('says so when nothing is waiting', () => {
    expect(summaryLine('hod', { ...base, access_requests: { pending_count: 0, oldest_requested_at: null }, approvals: approvals(0, 0) })).toBe(
      'Nothing is waiting for you.',
    )
  })

  it("tells the principal about final approvals and departments with no HOD", () => {
    const data = {
      ...base,
      approvals: approvals(0, 2),
      college: { departments: [department('CS', 'Computer Science', true), department('EC', 'Electronics and Communication', false)], departments_without_hod: 1 },
    }
    expect(summaryLine('principal', data)).toBe('Two events are waiting for your approval. Electronics and Communication has no HOD.')
  })

  it('puts events and announcements in one sentence', () => {
    const data = { ...base, approvals: approvals(1, 1), college: { departments: [department('CS', 'CS', true)], departments_without_hod: 0 } }
    expect(summaryLine('principal', data)).toBe('One event and one announcement are waiting for your approval.')
  })

  it('names how many departments have no HOD when there are several', () => {
    const data = {
      ...base,
      approvals: approvals(0, 0),
      college: { departments: [department('EC', 'EC', false), department('ME', 'ME', false)], departments_without_hod: 2 },
    }
    expect(summaryLine('principal', data)).toBe('Two departments have no HOD.')
  })

  it('tells an admin who is waiting to get in', () => {
    expect(summaryLine('admin', { ...base, access_requests: { pending_count: 3, oldest_requested_at: null } })).toBe('Three people are waiting to get in.')
  })
})

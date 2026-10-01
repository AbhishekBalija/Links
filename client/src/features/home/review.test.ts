import { describe, expect, it } from 'vitest'
import { waitingForReview } from './review'

describe('waitingForReview', () => {
  it('counts announcements and event proposals together', () => {
    expect(waitingForReview({ pending_count: 2, oldest_submitted_at: null, events_pending_count: 3, oldest_event_submitted_at: null })).toBe(5)
  })

  it('counts event proposals when no announcement waits', () => {
    expect(waitingForReview({ pending_count: 0, oldest_submitted_at: null, events_pending_count: 1, oldest_event_submitted_at: null })).toBe(1)
  })

  it('is zero for someone who reviews nothing', () => {
    expect(waitingForReview(undefined)).toBe(0)
  })
})

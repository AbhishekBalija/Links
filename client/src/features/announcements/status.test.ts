import { describe, expect, it } from 'vitest'
import { approverPhrase, hasExpired, hrefFor, standing, waited } from './status'
import type { Authored } from './types'

const now = new Date(2026, 8, 28, 10, 0)
const iso = (daysFromNow: number) => new Date(now.getTime() + daysFromNow * 86_400_000).toISOString()

function item(fields: Partial<Authored>): Authored {
  return {
    id: 'a1',
    title: 'Lab timings',
    body: 'Details.',
    category: 'department',
    status: 'published',
    publisher_id: 'u1',
    publisher_name: 'Meera N',
    audience: [],
    published_at: iso(-2),
    expires_at: null,
    created_at: iso(-3),
    ...fields,
  }
}

describe('standing', () => {
  it('opens drafts and sent-back ones in the composer', () => {
    expect(standing(item({ status: 'draft' }), now)).toMatchObject({ tag: 'Draft', opensInComposer: true, needsAuthor: false })
    expect(standing(item({ status: 'rejected' }), now)).toMatchObject({ tag: 'Sent back', opensInComposer: true, needsAuthor: true })
  })

  it('names who a waiting one is with', () => {
    expect(standing(item({ status: 'pending', approver: 'CS HOD' }), now).tag).toBe('With CS HOD')
  })

  it('calls a published one past its expiry "Expired", not "Live"', () => {
    expect(standing(item({ expires_at: iso(-1) }), now)).toMatchObject({ tag: 'Expired', expired: true })
    expect(standing(item({ expires_at: iso(3) }), now)).toMatchObject({ tag: 'Live', expired: false })
  })

  it('needs the author when a live one has an edit sent back', () => {
    const edit = { status: 'rejected' as const, title: 't', body: 'b', category: 'department' as const, audience: [], expires_at: null }
    expect(standing(item({ edit }), now).needsAuthor).toBe(true)
  })
})

describe('hrefFor', () => {
  it('sends drafts to the composer and the rest to the read view', () => {
    expect(hrefFor(item({ status: 'draft' }), now)).toBe('/mine/a1/edit')
    expect(hrefFor(item({}), now)).toBe('/mine/a1')
  })
})

describe('approverPhrase', () => {
  it('reads naturally in a sentence', () => {
    expect(approverPhrase('CS HOD')).toBe('the CS HOD')
    expect(approverPhrase('Principal or admin')).toBe('the principal or an admin')
    expect(approverPhrase(undefined)).toBe('the approver')
  })
})

describe('waited', () => {
  const hoursAgo = (h: number) => new Date(now.getTime() - h * 3600_000).toISOString()

  it.each([
    [1, 'waiting 1 hour', false],
    [5, 'waiting 5 hours', false],
    [30, 'waiting 1 day', false],
    [72, 'waiting 3 days', false],
    [73, 'waiting 3 days', true],
    [100, 'waiting 4 days', true],
  ])('%s hours reads "%s" (long: %s)', (hours, text, long) => {
    expect(waited(hoursAgo(hours), now)).toEqual({ text, long })
  })
})

describe('hasExpired', () => {
  it('is true only once the expiry date has passed', () => {
    expect(hasExpired({ expires_at: null }, now)).toBe(false)
    expect(hasExpired({ expires_at: iso(1) }, now)).toBe(false)
    expect(hasExpired({ expires_at: iso(-1) }, now)).toBe(true)
  })
})

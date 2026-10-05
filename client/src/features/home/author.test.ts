import { describe, expect, it } from 'vitest'
import { homeKind } from './roleHome'
import { authorLine, sentBackOnHome, workDetail, workHref, type MyWork } from './author'

const sentBack = (n: number): MyWork['sent_back'] =>
  Array.from({ length: n }, (_, i) => ({ kind: 'announcement', id: `a${i}`, title: `Post ${i}`, note: 'Fix it', sent_back_by: 'Asha Rao', sent_back_at: '2026-10-04T10:00:00Z', is_edit: false }))
const waiting = (n: number): MyWork['waiting'] =>
  Array.from({ length: n }, (_, i) => ({ kind: 'event', id: `e${i}`, title: `Event ${i}`, waiting_on: 'CS HOD', since: '2026-10-03T10:00:00Z', is_edit: false }))
const work = (back: number, wait: number, more = false): MyWork => ({ sent_back: sentBack(back), sent_back_has_more: more, waiting: waiting(wait), waiting_has_more: false })

describe('homeKind for authors', () => {
  it('gives faculty their own Home, unless a senior role or placement work comes first', () => {
    expect(homeKind(['faculty'])).toBe('faculty')
    expect(homeKind(['faculty', 'hod'])).toBe('hod')
    expect(homeKind(['faculty', 'placement_officer'])).toBe('everyone')
  })

  it('keeps student coordinators on the student Home', () => {
    expect(homeKind(['student', 'student_coordinator'])).toBe('everyone')
  })
})

describe('authorLine', () => {
  it('says what came back and what waits', () => {
    expect(authorLine(work(1, 2))).toBe('One post was sent back with a note. Two are waiting on others.')
    expect(authorLine(work(2, 0))).toBe('Two posts were sent back with a note.')
    expect(authorLine(work(0, 1))).toBe('One post is waiting on others.')
  })

  it('is calm when nothing is open', () => {
    expect(authorLine(work(0, 0))).toBe('Nothing of yours is waiting on anyone.')
    expect(authorLine(undefined)).toBe('Nothing of yours is waiting on anyone.')
  })
})

describe('sentBackOnHome', () => {
  it('shows three with their notes and counts the rest', () => {
    expect(sentBackOnHome(work(2, 0))).toEqual({ shown: sentBack(2), more: 0, orMore: false })
    const five = sentBackOnHome(work(5, 0))
    expect(five.shown).toHaveLength(3)
    expect(five.more).toBe(2)
  })

  it('says "or more" when the server cut the list', () => {
    expect(sentBackOnHome(work(10, 0, true))).toMatchObject({ more: 7, orMore: true })
  })
})

describe('work rows', () => {
  it('links each kind to its page in My posts', () => {
    expect(workHref({ kind: 'announcement', id: 'a1' })).toBe('/mine/a1')
    expect(workHref({ kind: 'event', id: 'e1' })).toBe('/mine/events/e1')
  })

  it('names the kind and who it waits on', () => {
    expect(workDetail(waiting(1)[0])).toBe('Event · With the CS HOD')
    expect(workDetail({ kind: 'announcement', waiting_on: 'Principal or admin', is_edit: true })).toBe('Edit to a live announcement · With the principal or an admin')
  })
})

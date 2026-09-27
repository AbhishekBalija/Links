import { describe, expect, it } from 'vitest'
import { mergeNewestFirst } from './merge'

const item = (id: string, day: number) => ({ id, at: new Date(2026, 8, day).toISOString() })

describe('mergeNewestFirst', () => {
  it('interleaves finished lists, newest first', () => {
    const merged = mergeNewestFirst([
      { items: [item('a1', 20), item('a2', 10)], complete: true },
      { items: [item('e1', 25), item('e2', 15)], complete: true },
    ])
    expect(merged.items.map((i) => i.id)).toEqual(['e1', 'a1', 'e2', 'a2'])
    expect(merged.hasMore).toBe(false)
  })

  it('holds back anything older than the last loaded item of a list with more pages', () => {
    // Events may still have posts from day 17 or 16 that belong before a2.
    const merged = mergeNewestFirst([
      { items: [item('a1', 20), item('a2', 10)], complete: true },
      { items: [item('e1', 25), item('e2', 18)], complete: false },
    ])
    expect(merged.items.map((i) => i.id)).toEqual(['e1', 'a1', 'e2'])
    expect(merged.hasMore).toBe(true)
  })

  it('shows nothing it cannot place while a list has more pages', () => {
    const merged = mergeNewestFirst([
      { items: [item('a1', 20)], complete: false },
      { items: [item('e1', 25), item('e2', 22)], complete: false },
    ])
    expect(merged.items.map((i) => i.id)).toEqual(['e1', 'e2'])
  })
})

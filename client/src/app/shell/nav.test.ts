import { describe, expect, it } from 'vitest'
import { navFor, tabsFor } from './nav'

const paths = (items: { to: string }[]) => items.map((item) => item.to)

describe('navigation', () => {
  it('gives students Home, Notices, Events, People and Profile everywhere', () => {
    expect(paths(navFor(['student']))).toEqual(['/', '/notices', '/events', '/people', '/profile'])
    expect(paths(tabsFor(['student']))).toEqual(['/', '/notices', '/events', '/people', '/profile'])
  })

  it('swaps Profile for Mine on the phone tabs for faculty, who can post', () => {
    expect(paths(tabsFor(['faculty']))).toEqual(['/', '/notices', '/events', '/people', '/mine'])
  })

  it('keeps an HOD on their work: Profile and then People leave the phone tabs', () => {
    expect(paths(navFor(['hod', 'faculty']))).toHaveLength(7)
    expect(paths(tabsFor(['hod', 'faculty']))).toEqual(['/', '/notices', '/events', '/approvals', '/mine'])
  })
})

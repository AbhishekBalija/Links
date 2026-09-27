import { describe, expect, it } from 'vitest'
import { navFor, tabsFor } from './nav'

const paths = (items: { to: string }[]) => items.map((item) => item.to)

describe('navigation', () => {
  it('gives students Home, Notices, People and Profile everywhere', () => {
    expect(paths(navFor(['student']))).toEqual(['/', '/notices', '/people', '/profile'])
    expect(paths(tabsFor(['student']))).toEqual(['/', '/notices', '/people', '/profile'])
  })

  it('keeps all five tabs for faculty, who can post', () => {
    expect(paths(tabsFor(['faculty']))).toEqual(['/', '/notices', '/people', '/mine', '/profile'])
  })

  it('drops Profile from the phone tabs when an HOD would have six', () => {
    expect(paths(navFor(['hod', 'faculty']))).toHaveLength(6)
    expect(paths(tabsFor(['hod', 'faculty']))).toEqual(['/', '/notices', '/people', '/approvals', '/mine'])
  })
})

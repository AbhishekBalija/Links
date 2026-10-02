import { describe, expect, it } from 'vitest'
import { hidesTabBar, navFor, tabsFor } from './nav'

const paths = (items: { to: string }[]) => items.map((item) => item.to)

describe('navigation', () => {
  it('gives students Jobs, and on the phone Jobs takes People\'s tab so Profile stays', () => {
    expect(paths(navFor(['student']))).toEqual(['/', '/notices', '/events', '/jobs', '/people', '/profile'])
    expect(paths(tabsFor(['student']))).toEqual(['/', '/notices', '/events', '/jobs', '/profile'])
  })

  it('keeps Jobs to students', () => {
    expect(paths(navFor(['faculty']))).not.toContain('/jobs')
  })

  it('swaps Profile for Mine on the phone tabs for faculty, who can post', () => {
    expect(paths(tabsFor(['faculty']))).toEqual(['/', '/notices', '/events', '/people', '/mine'])
  })

  it('keeps an HOD on their work: Profile and then People leave the phone tabs', () => {
    expect(paths(navFor(['hod', 'faculty']))).toHaveLength(7)
    expect(paths(tabsFor(['hod', 'faculty']))).toEqual(['/', '/notices', '/events', '/approvals', '/mine'])
  })

  it('gives placement staff Placement, and keeps it on the placement officer\'s phone tabs', () => {
    expect(paths(navFor(['placement_officer']))).toContain('/placement')
    expect(paths(tabsFor(['placement_officer']))).toEqual(['/', '/notices', '/events', '/placement', '/mine'])
    expect(paths(navFor(['student']))).not.toContain('/placement')
  })

  it('keeps Placement in the principal\'s sidebar but off their five phone tabs', () => {
    expect(paths(navFor(['principal']))).toContain('/placement')
    expect(paths(tabsFor(['principal']))).toEqual(['/', '/notices', '/events', '/approvals', '/mine'])
  })
})

describe('the Admin workspace', () => {
  it('is in the sidebar for admins only, not the principal or HODs', () => {
    expect(navFor(['admin']).map((item) => item.to)).toContain('/admin')
    expect(navFor(['principal']).map((item) => item.to)).not.toContain('/admin')
    expect(navFor(['hod']).map((item) => item.to)).not.toContain('/admin')
  })

  it("steps out of an admin's phone tabs, which keep their work tabs", () => {
    expect(tabsFor(['admin']).map((item) => item.to)).toEqual(['/', '/notices', '/events', '/approvals', '/mine'])
  })
})

describe('hidesTabBar', () => {
  it('hides the phone tab bar on a page with its own action bar', () => {
    for (const path of ['/mine/abc', '/approvals/abc', '/approvals/access/abc', '/admin/requests/abc', '/events/abc', '/jobs/abc', '/placement/abc']) {
      expect(hidesTabBar(path), path).toBe(true)
    }
  })

  it('keeps it on lists, including the Access requests list', () => {
    for (const path of ['/', '/mine', '/approvals', '/approvals/access', '/admin/requests', '/events', '/people/someone']) {
      expect(hidesTabBar(path), path).toBe(false)
    }
  })
})

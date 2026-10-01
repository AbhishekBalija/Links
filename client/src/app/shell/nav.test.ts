import { describe, expect, it } from 'vitest'
import { navFor, tabsFor } from './nav'

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

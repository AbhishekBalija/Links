import { describe, expect, it } from 'vitest'
import { safeReturn } from './returnTo'
import { cleanCode } from './signIn'

describe('safeReturn', () => {
  it('keeps a page on this site, with its query', () => {
    expect(safeReturn('/jobs/abc')).toBe('/jobs/abc')
    expect(safeReturn('/events?when=past')).toBe('/events?when=past')
  })

  it('refuses anything that could leave the site or loop back to sign-in', () => {
    for (const bad of [null, '', 'jobs', 'https://evil.example/x', '//evil.example/x', '/\\evil.example', '/login', '/welcome', '/', '/account-pending']) {
      expect(safeReturn(bad)).toBeNull()
    }
  })
})

describe('cleanCode', () => {
  it('reads a code out of a pasted line of text', () => {
    expect(cleanCode('Your LINKS sign-in code: 482 913')).toBe('482913')
  })
})

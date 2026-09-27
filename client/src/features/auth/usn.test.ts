import { describe, expect, it } from 'vitest'
import { isUSNFormat } from './usn'

describe('isUSNFormat', () => {
  it.each(['4MN22CS001', '4mn24is042', '4MN20MB002'])('accepts %s, whatever the department code', (usn) => {
    expect(isUSNFormat(usn)).toBe(true)
  })

  it.each(['', '4MN22C5001', '4MN22CS01', '5MN22CS001', '4MN22CS0011'])('rejects %s', (usn) => {
    expect(isUSNFormat(usn)).toBe(false)
  })
})

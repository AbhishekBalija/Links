import { describe, expect, it } from 'vitest'
import { greeting, nextGreetingChange } from './greeting'

const at = (hour: number, minute = 0) => new Date(2026, 9, 4, hour, minute)

describe('greeting', () => {
  it('follows the time of day', () => {
    expect(greeting(at(9))).toBe('Good morning')
    expect(greeting(at(13, 30))).toBe('Good afternoon')
    expect(greeting(at(19))).toBe('Good evening')
  })

  it('says good evening after midnight, not good morning', () => {
    expect(greeting(at(0, 25))).toBe('Good evening')
    expect(greeting(at(4, 59))).toBe('Good evening')
    expect(greeting(at(5))).toBe('Good morning')
  })
})

describe('nextGreetingChange', () => {
  it('is the next hour the greeting changes at', () => {
    expect(nextGreetingChange(at(11, 40))).toEqual(at(12))
    expect(nextGreetingChange(at(16, 59))).toEqual(at(17))
    expect(nextGreetingChange(at(0, 25))).toEqual(at(5))
  })

  it('rolls over to tomorrow morning in the evening', () => {
    expect(nextGreetingChange(at(21))).toEqual(new Date(2026, 9, 5, 5, 0))
  })
})

import { describe, expect, it } from 'vitest'
import { ApiRequestError } from '../../shared/api/types'
import { cleanCode, codeState, greetingName, isEmail, limitText, mailLinks, outcomeOf, readUSN, resendIn } from './signIn'

describe('isEmail', () => {
  it.each(['asha.rao@gmail.com', ' kiran@college.edu.in ', 'a+b@x.io'])('accepts %s', (email) => {
    expect(isEmail(email)).toBe(true)
  })

  it.each(['', 'asha.rao@gmail', 'asha.rao', '@gmail.com', 'a b@x.io'])('rejects %s', (email) => {
    expect(isEmail(email)).toBe(false)
  })
})

describe('cleanCode', () => {
  it('keeps only the first six digits', () => {
    expect(cleanCode('48 29-13x7')).toBe('482913')
  })

  it('keeps a pasted code from an email', () => {
    expect(cleanCode('Your code: 004211')).toBe('004211')
  })
})

describe('mailLinks', () => {
  it('shows only Gmail for a Gmail address', () => {
    expect(mailLinks('asha.rao@gmail.com')).toEqual(['gmail'])
  })

  it('shows only Outlook for a Microsoft address', () => {
    expect(mailLinks('rohan.s@outlook.com')).toEqual(['outlook'])
    expect(mailLinks('meera@hotmail.com')).toEqual(['outlook'])
  })

  it('shows both for any other address, such as a college one', () => {
    expect(mailLinks('kiran.hegde@college.edu')).toEqual(['gmail', 'outlook'])
  })
})

describe('outcomeOf', () => {
  it('reads NOT_ON_LIST with what prefills the request', () => {
    const error = new ApiRequestError(403, {
      code: 'NOT_ON_LIST',
      message: "this email isn't on any list for LINKS yet",
      details: { email: 'asha.rao@gmail.com', full_name: 'Asha Rao', request_token: 'tok' },
    })
    expect(outcomeOf(error)).toEqual({ kind: 'not-on-list', email: 'asha.rao@gmail.com', fullName: 'Asha Rao', requestToken: 'tok' })
  })

  it('tells a waiting request from a refused one and a suspended account', () => {
    const notActive = (status: string) => new ApiRequestError(403, { code: 'ACCOUNT_NOT_ACTIVE', message: '', details: { status } })
    expect(outcomeOf(notActive('pending'))).toEqual({ kind: 'waiting' })
    expect(outcomeOf(notActive('rejected'))).toEqual({ kind: 'declined' })
    expect(outcomeOf(notActive('suspended'))).toEqual({ kind: 'suspended' })
  })

  it('reads a rate limit with the server message and which limit it was', () => {
    const error = new ApiRequestError(429, { code: 'RATE_LIMITED', message: 'too many codes asked for; try again in 15 minutes', details: { limit: 'email' } })
    expect(outcomeOf(error)).toEqual({ kind: 'limit', message: 'too many codes asked for; try again in 15 minutes', by: 'email' })
    const network = new ApiRequestError(429, { code: 'RATE_LIMITED', message: 'too many codes asked for from this network; try again in 15 minutes', details: { limit: 'network' } })
    expect(outcomeOf(network)).toMatchObject({ kind: 'limit', by: 'network' })
    // An older server sends no details: treat it as the email's limit.
    expect(outcomeOf(new ApiRequestError(429, { code: 'RATE_LIMITED', message: 'x; try again later' }))).toMatchObject({ by: 'email' })
  })

  it('words the limits apart (#202)', () => {
    const email = { kind: 'limit', message: 'too many codes asked for today; try again tomorrow', by: 'email' } as const
    const network = { kind: 'limit', message: 'too many codes asked for from this network; try again in 15 minutes', by: 'network' } as const
    expect(limitText(email, { google: true })).toBe('Too many codes asked for this email. Try again tomorrow, or use Continue with Google.')
    expect(limitText(email, { google: false })).toBe('Too many codes asked for this email. Try again tomorrow.')
    expect(limitText(network, { google: true })).toBe('Too many codes asked for from this network. Lots of people here are signing in at once. Try again in 15 minutes, or switch to mobile data.')
  })

  it('treats a refused code or Google token as a refusal', () => {
    expect(outcomeOf(new ApiRequestError(401, { code: 'UNAUTHENTICATED', message: 'the code is wrong or has expired' }))).toEqual({ kind: 'refused' })
  })

  it('treats anything else, like a network failure, as an error', () => {
    expect(outcomeOf(new TypeError('Failed to fetch'))).toEqual({ kind: 'error' })
    expect(outcomeOf(new ApiRequestError(500, { code: 'INTERNAL_ERROR', message: 'boom' }))).toEqual({ kind: 'error' })
  })
})

describe('codeState', () => {
  const sentAt = 1_000_000

  it('is live inside ten minutes with tries left', () => {
    expect(codeState({ sentAt, wrongTries: 4 }, sentAt + 9 * 60_000)).toBe('live')
  })

  it('stops working after ten minutes', () => {
    expect(codeState({ sentAt, wrongTries: 0 }, sentAt + 10 * 60_000)).toBe('dead')
  })

  it('stops working after five wrong tries', () => {
    expect(codeState({ sentAt, wrongTries: 5 }, sentAt + 60_000)).toBe('dead')
  })
})

describe('resendIn', () => {
  it('counts down a minute from when the code was sent', () => {
    expect(resendIn(1_000_000, 1_000_000 + 18_000)).toBe('0:42')
  })

  it('is empty once a new code can be asked for', () => {
    expect(resendIn(1_000_000, 1_000_000 + 61_000)).toBe('')
  })
})

describe('readUSN', () => {
  const departments = [{ code: 'CS', name: 'Computer Science and Engineering' }, { code: 'IS', name: 'Information Science and Engineering' }]

  it('splits a USN and names its Department and Batch', () => {
    expect(readUSN(' 4mn23cs042 ', departments)).toEqual({
      ok: true,
      usn: '4MN23CS042',
      parts: { college: '4MN', batch: '23', department: 'CS', roll: '042' },
      batchYear: 2023,
      departmentName: 'Computer Science and Engineering',
    })
  })

  it('says when no Department has the code', () => {
    expect(readUSN('4MN23XX042', departments)).toEqual({ ok: false, reason: 'unknown-department', code: 'XX' })
  })

  it('says when the USN has the wrong shape', () => {
    expect(readUSN('4MN23CS42', departments)).toEqual({ ok: false, reason: 'format' })
  })

  it('waits while the USN is still being typed', () => {
    expect(readUSN('', departments)).toEqual({ ok: false, reason: 'empty' })
  })
})

describe('greetingName', () => {
  it('greets by first name', () => {
    expect(greetingName('Asha Rao')).toBe('Asha')
  })

  it('skips a title such as Prof. or Dr.', () => {
    expect(greetingName('Prof. Kiran Hegde')).toBe('Kiran')
    expect(greetingName('Dr Suresh Kumar')).toBe('Suresh')
  })

  it('keeps a one-word name', () => {
    expect(greetingName('Meghana')).toBe('Meghana')
  })
})

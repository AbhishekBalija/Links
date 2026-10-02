import { ApiRequestError } from '../../shared/api/types'
import { isUSNFormat } from './usn'
import type { PublicDepartment } from './types'

// The rules behind the sign-in screens (spec #129), kept apart from the
// screens so they can be tested on their own.

const emailFormat = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function isEmail(email: string): boolean {
  return emailFormat.test(email.trim())
}

// cleanCode keeps the first six digits, so a code pasted with spaces or a
// "Your code:" prefix still works.
export function cleanCode(input: string): string {
  return input.replace(/\D/g, '').slice(0, 6)
}

export type MailLink = 'gmail' | 'outlook'

// mailLinks says which "Open your mail" links to offer: only the one that
// matches the address when we can tell, both otherwise.
export function mailLinks(email: string): MailLink[] {
  const domain = email.trim().toLowerCase().split('@')[1] ?? ''
  if (domain === 'gmail.com' || domain === 'googlemail.com') return ['gmail']
  if (['outlook.com', 'hotmail.com', 'live.com', 'msn.com'].includes(domain)) return ['outlook']
  return ['gmail', 'outlook']
}

export type Outcome =
  | { kind: 'not-on-list'; email: string; fullName: string; requestToken: string }
  | { kind: 'waiting' }
  | { kind: 'declined' }
  | { kind: 'suspended' }
  | { kind: 'limit'; message: string }
  | { kind: 'refused' }
  | { kind: 'error' }

// outcomeOf turns a failed sign-in into the screen to show next.
export function outcomeOf(error: unknown): Outcome {
  if (!(error instanceof ApiRequestError)) return { kind: 'error' }
  const details = error.details ?? {}
  if (error.code === 'NOT_ON_LIST') {
    return {
      kind: 'not-on-list',
      email: String(details.email ?? ''),
      fullName: String(details.full_name ?? ''),
      requestToken: String(details.request_token ?? ''),
    }
  }
  if (error.code === 'ACCOUNT_NOT_ACTIVE') {
    if (details.status === 'pending') return { kind: 'waiting' }
    if (details.status === 'rejected') return { kind: 'declined' }
    return { kind: 'suspended' }
  }
  if (error.status === 429) return { kind: 'limit', message: error.message }
  if (error.status === 401) return { kind: 'refused' }
  return { kind: 'error' }
}

const codeLifetime = 10 * 60_000
const maxWrongTries = 5
const resendWait = 60_000

// codeState mirrors the server's limits, so the screen can say a code has
// stopped working before anyone types into a dead one.
export function codeState(code: { sentAt: number; wrongTries: number }, now: number): 'live' | 'dead' {
  if (now - code.sentAt >= codeLifetime || code.wrongTries >= maxWrongTries) return 'dead'
  return 'live'
}

// resendIn is the "new code in 0:42" countdown, empty once a new code can be
// asked for.
export function resendIn(sentAt: number, now: number): string {
  const left = Math.ceil((sentAt + resendWait - now) / 1000)
  if (left <= 0) return ''
  return `${Math.floor(left / 60)}:${String(left % 60).padStart(2, '0')}`
}

export type USNReading =
  | {
      ok: true
      usn: string
      parts: { college: string; batch: string; department: string; roll: string }
      batchYear: number
      departmentName: string
    }
  | { ok: false; reason: 'empty' | 'format' }
  | { ok: false; reason: 'unknown-department'; code: string }

// readUSN splits a USN the way the request form shows it back:
// 4MN · 23 · CS · 042 is Computer Science, batch 2023.
export function readUSN(input: string, departments: PublicDepartment[]): USNReading {
  const usn = input.trim().toUpperCase()
  if (!usn) return { ok: false, reason: 'empty' }
  if (!isUSNFormat(usn)) return { ok: false, reason: 'format' }
  const parts = { college: usn.slice(0, 3), batch: usn.slice(3, 5), department: usn.slice(5, 7), roll: usn.slice(7) }
  const department = departments.find((d) => d.code === parts.department)
  if (!department) return { ok: false, reason: 'unknown-department', code: parts.department }
  return { ok: true, usn, parts, batchYear: 2000 + Number(parts.batch), departmentName: department.name }
}

const titles = new Set(['prof', 'dr', 'mr', 'mrs', 'ms', 'shri', 'smt'])

// greetingName is the name to greet someone by: their first name, skipping a
// title, so "Prof. Kiran Hegde" is "Kiran".
export function greetingName(fullName: string): string {
  const words = fullName.trim().split(/\s+/)
  const first = words.find((word) => !titles.has(word.replace(/\.$/, '').toLowerCase()))
  return first ?? words[0] ?? ''
}

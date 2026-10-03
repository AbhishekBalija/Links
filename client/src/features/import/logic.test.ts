import { describe, expect, it } from 'vitest'
import { checkTiles, failedRowsCSV, fileProblem, fileTrouble, rowStatus, whyNotAdded } from './logic'
import type { ImportGroup, ImportResult, ImportRow } from './types'

const group = (over: Partial<ImportGroup> = {}): ImportGroup => ({
  department_code: 'CS',
  department_name: 'Computer Science and Engineering',
  batch_year: 2025,
  rows: 57,
  ready: 57,
  created: 0,
  failed: 0,
  outside: false,
  ...over,
})
const row = (over: Partial<ImportRow> = {}): ImportRow => ({
  row: 2,
  email: 'kavya@gmail.com',
  full_name: 'Kavya Rao',
  usn: '4MN25CS001',
  department_code: 'CS',
  batch_year: 2025,
  status: 'ready',
  ...over,
})
const result = (over: Partial<ImportResult> = {}): ImportResult => ({
  dry_run: true,
  department: null,
  ready: 0,
  created: 0,
  failed: 0,
  groups: [],
  rows: [],
  ...over,
})

describe('whyNotAdded', () => {
  it('reads the server reason as a sentence', () => {
    expect(whyNotAdded(row({ status: 'failed', error: 'the email is already registered' }))).toBe('This email already has an account.')
    expect(whyNotAdded(row({ status: 'failed', error: 'the USN appears earlier in this file' }))).toBe('The USN appears earlier in this file.')
  })

  it("names the other department for a row outside the import's", () => {
    const outside = row({ status: 'failed', outside: true, department_code: 'EC', error: 'the USN is in EC, not CS' })
    expect(whyNotAdded(outside, 'Computer Science and Engineering', 'Electronics and Communication Engineering')).toBe(
      "Electronics and Communication Engineering, not Computer Science and Engineering. Send it to that department's HOD.",
    )
  })

  it('explains a USN that cannot be read', () => {
    expect(whyNotAdded(row({ status: 'failed', usn: '4MN25CS02', department_code: undefined, error: 'invalid USN: invalid USN format' }))).toBe(
      "The USN can't be read. USNs look like 4MN25CS001.",
    )
  })
})

describe('rowStatus', () => {
  it('says what happens to a row', () => {
    expect(rowStatus(row())).toEqual({ ok: true, text: 'Will be added' })
    expect(rowStatus(row({ status: 'created' }))).toEqual({ ok: true, text: 'Added' })
    expect(rowStatus(row({ status: 'failed', error: 'the email is already registered' }))).toEqual({
      ok: false,
      text: "Won't be added: this email already has an account",
    })
  })

  it("keeps a department's name as it is", () => {
    const outside = row({ status: 'failed', outside: true, department_code: 'ME' })
    expect(rowStatus(outside, 'Civil Engineering', 'Mechanical Engineering').text).toBe(
      "Won't be added: Mechanical Engineering, not Civil Engineering. Send it to that department's HOD",
    )
  })
})

describe('checkTiles', () => {
  it('counts the rows each department and batch gets, then what fails', () => {
    const checked = result({
      ready: 59,
      failed: 1,
      groups: [group(), group({ department_code: 'EC', department_name: 'Electronics and Communication Engineering', rows: 2, ready: 2 })],
      rows: [row({ status: 'failed', department_code: undefined, error: 'invalid USN: invalid USN format' })],
    })
    expect(checkTiles(checked)).toEqual([
      { count: 57, label: 'Computer Science and Engineering, 2025', bad: false },
      { count: 2, label: 'Electronics and Communication Engineering, 2025', bad: false },
      { count: 1, label: "USN can't be read", bad: true },
    ])
  })

  it('puts rows outside the department and rows already in LINKS in their own tiles', () => {
    const checked = result({
      ready: 0,
      failed: 3,
      groups: [group({ ready: 0, failed: 1, rows: 1 }), group({ department_code: 'EC', department_name: 'Electronics', rows: 2, ready: 0, failed: 2, outside: true })],
      rows: [
        row({ status: 'failed', error: 'the email is already registered' }),
        row({ status: 'failed', outside: true, department_code: 'EC', error: 'the USN is in EC, not CS' }),
        row({ status: 'failed', outside: true, department_code: 'EC', error: 'the USN is in EC, not CS' }),
      ],
    })
    expect(checkTiles(checked)).toEqual([
      { count: 0, label: 'will be added', bad: false },
      { count: 2, label: 'in other departments', bad: true },
      { count: 1, label: 'already have an account', bad: true },
    ])
  })
})

describe('failedRowsCSV', () => {
  it('writes the rows that were not added back as a class list, quoting where needed', () => {
    const rows = [
      row({ status: 'failed', full_name: 'Kumar, Ravi', email: 'ravi@gmail.com', usn: '4MN25CS02' }),
      row({ status: 'created' }),
      row({ status: 'failed', full_name: 'Sneha "Snu" D', email: 'sneha@gmail.com', usn: '4MN25EC019' }),
    ]
    expect(failedRowsCSV(rows)).toBe('email,full_name,usn\nravi@gmail.com,"Kumar, Ravi",4MN25CS02\nsneha@gmail.com,"Sneha ""Snu"" D",4MN25EC019\n')
  })
})

describe('fileProblem', () => {
  it('accepts a CSV up to 1 MB', () => {
    expect(fileProblem({ name: 'batch-2025.csv', size: 4000 })).toBe('')
  })

  it('refuses another kind of file or one that is too large', () => {
    expect(fileProblem({ name: 'batch-2025.xlsx', size: 4000 })).toBe('This is not a CSV file. Save it from your spreadsheet as CSV and try again.')
    expect(fileProblem({ name: 'big.csv', size: 2_000_000 })).toBe('This file is over 1 MB. A class list of 200 students is far smaller; check it is the right file.')
  })
})

describe('fileTrouble', () => {
  it("explains a header the server can't read", () => {
    expect(fileTrouble('invalid header', { file: 'the header must be email,full_name,usn' })).toBe('header')
  })

  it("passes on the server's reason otherwise", () => {
    expect(fileTrouble('too many rows', { file: 'at most 200 students per file' })).toBe('At most 200 students per file.')
  })
})

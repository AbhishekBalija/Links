import type { ImportResult, ImportRow } from './types'

export const MAX_FILE_BYTES = 1_000_000

// The example class list the screen offers to download.
export const exampleCSV = 'email,full_name,usn\nasha.rao@gmail.com,Asha Rao,4MN23CS042\nrohan.s@college.edu,Rohan Shetty,4MN23CS017\n'

function sentence(text: string): string {
  const trimmed = text.trim()
  if (!trimmed) return ''
  const capital = trimmed.charAt(0).toUpperCase() + trimmed.slice(1)
  return /[.!?]$/.test(capital) ? capital : `${capital}.`
}

const knownReasons: Record<string, string> = {
  'the email is already registered': 'This email already has an account.',
  'the USN is already registered': 'This USN already has an account.',
}

// whyNotAdded says why a row won't be (or wasn't) added, as a sentence. A
// row outside the import's Department names both Departments.
export function whyNotAdded(row: ImportRow, importDepartment?: string, rowDepartment?: string): string {
  if (row.outside) {
    const theirs = rowDepartment ?? row.department_code ?? 'Another department'
    return `${theirs}, not ${importDepartment ?? 'your department'}. Send it to that department's HOD.`
  }
  const error = row.error ?? ''
  if (error.startsWith('invalid USN')) return "The USN can't be read. USNs look like 4MN25CS001."
  return knownReasons[error] ?? sentence(error)
}

export type RowStatus = { ok: boolean; text: string }

// rowStatus is the last column of the check table.
export function rowStatus(row: ImportRow, importDepartment?: string, rowDepartment?: string): RowStatus {
  if (row.status === 'ready') return { ok: true, text: 'Will be added' }
  if (row.status === 'created') return { ok: true, text: 'Added' }
  const why = whyNotAdded(row, importDepartment, rowDepartment).replace(/\.$/, '')
  // A sentence reads on after the colon in lower case; a department's name
  // keeps its capitals.
  const text = row.outside ? why : why.charAt(0).toLowerCase() + why.slice(1)
  return { ok: false, text: `Won't be added: ${text}` }
}

export type Tile = { count: number; label: string; bad: boolean }

// checkTiles sums up a check: how many students each Department and Batch
// gets, then the rows that won't be added, by reason.
export function checkTiles(result: ImportResult): Tile[] {
  const counted = (n: number) => n > 0
  const good = result.groups
    .filter((g) => !g.outside && counted(result.dry_run ? g.ready : g.created))
    .map((g) => ({ count: result.dry_run ? g.ready : g.created, label: `${g.department_name || g.department_code}, ${g.batch_year}`, bad: false }))
  const tiles: Tile[] = good.length > 0 ? good : [{ count: 0, label: result.dry_run ? 'will be added' : 'added', bad: false }]

  const failed = result.rows.filter((r) => r.status === 'failed')
  const outside = failed.filter((r) => r.outside).length
  const unreadable = failed.filter((r) => !r.outside && (r.error ?? '').startsWith('invalid USN')).length
  const registered = failed.filter((r) => !r.outside && (r.error ?? '').endsWith('already registered')).length
  const other = failed.length - outside - unreadable - registered
  if (outside) tiles.push({ count: outside, label: 'in other departments', bad: true })
  if (unreadable) tiles.push({ count: unreadable, label: "USN can't be read", bad: true })
  if (registered) tiles.push({ count: registered, label: 'already have an account', bad: true })
  if (other) tiles.push({ count: other, label: 'with other problems', bad: true })
  return tiles
}

function csvCell(value: string): string {
  return /[",\n\r]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
}

// failedRowsCSV writes the rows that were not added as a class list again,
// so they can be fixed in a spreadsheet and uploaded on their own.
export function failedRowsCSV(rows: ImportRow[]): string {
  const lines = rows.filter((r) => r.status === 'failed').map((r) => [r.email, r.full_name, r.usn].map(csvCell).join(','))
  return ['email,full_name,usn', ...lines].join('\n') + '\n'
}

// fileProblem checks a chosen file before uploading it.
export function fileProblem(file: { name: string; size: number }): string {
  if (!file.name.toLowerCase().endsWith('.csv')) return 'This is not a CSV file. Save it from your spreadsheet as CSV and try again.'
  if (file.size > MAX_FILE_BYTES) return 'This file is over 1 MB. A class list of 200 students is far smaller; check it is the right file.'
  return ''
}

// fileTrouble reads a 400 for the whole file: "header" for a header the
// server can't read (the screen explains it in full), else the reason.
export function fileTrouble(message: string, details?: Record<string, unknown>): string {
  if (message === 'invalid header' || message === 'the file is empty') return 'header'
  const reason = typeof details?.file === 'string' ? details.file : message
  return sentence(reason)
}

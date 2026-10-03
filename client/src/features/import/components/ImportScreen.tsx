import { CircleAlert, Download, FileText, Upload } from 'lucide-react'
import { useId, useRef, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { Skeleton } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { buttonStyles } from '../../announcements/buttons'
import { saveText, useImport } from '../api'
import { checkTiles, exampleCSV, failedRowsCSV, fileProblem, fileTrouble, rowStatus, whyNotAdded, type Tile } from '../logic'
import type { ImportResult, ImportRow } from '../types'

type Props = {
  // An HOD's import is held to their own Department; an admin's takes any.
  department?: { code: string; name: string }
}

type Step = 'upload' | 'checked' | 'done'

// ImportScreen adds a class list in two steps: check every row (nothing is
// saved), then import the rows that will be added. Each USN decides the
// student's Department and Batch.
export function ImportScreen({ department }: Props) {
  const [step, setStep] = useState<Step>('upload')
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [fileError, setFileError] = useState('')
  const [checkFailed, setCheckFailed] = useState(false)
  const send = useImport()

  async function check(chosen: File) {
    setFile(chosen)
    setResult(null)
    setCheckFailed(false)
    const problem = fileProblem(chosen)
    setFileError(problem)
    if (problem) {
      setStep('upload')
      return
    }
    setStep('checked')
    try {
      setResult(await send.mutateAsync({ file: chosen, dryRun: true }))
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400) {
        setFileError(fileTrouble(err.message, err.details))
        setStep('upload')
      } else {
        setCheckFailed(true)
      }
    }
  }

  async function save() {
    if (!file) return
    try {
      setResult(await send.mutateAsync({ file, dryRun: false }))
      setStep('done')
    } catch {
      setCheckFailed(true)
    }
  }

  function startOver() {
    setStep('upload')
    setFile(null)
    setResult(null)
    setFileError('')
    setCheckFailed(false)
  }

  if (step === 'done' && result && file) {
    return <Results result={result} fileName={file.name} department={department} onAnother={startOver} />
  }
  if (step === 'checked' && file) {
    return (
      <Check
        file={file}
        result={result}
        department={department}
        checking={send.isPending && !result}
        importing={send.isPending && Boolean(result)}
        failed={checkFailed}
        onRetry={() => (result ? save() : check(file))}
        onChoose={check}
        onImport={save}
      />
    )
  }
  return <UploadStep department={department} error={fileError} checking={send.isPending} fileName={file?.name} onChoose={check} />
}

function UploadStep({ department, error, checking, fileName, onChoose }: {
  department?: { code: string; name: string }
  error: string
  checking: boolean
  fileName?: string
  onChoose: (file: File) => void
}) {
  const [dragging, setDragging] = useState(false)
  const input = useRef<HTMLInputElement>(null)
  const hintId = useId()

  return (
    <div className="grid items-start gap-3.5 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:gap-6">
      <section className="flex min-w-0 flex-col gap-4 rounded-xl border border-line bg-surface px-[18px] py-5 lg:px-7 lg:py-6">
        <h2 className="font-serif text-[22px] leading-[1.2] font-medium">Upload a class list</h2>
        {error && (
          <Alert>
            {error === 'header' ? (
              <>
                <b className="font-semibold">This file can't be read as a class list.</b> The first line must be exactly{' '}
                <span className="font-mono">email,full_name,usn</span>. Save it from your spreadsheet as CSV and try again.
              </>
            ) : (
              <>
                <b className="font-semibold">{fileName ? `${fileName} can't be imported.` : "This file can't be imported."}</b> {error}
              </>
            )}
          </Alert>
        )}
        <div
          onDragOver={(e) => {
            e.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(e) => {
            e.preventDefault()
            setDragging(false)
            const dropped = e.dataTransfer.files[0]
            if (dropped) onChoose(dropped)
          }}
          className={cn(
            'flex min-h-[200px] flex-col items-center justify-center gap-2.5 rounded-xl border-[1.5px] border-dashed bg-paper px-6 py-5 text-center',
            dragging ? 'border-rust' : 'border-[#CFC4B2]',
          )}
        >
          <Upload aria-hidden="true" className="size-5 text-ink-2" />
          <span className="hidden text-[15px] font-semibold lg:block">Drop a CSV file here</span>
          <span className="hidden text-[13px] text-ink-3 lg:block">or</span>
          <input
            ref={input}
            type="file"
            accept=".csv,text/csv"
            className="sr-only"
            aria-describedby={hintId}
            onChange={(e) => {
              const chosen = e.target.files?.[0]
              e.target.value = ''
              if (chosen) onChoose(chosen)
            }}
          />
          <button type="button" disabled={checking} onClick={() => input.current?.click()} className={buttonStyles.secondary}>
            {checking ? 'Checking…' : 'Choose a file'}
          </button>
        </div>
        <p id={hintId} className="text-[13px] text-ink-3">
          You'll check every row before anything is saved.
        </p>
      </section>
      <FileRules department={department} />
    </div>
  )
}

function FileRules({ department }: { department?: { code: string; name: string } }) {
  return (
    <section className="flex min-w-0 flex-col gap-3.5 rounded-xl border border-line bg-surface px-[18px] py-5 lg:px-6 lg:py-[22px]">
      <h2 className="font-serif text-xl leading-[1.2] font-medium">What the file needs</h2>
      <ul className="flex flex-col gap-1.5">
        <Rule>Up to 200 students per file. Three columns, in this order.</Rule>
        {department ? (
          <Rule>
            Only {department.name} USNs (with {department.code} in the middle, like 4MN23<b className="font-semibold">{department.code}</b>042). Rows for
            other departments won't be added.
          </Rule>
        ) : (
          <Rule>Any department. The USN decides each student's department and batch.</Rule>
        )}
        <Rule>Nothing is emailed. Students sign in with Google or an email code, using the email in the file.</Rule>
      </ul>
      <pre className="overflow-x-auto rounded-lg bg-paper px-4 py-3.5 font-mono text-[13px] leading-[1.7] text-ink-2">
        <b className="font-medium text-ink">email,full_name,usn</b>
        {'\n'}asha.rao@gmail.com,Asha Rao,4MN23CS042{'\n'}rohan.s@college.edu,Rohan Shetty,4MN23CS017
      </pre>
      <button type="button" onClick={() => saveText('class-list-example.csv', exampleCSV)} className="inline-flex min-h-11 items-center gap-1.5 self-start text-sm font-semibold text-rust">
        <Download aria-hidden="true" className="size-4" />
        Download an example file
      </button>
    </section>
  )
}

function Rule({ children }: { children: ReactNode }) {
  return (
    <li className="flex gap-2.5 text-sm leading-normal text-ink-2">
      <span aria-hidden="true" className="text-rust">
        ·
      </span>
      <span>{children}</span>
    </li>
  )
}

function FileChip({ file, rows, onChoose, disabled }: { file: File; rows?: number; onChoose?: (file: File) => void; disabled?: boolean }) {
  const input = useRef<HTMLInputElement>(null)
  return (
    <div className="flex items-center gap-3.5 rounded-[10px] bg-paper px-3.5 py-3">
      <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-surface text-ink-2">
        <FileText aria-hidden="true" className="size-[18px]" />
      </span>
      <span className="flex min-w-0 grow flex-col gap-0.5">
        <span className="text-[15px] font-semibold [overflow-wrap:anywhere]">{file.name}</span>
        <span className="font-mono text-xs text-ink-3">
          {rows !== undefined && `${rows} ${rows === 1 ? 'row' : 'rows'} · `}
          {Math.max(1, Math.round(file.size / 1000))} KB
        </span>
      </span>
      {onChoose && (
        <>
          <input
            ref={input}
            type="file"
            accept=".csv,text/csv"
            className="sr-only"
            tabIndex={-1}
            onChange={(e) => {
              const chosen = e.target.files?.[0]
              e.target.value = ''
              if (chosen) onChoose(chosen)
            }}
          />
          <button type="button" disabled={disabled} onClick={() => input.current?.click()} className="min-h-11 px-2 text-sm font-semibold whitespace-nowrap text-rust disabled:opacity-50">
            Choose another
          </button>
        </>
      )}
    </div>
  )
}

function Check({ file, result, department, checking, importing, failed, onRetry, onChoose, onImport }: {
  file: File
  result: ImportResult | null
  department?: { code: string; name: string }
  checking: boolean
  importing: boolean
  failed: boolean
  onRetry: () => void
  onChoose: (file: File) => void
  onImport: () => void
}) {
  const isDesktop = useIsDesktop()
  const [onlyFailed, setOnlyFailed] = useState(false)
  const [showRows, setShowRows] = useState(false)
  const failedRows = result?.rows.filter((r) => r.status === 'failed') ?? []
  const ready = result?.ready ?? 0
  const names = departmentNames(result)
  const importName = department?.name ?? result?.department?.name

  let body: ReactNode
  if (!result && failed) {
    body = <Alert>{<><b className="font-semibold">Couldn't check {file.name}.</b> Nothing was saved. Check your connection and try again.</>}</Alert>
  } else if (!result || checking) {
    body = (
      <div role="status" className="flex flex-col gap-3">
        <p className="text-sm text-ink-2">Checking every row…</p>
        <Skeleton className="h-3.5 w-full" />
        <Skeleton className="h-3.5 w-[92%]" />
        <Skeleton className="h-3.5 w-[70%]" />
      </div>
    )
  } else {
    const rows = onlyFailed ? failedRows : result.rows
    body = (
      <>
        <p className="text-sm text-ink-2">
          {department
            ? `You're the ${department.name} HOD, so rows for other departments won't be added. Send those to their HOD.`
            : 'One file can hold any departments. Each student goes to the department and batch in their USN.'}
        </p>
        <Tiles tiles={checkTiles(result)} />
        {failed && <Alert>{<><b className="font-semibold">The import didn't go through.</b> Nothing was saved. Check your connection and try again.</>}</Alert>}
        {isDesktop ? (
          <>
            {failedRows.length > 0 && (
              <RowTabs all={result.rows.length} failed={failedRows.length} onlyFailed={onlyFailed} onChange={setOnlyFailed} />
            )}
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="text-left text-xs font-semibold text-ink-3">
                  <th scope="col" className="px-3 pb-1.5 font-semibold">Row</th>
                  <th scope="col" className="px-3 pb-1.5 font-semibold">Name</th>
                  <th scope="col" className="px-3 pb-1.5 font-semibold">USN</th>
                  <th scope="col" className="px-3 pb-1.5 font-semibold">Department, batch</th>
                  <th scope="col" className="px-3 pb-1.5 font-semibold">
                    <span className="sr-only">Result</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const status = rowStatus(row, importName, names[row.department_code ?? ''])
                  return (
                    <tr key={row.row}>
                      <td className="px-3 py-[9px] font-mono text-[13px] text-ink-2">{row.row}</td>
                      <td className="px-3 py-[9px] font-semibold">{row.full_name}</td>
                      <td className="px-3 py-[9px] font-mono text-[13px]">{row.usn}</td>
                      <td className="px-3 py-[9px]">
                        {row.department_code ? `${names[row.department_code] ?? row.department_code}, ${row.batch_year}` : <span className="text-ink-3">Unknown</span>}
                      </td>
                      <td className={cn('px-3 py-[9px] text-right font-medium', status.ok ? 'text-success-ink' : 'text-danger-ink')}>{status.text}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </>
        ) : (
          failedRows.length > 0 && (
            <div className="flex flex-col gap-2">
              <button type="button" aria-expanded={showRows} onClick={() => setShowRows((s) => !s)} className="min-h-11 self-start text-sm font-semibold text-rust">
                {showRows ? 'Hide' : `See the ${failedRows.length} ${failedRows.length === 1 ? 'row' : 'rows'} and why`}
              </button>
              {showRows && <FailedList rows={failedRows} importName={importName} names={names} />}
            </div>
          )
        )}
      </>
    )
  }

  const nothing = Boolean(result) && ready === 0
  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-3.5 rounded-xl border border-line bg-surface px-[18px] py-5 lg:px-7 lg:py-[22px]">
        <FileChip file={file} rows={result?.rows.length} onChoose={checking || importing ? undefined : onChoose} />
        <h2 className="font-serif text-[22px] leading-[1.2] font-medium">Check before importing</h2>
        {body}
      </section>
      <div role="status" className="flex flex-col items-stretch gap-3 lg:flex-row lg:items-center lg:justify-end lg:gap-4">
        <span className="text-sm text-ink-2">
          {importing ? 'Adding them now. Please keep this page open.' : nothing ? 'None of these rows can be added.' : 'Nothing is saved until you import.'}
        </span>
        {!result && failed ? (
          <button type="button" onClick={onRetry} className={buttonStyles.primary}>
            Try again
          </button>
        ) : (
          <button
            type="button"
            disabled={!result || checking || importing || nothing}
            onClick={failed ? onRetry : onImport}
            className={cn(buttonStyles.primary, 'disabled:opacity-40')}
          >
            {importing ? `Importing ${ready} ${ready === 1 ? 'student' : 'students'}…` : failed ? 'Try again' : nothing || !result ? 'Import students' : `Import ${ready} ${ready === 1 ? 'student' : 'students'}`}
          </button>
        )}
      </div>
    </div>
  )
}

function Results({ result, fileName, department, onAnother }: {
  result: ImportResult
  fileName: string
  department?: { code: string; name: string }
  onAnother: () => void
}) {
  const failedRows = result.rows.filter((r) => r.status === 'failed')
  const names = departmentNames(result)
  const importName = department?.name ?? result.department?.name
  const added = result.groups.filter((g) => g.created > 0)
  const [open, setOpen] = useState<string | null>(null)
  const outside = failedRows.filter((r) => r.outside).length
  const fixable = failedRows.length - outside

  let next = `The ${result.created} can sign in now.`
  if (fixable > 0) next += ` Fix the ${fixable === 1 ? 'row' : `${fixable} rows`} below and upload ${fixable === 1 ? 'it' : 'just those'} again.`
  if (outside > 0) next += ` Send the ${outside === 1 ? 'row' : `${outside} rows`} for other departments to their HOD.`

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:gap-3">
        <div className="flex gap-3">
          <Count tile={{ count: result.created, label: 'added', bad: false }} />
          {failedRows.length > 0 && <Count tile={{ count: failedRows.length, label: 'not added', bad: true }} />}
        </div>
        <div role="status" className="flex flex-col gap-0.5 lg:ml-3">
          <span className="text-[15px] font-semibold [overflow-wrap:anywhere]">{fileName}</span>
          <span className="text-sm text-ink-2">{next}</span>
        </div>
      </div>

      {failedRows.length > 0 && (
        <section className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-[18px] py-5 lg:px-7 lg:py-[22px]">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 className="font-serif text-[22px] leading-[1.2] font-medium">Not added</h2>
            <button
              type="button"
              onClick={() => saveText(fileName.replace(/\.csv$/i, '') + '-not-added.csv', failedRowsCSV(result.rows))}
              className={buttonStyles.secondary}
            >
              <Download aria-hidden="true" className="size-4" />
              Download the {failedRows.length === 1 ? 'row' : `${failedRows.length} rows`}
            </button>
          </div>
          <FailedList rows={failedRows} importName={importName} names={names} withEmail />
        </section>
      )}

      {added.length > 0 && (
        <section className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-[18px] py-5 lg:px-7 lg:py-[22px]">
          <h2 className="font-serif text-[22px] leading-[1.2] font-medium">Added</h2>
          <ul className="flex flex-col gap-1.5">
            {added.map((g) => {
              const key = `${g.department_code}-${g.batch_year}`
              const people = result.rows.filter((r) => r.status === 'created' && r.department_code === g.department_code && r.batch_year === g.batch_year)
              return (
                <li key={key} className="flex flex-col rounded-lg bg-paper px-3.5 py-2">
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-[15px] font-semibold">
                      {g.department_name || g.department_code}, {g.batch_year}
                    </span>
                    <span className="flex items-center gap-3">
                      <span className="font-mono text-[13px] text-success-ink">
                        {g.created} {g.created === 1 ? 'student' : 'students'}
                      </span>
                      <button type="button" aria-expanded={open === key} onClick={() => setOpen(open === key ? null : key)} className="min-h-11 text-sm font-semibold text-rust">
                        {open === key ? 'Hide' : 'Show'}
                      </button>
                    </span>
                  </div>
                  {open === key && (
                    <ul className="flex flex-col gap-1 pb-2 text-sm text-ink-2">
                      {people.map((p) => (
                        <li key={p.row}>
                          {p.full_name} <span className="font-mono text-xs text-ink-3">{p.usn}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              )
            })}
          </ul>
        </section>
      )}

      <div className="flex justify-end">
        <button type="button" onClick={onAnother} className={buttonStyles.primary}>
          <Upload aria-hidden="true" className="size-4" />
          Import another file
        </button>
      </div>
    </div>
  )
}

function FailedList({ rows, importName, names, withEmail = false }: {
  rows: ImportRow[]
  importName?: string
  names: Record<string, string>
  withEmail?: boolean
}) {
  return (
    <>
      <table className="hidden w-full border-collapse text-sm lg:table">
        <thead>
          <tr className="text-left text-xs text-ink-3">
            <th scope="col" className="px-3 pb-1.5 font-semibold">Row</th>
            <th scope="col" className="px-3 pb-1.5 font-semibold">{withEmail ? 'Email' : 'Name'}</th>
            <th scope="col" className="px-3 pb-1.5 font-semibold">USN</th>
            <th scope="col" className="px-3 pb-1.5 font-semibold">Why it wasn't added</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.row}>
              <td className="px-3 py-2.5 font-mono text-[13px] text-ink-2">{row.row}</td>
              <td className="px-3 py-2.5">{withEmail ? row.email : row.full_name}</td>
              <td className="px-3 py-2.5 font-mono text-[13px]">{row.usn}</td>
              <td className="px-3 py-2.5">{whyNotAdded(row, importName, names[row.department_code ?? ''])}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <ul className="flex flex-col gap-2 lg:hidden">
        {rows.map((row) => (
          <li key={row.row} className="flex flex-col gap-0.5 rounded-lg bg-paper px-3.5 py-2.5 text-sm">
            <span className="font-semibold">
              Row {row.row} · {row.full_name}
            </span>
            <span className="font-mono text-xs text-ink-3">{row.usn}</span>
            <span className="text-danger-ink">{whyNotAdded(row, importName, names[row.department_code ?? ''])}</span>
          </li>
        ))}
      </ul>
    </>
  )
}

function RowTabs({ all, failed, onlyFailed, onChange }: { all: number; failed: number; onlyFailed: boolean; onChange: (only: boolean) => void }) {
  const tab = (on: boolean) =>
    cn('flex min-h-10 items-center gap-1.5 rounded-[7px] px-4 text-sm whitespace-nowrap', on ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2')
  return (
    <div role="group" aria-label="Which rows" className="flex gap-1 self-start rounded-[10px] bg-well p-1">
      <button type="button" aria-pressed={!onlyFailed} onClick={() => onChange(false)} className={tab(!onlyFailed)}>
        All rows <span className={cn('font-mono text-xs', onlyFailed ? 'text-ink-3' : 'text-rust')}>{all}</span>
      </button>
      <button type="button" aria-pressed={onlyFailed} onClick={() => onChange(true)} className={tab(onlyFailed)}>
        Won't be added <span className={cn('font-mono text-xs', onlyFailed ? 'text-rust' : 'text-ink-3')}>{failed}</span>
      </button>
    </div>
  )
}

function Tiles({ tiles }: { tiles: Tile[] }) {
  return (
    <div className="grid grid-cols-2 gap-2.5 lg:flex lg:flex-wrap">
      {tiles.map((tile) => (
        <Count key={tile.label} tile={tile} />
      ))}
    </div>
  )
}

function Count({ tile }: { tile: Tile }) {
  return (
    <div className={cn('flex min-w-[120px] flex-col gap-0.5 rounded-[10px] px-4 py-3', tile.bad ? 'bg-danger-soft text-danger-ink' : 'bg-success-soft text-success-ink')}>
      <span className="font-mono text-[22px] font-medium">{tile.count}</span>
      <span className="text-[13px] font-semibold">{tile.label}</span>
    </div>
  )
}

function Alert({ children }: { children: ReactNode }) {
  return (
    <div role="alert" className="flex items-start gap-2.5 rounded-[10px] bg-danger-soft px-4 py-3.5 text-sm leading-[1.45] text-danger-ink">
      <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <span>{children}</span>
    </div>
  )
}

// departmentNames maps each Department code in the result to its name.
function departmentNames(result: ImportResult | null): Record<string, string> {
  const names: Record<string, string> = {}
  for (const g of result?.groups ?? []) if (g.department_name) names[g.department_code] = g.department_name
  return names
}

import { ChevronDown, ChevronLeft, Download, Search, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useDepartments } from '../../announcements/api'
import { batchOptions } from '../../announcements/audience'
import { buttonStyles } from '../../announcements/buttons'
import { dayAndDate, dayMonth } from '../../events/standing'
import type { ApplicantCounts, ApplicationStatus, Opportunity } from '../../jobs/types'
import { useApplicants, useApplicantStatus, useExportApplicants, useManagedOne } from '../api'
import { applicantTabs, staffStatuses, staffStatusLabel, type Applicant, type ApplicantFilters } from '../applicants'

const allStatuses: ApplicationStatus[] = ['applied', 'shortlisted', 'selected', 'rejected', 'withdrawn']
const isStatus = (value: string | null): value is ApplicationStatus => allStatuses.includes(value as ApplicationStatus)

// The four most recent batches are the students still on campus.
const batches = batchOptions().slice(0, 4).reverse()

const tones: Record<ApplicationStatus, string> = {
  applied: 'bg-well text-ink-2',
  shortlisted: 'bg-warning-soft text-warning-ink',
  selected: 'bg-success-soft text-success-ink',
  rejected: 'bg-danger-soft text-danger-ink',
  withdrawn: 'text-ink-2 shadow-[inset_0_0_0_1px_#d6ccbb]',
}

type Saved = { applicant: Applicant; from: ApplicationStatus; to: ApplicationStatus }

// Applicants is an Opportunity's applicant list for placement staff: a table
// on desktop and a readable list on phones, with the status, department,
// batch and search filters kept in the address bar.
export default function Applicants() {
  const { id = '' } = useParams()
  const item = useManagedOne(id)
  if (item.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <LoadingStatus label="Loading applicants" />
        <Skeleton className="h-10 w-64" />
        <TableSkeleton />
      </div>
    )
  }
  if (item.isError) {
    return item.error instanceof ApiRequestError && item.error.status === 404 ? (
      <EmptyState title="This opportunity doesn't exist">The link may be wrong.</EmptyState>
    ) : (
      <ErrorState message="Applicants could not be loaded. Nothing was changed." onRetry={() => item.refetch()} />
    )
  }
  return <ApplicantList item={item.data} />
}

function ApplicantList({ item }: { item: Opportunity }) {
  const [params, setParams] = useSearchParams()
  const status = isStatus(params.get('status')) ? (params.get('status') as ApplicationStatus) : null
  const filters: ApplicantFilters = {
    status,
    department: params.get('department') ?? '',
    batch: params.get('batch') ?? '',
    q: params.get('q') ?? '',
  }
  const [search, setSearch] = useState(filters.q)
  const list = useApplicants(item.id, filters)
  const departments = useDepartments()
  const change = useApplicantStatus(item.id)
  const [saved, setSaved] = useState<Saved | null>(null)
  const [problem, setProblem] = useState<{ id: string; text: string } | null>(null)
  const rows = list.data?.pages.flatMap((page) => page.data) ?? []
  const filtered = Boolean(filters.q || filters.department || filters.batch)

  function setParam(key: string, value: string) {
    setParams(
      (current) => {
        const next = new URLSearchParams(current)
        if (value) next.set(key, value)
        else next.delete(key)
        return next
      },
      { replace: true },
    )
  }

  // Searching waits for a pause in typing, so each keystroke isn't a request.
  useEffect(() => {
    const q = search.trim()
    if (q === filters.q) return
    const timer = setTimeout(() => {
      setParams(
        (current) => {
          const next = new URLSearchParams(current)
          if (q) next.set('q', q)
          else next.delete('q')
          return next
        },
        { replace: true },
      )
    }, 300)
    return () => clearTimeout(timer)
  }, [search, filters.q, setParams])

  // The saved message steps aside after a few seconds.
  useEffect(() => {
    if (!saved) return
    const timer = setTimeout(() => setSaved(null), 6000)
    return () => clearTimeout(timer)
  }, [saved])

  async function move(applicant: Applicant, to: ApplicationStatus, undo = false) {
    setProblem(null)
    try {
      await change.mutateAsync({ applicant, to })
      setSaved(undo ? null : { applicant: { ...applicant, status: to }, from: applicant.status, to })
    } catch (err) {
      setSaved(null)
      setProblem({
        id: applicant.id,
        text:
          err instanceof ApiRequestError && err.status === 409
            ? `Not changed. Someone else changed ${applicant.student.full_name}'s status a moment ago, or the student withdrew. The row now shows the latest status, so check it before changing it again.`
            : `Not changed. ${applicant.student.full_name}'s status could not be saved; try again.`,
      })
    }
  }

  const closed = item.status !== 'published' || new Date(item.apply_by) <= new Date()
  const counts = item.applicant_counts
  const nothingYet = counts !== undefined && counts.total === 0 && counts.withdrawn === 0

  return (
    <div className="flex flex-col gap-4 pb-24 lg:gap-5 lg:pb-0">
      <Link to={`/placement/${item.id}`} className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        {item.title}
      </Link>
      <header className="-mt-2 flex items-end justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Applicants</h1>
          <p className="text-[13px] text-ink-2 lg:text-[15px]">
            {item.company} · {item.application_mode === 'internal' ? 'students apply in LINKS' : "students mark that they applied on the company's site"} ·{' '}
            {closed ? `closed ${dayMonth(item.closed_at ?? item.apply_by)}` : `closes ${dayAndDate(item.apply_by)}`}
          </p>
        </div>
        {!nothingYet && <ExportMenu item={item} status={status} counts={counts} />}
      </header>

      {nothingYet ? (
        <EmptyState title="No one has applied yet">
          {closed
            ? 'It closed without any applications.'
            : `Eligible students see it in Jobs. Applications show here as they come in, oldest first. It closes ${dayAndDate(item.apply_by)}.`}
        </EmptyState>
      ) : (
        <>
          <StatusTabs counts={counts} status={status} params={params} />
          <div className="flex flex-col gap-2.5 lg:flex-row lg:items-center">
            <label className="flex min-h-11 items-center gap-2 rounded-[10px] border border-[#d6ccbb] bg-surface px-3 text-sm lg:w-[360px]">
              <Search aria-hidden="true" className="size-4 shrink-0 text-ink-3" />
              <input
                type="search"
                aria-label="Search applicants"
                placeholder="Search name, USN or email"
                value={search}
                maxLength={100}
                onChange={(e) => setSearch(e.target.value)}
                className="min-w-0 flex-1 bg-transparent outline-none placeholder:text-ink-3 [&::-webkit-search-cancel-button]:hidden"
              />
              {search && (
                <button type="button" aria-label="Clear search" onClick={() => setSearch('')} className="flex size-8 items-center justify-center rounded-md text-ink-2 hover:bg-well">
                  <X aria-hidden="true" className="size-4" />
                </button>
              )}
            </label>
            <div className="flex gap-2.5">
              <Filter label="Department" value={filters.department} onChange={(v) => setParam('department', v)} options={(departments.data ?? []).map((d) => ({ value: d.code, label: d.code }))} all="All departments" />
              <Filter label="Batch" value={filters.batch} onChange={(v) => setParam('batch', v)} options={batches.map((y) => ({ value: String(y), label: String(y) }))} all="All batches" />
            </div>
          </div>

          {list.isPending ? (
            <>
              <LoadingStatus label="Loading applicants" />
              <TableSkeleton />
            </>
          ) : list.isError && rows.length === 0 ? (
            <ErrorState message="Applicants could not be loaded. Nothing was changed." onRetry={() => list.refetch()} />
          ) : rows.length === 0 ? (
            <EmptyState title={filters.q ? `No applicant matches “${filters.q}”` : 'No applicants here'}>
              <p>{filtered ? 'Search looks at name, USN and email, within the status and filters chosen. Try fewer letters, or clear the filters.' : 'No one is in this status yet.'}</p>
              {filtered && (
                <button
                  type="button"
                  onClick={() => {
                    setSearch('')
                    setParams(status ? { status } : {}, { replace: true })
                  }}
                  className={cn(buttonStyles.secondary, 'mt-3')}
                >
                  Clear search and filters
                </button>
              )}
            </EmptyState>
          ) : (
            <>
              <Table rows={rows} problem={problem} busy={change.isPending} onMove={move} onDismiss={() => setProblem(null)} />
              <PhoneList rows={rows} problem={problem} busy={change.isPending} onMove={move} onDismiss={() => setProblem(null)} />
              <div className="flex items-center justify-between gap-3">
                <span className="text-[13px] text-ink-3">
                  {list.hasNextPage ? `Showing the first ${rows.length}, in the order they applied` : `${rows.length} ${rows.length === 1 ? 'applicant' : 'applicants'}, in the order they applied`}
                </span>
                {list.hasNextPage && (
                  <button type="button" onClick={() => list.fetchNextPage()} disabled={list.isFetchingNextPage} className={buttonStyles.secondary}>
                    {list.isFetchingNextPage ? 'Loading…' : 'Load more'}
                  </button>
                )}
              </div>
            </>
          )}
        </>
      )}

      {saved && (
        <div
          role="status"
          className="fixed inset-x-4 bottom-6 z-50 flex items-center justify-between gap-4 rounded-[10px] border border-[#d6ccbb] bg-surface py-2 pr-2 pl-[18px] text-sm shadow-[0_8px_24px_rgba(27,24,20,0.10)] lg:inset-x-auto lg:left-1/2 lg:-translate-x-1/2"
        >
          <span>
            {saved.applicant.student.full_name} is now <b className="font-semibold">{staffStatusLabel(saved.to)}</b>.
          </span>
          <button type="button" onClick={() => move(saved.applicant, saved.from, true)} className={cn(buttonStyles.secondary, 'min-h-9')}>
            Undo
          </button>
        </div>
      )}
    </div>
  )
}

function StatusTabs({ counts, status, params }: { counts: ApplicantCounts | undefined; status: ApplicationStatus | null; params: URLSearchParams }) {
  return (
    <nav aria-label="Status" className="-mx-4 flex gap-2 overflow-x-auto px-4 lg:mx-0 lg:gap-1 lg:self-start lg:rounded-[10px] lg:bg-well lg:p-1 lg:px-1">
      {applicantTabs(counts).map((tab) => {
        const next = new URLSearchParams(params)
        if (tab.value) next.set('status', tab.value)
        else next.delete('status')
        const active = tab.value === status
        return (
          <Link
            key={tab.label}
            to={`?${next}`}
            replace
            aria-current={active ? 'true' : undefined}
            className={cn(
              'flex min-h-11 shrink-0 items-center gap-1.5 rounded-full border px-3.5 text-sm whitespace-nowrap lg:min-h-9 lg:rounded-[7px] lg:border-0',
              active
                ? 'border-ink bg-ink font-semibold text-paper hover:text-paper lg:bg-surface lg:text-ink lg:shadow-[0_1px_2px_rgba(27,24,20,0.08)] lg:hover:text-ink'
                : 'border-[#d6ccbb] bg-surface text-ink-2 hover:text-ink lg:bg-transparent',
            )}
          >
            {tab.label}
            {tab.count !== undefined && <span className="font-mono text-xs">{tab.count}</span>}
          </Link>
        )
      })}
    </nav>
  )
}

function Filter({ label, value, onChange, options, all }: { label: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[]; all: string }) {
  return (
    <label className="relative flex flex-1 lg:flex-none">
      <span className="sr-only">{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          'min-h-11 w-full cursor-pointer appearance-none rounded-[10px] border py-0 pr-9 pl-3.5 text-sm font-medium',
          value ? 'border-ink bg-ink text-paper' : 'border-[#d6ccbb] bg-surface text-ink',
        )}
      >
        <option value="" className="bg-surface text-ink">
          {all}
        </option>
        {options.map((option) => (
          <option key={option.value} value={option.value} className="bg-surface text-ink">
            {option.label}
          </option>
        ))}
      </select>
      <ChevronDown aria-hidden="true" className={cn('pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2', value ? 'text-paper' : 'text-ink-3')} />
    </label>
  )
}

type RowsProps = {
  rows: Applicant[]
  problem: { id: string; text: string } | null
  busy: boolean
  onMove: (applicant: Applicant, to: ApplicationStatus) => void
  onDismiss: () => void
}

// StatusControl changes an Application's status in place. A withdrawn one is
// the student's decision, so it is shown, not offered.
function StatusControl({ applicant, busy, onMove, phone }: { applicant: Applicant; busy: boolean; onMove: RowsProps['onMove']; phone?: boolean }) {
  const name = applicant.student.full_name
  if (applicant.status === 'withdrawn') {
    return (
      <span title="The student withdrew" className={cn('inline-block rounded px-[7px] py-0.5 text-[11px] leading-4 font-semibold', tones.withdrawn)}>
        Withdrawn
      </span>
    )
  }
  return (
    <label className="relative inline-flex">
      <select
        aria-label={`Status for ${name}`}
        value={applicant.status}
        disabled={busy}
        onChange={(e) => onMove(applicant, e.target.value as ApplicationStatus)}
        className={cn(
          'cursor-pointer appearance-none rounded-md py-0 pr-7 pl-2 text-xs font-semibold outline-offset-2 hover:ring-1 hover:ring-[#d6ccbb] disabled:cursor-wait',
          phone ? 'min-h-11' : 'min-h-9',
          tones[applicant.status],
        )}
      >
        {staffStatuses.map((status) => (
          <option key={status} value={status} className="bg-surface text-ink">
            {staffStatusLabel(status)}
          </option>
        ))}
      </select>
      <ChevronDown aria-hidden="true" className="pointer-events-none absolute top-1/2 right-2 size-3.5 -translate-y-1/2 opacity-70" />
    </label>
  )
}

function Problem({ text, onDismiss }: { text: string; onDismiss: () => void }) {
  return (
    <div role="alert" className="flex items-start gap-3 text-sm text-warning-ink">
      <span className="flex-1">{text}</span>
      <button type="button" onClick={onDismiss} className="min-h-8 shrink-0 rounded-md px-2 font-semibold hover:bg-[#efdcc0]">
        OK
      </button>
    </div>
  )
}

function Table({ rows, problem, busy, onMove, onDismiss }: RowsProps) {
  return (
    <div className="hidden overflow-hidden rounded-xl border border-line bg-surface lg:block">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-paper">
          <tr className="text-left font-mono text-[11px] tracking-[1px] text-ink-3 uppercase">
            <th scope="col" className="px-3.5 py-2.5 font-medium">Student</th>
            <th scope="col" className="w-[150px] px-3.5 py-2.5 font-medium">USN</th>
            <th scope="col" className="w-[80px] px-3.5 py-2.5 font-medium">Dept</th>
            <th scope="col" className="w-[80px] px-3.5 py-2.5 font-medium">Batch</th>
            <th scope="col" className="w-[110px] px-3.5 py-2.5 font-medium">Applied</th>
            <th scope="col" className="w-[170px] px-3.5 py-2.5 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => {
            const flagged = problem?.id === row.id
            return [
              <tr key={row.id} className={cn(i > 0 && 'shadow-[inset_0_1px_0_#efe9de]', flagged && 'bg-warning-soft')}>
                <td className="px-3.5 py-3">
                  <span className="flex flex-col gap-0.5">
                    <Link to={`/people/${row.student.username}`} className="font-semibold text-ink hover:text-rust">
                      {row.student.full_name}
                    </Link>
                    {row.student.email && <span className="text-[13px] text-ink-3">{row.student.email}</span>}
                  </span>
                </td>
                <td className="px-3.5 py-3 font-mono text-[13px]">{row.student.usn ?? '—'}</td>
                <td className="px-3.5 py-3">{row.student.department_code ?? '—'}</td>
                <td className="px-3.5 py-3">{row.student.batch_year ?? '—'}</td>
                <td className="px-3.5 py-3 font-mono text-[13px] text-ink-3">{dayMonth(row.applied_at)}</td>
                <td className="px-3.5 py-3">
                  <StatusControl applicant={row} busy={busy} onMove={onMove} />
                </td>
              </tr>,
              flagged ? (
                <tr key={`${row.id}-problem`} className="bg-warning-soft">
                  <td colSpan={6} className="px-3.5 pb-3">
                    <Problem text={problem.text} onDismiss={onDismiss} />
                  </td>
                </tr>
              ) : null,
            ]
          })}
        </tbody>
      </table>
    </div>
  )
}

function PhoneList({ rows, problem, busy, onMove, onDismiss }: RowsProps) {
  return (
    <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:hidden [&>li+li]:shadow-[0_-1px_0_#efe9de]">
      {rows.map((row) => (
        <li key={row.id} className={cn('flex flex-col gap-2 px-3.5 py-3', problem?.id === row.id && 'bg-warning-soft')}>
          <div className="flex items-center gap-2.5">
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-[15px] font-semibold">{row.student.full_name}</span>
              <span className="font-mono text-xs text-ink-2">
                {[row.student.usn, row.student.department_code, row.student.batch_year].filter(Boolean).join(' · ')}
              </span>
            </span>
            <StatusControl applicant={row} busy={busy} onMove={onMove} phone />
          </div>
          {problem?.id === row.id && <Problem text={problem.text} onDismiss={onDismiss} />}
        </li>
      ))}
    </ul>
  )
}

// ExportMenu downloads the applicants as CSV: the status being looked at
// first, then everyone, then each status on its own.
function ExportMenu({ item, status, counts }: { item: Opportunity; status: ApplicationStatus | null; counts: ApplicantCounts | undefined }) {
  const [open, setOpen] = useState(false)
  const exporter = useExportApplicants(item)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function away(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', away)
    return () => document.removeEventListener('mousedown', away)
  }, [open])

  const everyone = counts ? counts.total + counts.withdrawn : undefined
  const choices: { status: ApplicationStatus | null; label: string; n?: number }[] = [
    { status: null, label: 'Everyone, withdrawn too', n: everyone },
    ...allStatuses.map((s) => ({ status: s, label: `${staffStatusLabel(s)} only`, n: counts?.[s] })),
  ]
  // The status on screen comes first: it is usually what's being sent.
  if (status) choices.sort((a, b) => (a.status === status ? -1 : b.status === status ? 1 : 0))

  return (
    <div ref={ref} className="relative" onKeyDown={(e) => e.key === 'Escape' && setOpen(false)}>
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Export CSV"
        onClick={() => setOpen((o) => !o)}
        className={cn(buttonStyles.secondary, 'px-3 lg:px-[18px]', open && 'ring-2 ring-ink')}
      >
        <Download aria-hidden="true" className="size-4" />
        <span className="hidden lg:inline">Export CSV</span>
      </button>
      {open && (
        <div role="menu" className="absolute top-12 right-0 z-30 flex w-[300px] flex-col rounded-[10px] border border-[#d6ccbb] bg-surface p-1.5 shadow-[0_10px_28px_rgba(27,24,20,0.16)]">
          {choices.map((choice, i) => (
            <button
              key={choice.label}
              type="button"
              role="menuitem"
              disabled={exporter.isPending || choice.n === 0}
              onClick={async () => {
                await exporter.mutateAsync(choice.status).catch(() => undefined)
                setOpen(false)
              }}
              className={cn('flex min-h-[42px] items-center justify-between rounded-md px-2.5 text-left text-sm hover:bg-paper disabled:text-ink-3 disabled:hover:bg-transparent', i === 0 && 'font-semibold')}
            >
              <span>{choice.label}</span>
              {choice.n !== undefined && <span className="font-mono text-xs text-ink-3">{choice.n}</span>}
            </button>
          ))}
          <p className="px-2.5 pt-1 pb-1.5 text-xs text-ink-3">Name, email, USN, department, batch, status and dates. No phone numbers. Each download is logged.</p>
          {exporter.isError && (
            <p role="alert" className="px-2.5 pb-1.5 text-xs font-semibold text-danger">
              The download didn't work. Try again.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function TableSkeleton() {
  return (
    <div aria-hidden="true" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4">
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex items-center gap-4">
          <span className="flex w-1/3 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-36" />
            <Skeleton className="h-3 w-48" />
          </span>
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3.5 w-10" />
          <Skeleton className="h-3.5 w-12" />
          <Skeleton className="h-7 w-28" />
        </div>
      ))}
    </div>
  )
}

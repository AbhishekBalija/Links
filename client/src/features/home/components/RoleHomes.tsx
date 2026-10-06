import { Plus, Upload } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { buttonStyles } from '../../announcements/buttons'
import { waited } from '../../announcements/status'
import type { CollegeDepartment, Dashboard, DepartmentPanel, Lists } from '../api'
import { summaryLine, yourPosts, type HomeKind } from '../roleHome'
import { COLLEGE_TIME_ZONE, collegeDay } from '../../../shared/time/college'

type Props = {
  kind: Exclude<HomeKind, 'everyone'>
  data: Dashboard
  greeting: ReactNode
  notices: ReactNode
  // The principal's side column also shows what's coming up in the college.
  comingUp?: ReactNode
  now: Date
}

// RoleHome is the Home of an HOD, the principal or an admin, built around
// their normal job (the "can vs normally does" principle): an HOD runs a
// Department, the principal oversees the college, an admin keeps people and
// lists in order. Latest notices sit beside it.
export function RoleHome({ kind, data, greeting, notices, comingUp, now }: Props) {
  return (
    <>
      <header className="flex flex-col gap-3 px-1 lg:flex-row lg:items-end lg:justify-between lg:px-0">
        <div className="flex flex-col gap-1.5 lg:gap-2">
          {greeting}
          <p className="text-[15px] text-ink-2 lg:text-base">{summaryLine(kind, data)}</p>
        </div>
        <HeaderActions kind={kind} />
      </header>
      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)] lg:gap-7">
        {kind === 'hod' && (
          <>
            <div className="flex flex-col gap-5 lg:gap-6">
              <WaitingForYou rows={hodWaiting(data, now)} />
              {data.department && <DepartmentSection department={data.department} lists={data.lists} />}
            </div>
            <Side>{notices}</Side>
          </>
        )}
        {kind === 'principal' && (
          <>
            {data.college && <CollegeSection departments={data.college.departments} />}
            <Side>
              {/* On phones Waiting comes first, then the college, then notices. */}
              <div className="contents lg:flex lg:flex-col lg:gap-6">
                <div className="max-lg:order-first max-lg:-mt-0">
                  <WaitingForYou rows={principalWaiting(data, now)} />
                </div>
                <div className="flex flex-col gap-5 max-lg:order-last lg:gap-6">
                  {notices}
                  {comingUp}
                </div>
              </div>
            </Side>
          </>
        )}
        {kind === 'admin' && (
          <>
            <div className="flex flex-col gap-5 lg:gap-6">
              <ToDo data={data} now={now} />
              {data.lists && data.lists.recent_imports.length > 0 && <RecentImports lists={data.lists} now={now} />}
            </div>
            <Side>{notices}</Side>
          </>
        )}
      </div>
    </>
  )
}

// Side is the right-hand column on desktop; on phones its contents simply
// follow the main column.
function Side({ children }: { children: ReactNode }) {
  return <div className="contents lg:col-start-2 lg:row-start-1 lg:flex lg:flex-col lg:gap-6">{children}</div>
}

function HeaderActions({ kind }: { kind: Props['kind'] }) {
  if (kind === 'admin') {
    return (
      <div className="hidden gap-2 lg:flex">
        <Link to="/admin/staff" className={buttonStyles.secondary}>
          <Plus aria-hidden="true" className="size-4" />
          Add staff
        </Link>
        <Link to="/admin/import" className={cn(buttonStyles.primary, 'hover:text-paper')}>
          <Upload aria-hidden="true" className="size-4" />
          Import students
        </Link>
      </div>
    )
  }
  return (
    <Link to="/mine/new" className={cn(buttonStyles.primary, 'hidden self-end hover:text-paper lg:inline-flex')}>
      <Plus aria-hidden="true" className="size-4" />
      New announcement
    </Link>
  )
}

// A row without `to` is information only, until its screen exists.
type WaitingRow = { count: number; title: string; detail: string; to?: string; action?: string; tone?: 'warning' }

function oldest(...dates: (string | null | undefined)[]): string | null {
  const known = dates.filter((d): d is string => Boolean(d))
  if (known.length === 0) return null
  return known.reduce((a, b) => (new Date(a) < new Date(b) ? a : b))
}

function waitingDetail(at: string | null, now: Date, single = false): string {
  if (!at) return ''
  const text = waited(at, now).text
  return single ? text : `oldest ${text}`
}

// requestsRow is the Access requests row: the HOD reviews them in their
// Approval queue, an admin in the Admin workspace.
function requestsRow(data: Dashboard, now: Date, to: string): WaitingRow[] {
  const requests = data.access_requests
  if (!requests || requests.pending_count === 0) return []
  return [
    {
      count: requests.pending_count,
      title: 'Access requests',
      detail: waitingDetail(requests.oldest_requested_at, now, requests.pending_count === 1),
      to,
      tone: 'warning',
    },
  ]
}

// postsRow is the person's own drafts and posts sent back, when there are any.
function postsRow(data: Dashboard): WaitingRow[] {
  const posts = yourPosts(data.my_announcements)
  return posts ? [{ count: posts.count, title: 'Your posts', detail: posts.detail, to: '/mine', action: 'Open' }] : []
}

function hodWaiting(data: Dashboard, now: Date): WaitingRow[] {
  const rows = requestsRow(data, now, '/approvals/access')
  const a = data.approvals
  const posts = (a?.pending_count ?? 0) + (a?.events_pending_count ?? 0)
  if (a && posts > 0) {
    rows.push({
      count: posts,
      title: 'Announcements and events',
      detail: waitingDetail(oldest(a.oldest_submitted_at, a.oldest_event_submitted_at), now, posts === 1),
      to: '/approvals',
    })
  }
  return [...rows, ...postsRow(data)]
}

function principalWaiting(data: Dashboard, now: Date): WaitingRow[] {
  const a = data.approvals
  if (!a) return []
  const rows: WaitingRow[] = []
  if (a.events_pending_count > 0) {
    rows.push({
      count: a.events_pending_count,
      title: a.events_pending_count === 1 ? 'Event for your approval' : 'Events for your approval',
      detail: waitingDetail(a.oldest_event_submitted_at, now, a.events_pending_count === 1),
      to: '/approvals',
    })
  }
  if (a.pending_count > 0) {
    rows.push({
      count: a.pending_count,
      title: a.pending_count === 1 ? 'Announcement to approve' : 'Announcements to approve',
      detail: waitingDetail(a.oldest_submitted_at, now, a.pending_count === 1),
      to: '/approvals',
    })
  }
  return [...rows, ...postsRow(data)]
}

function WaitingForYou({ rows }: { rows: WaitingRow[] }) {
  return <CountSection id="waiting-h" title="Waiting for you" rows={rows} empty="Nothing is waiting for you." />
}

// CountSection is a panel of counts, each opening where the work is; the
// first, most pressing one is tinted.
function CountSection({ id, title, rows, empty }: { id: string; title: string; rows: WaitingRow[]; empty: string }) {
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2 rounded-xl border border-line bg-surface px-[18px] py-4 lg:px-7 lg:py-6">
      <h2 id={id} className="font-serif text-xl font-medium lg:text-[22px]">
        {title}
      </h2>
      {rows.length === 0 ? (
        <p className="text-[15px] text-ink-2">{empty}</p>
      ) : (
        <ul className="-mx-2 flex flex-col gap-1 lg:-mx-3">
          {rows.map((row, i) => (
            <li key={row.title}>
              <CountLink row={row} highlight={i === 0} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function CountLink({ row, highlight }: { row: WaitingRow; highlight: boolean }) {
  const className = cn('flex items-center gap-3.5 rounded-lg px-3.5 py-3 text-ink lg:gap-4 lg:px-4 lg:py-3.5', highlight && 'bg-paper')
  const body = (
    <>
      <span className={cn('min-w-7 font-mono text-[22px] font-medium lg:text-[26px]', highlight ? 'text-rust' : 'text-ink')}>{row.count}</span>
      <span className="flex grow flex-col gap-0.5">
        <span className="text-[15px] font-semibold lg:text-base">{row.title}</span>
        {row.detail && <span className={cn('text-[13px]', row.tone === 'warning' ? 'text-warning-ink' : 'text-ink-3')}>{row.detail}</span>}
      </span>
      {row.to && (
        <span aria-hidden="true" className="text-sm font-semibold whitespace-nowrap text-rust">
          <span className="hidden lg:inline">{row.action ?? 'Review'} </span>→
        </span>
      )}
    </>
  )
  if (!row.to) return <div className={className}>{body}</div>
  return (
    <Link to={row.to} className={cn(className, 'hover:bg-paper hover:text-ink')}>
      {body}
    </Link>
  )
}

function DepartmentSection({ department, lists }: { department: DepartmentPanel; lists?: Lists }) {
  const waiting = lists?.waiting_count ?? 0
  return (
    <section aria-labelledby="dept-h" className="flex flex-col gap-3.5 rounded-xl border border-line bg-surface px-5 py-5 lg:px-7 lg:py-6">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <h2 id="dept-h" className="font-serif text-2xl font-medium">
          {department.name}
        </h2>
        <span className="font-mono text-[13px] text-ink-3">
          {department.students} {department.students === 1 ? 'student' : 'students'} · {department.staff} staff
        </span>
      </div>
      {department.students_by_batch.length > 0 ? (
        <dl className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
          {department.students_by_batch.map((batch) => (
            <div key={batch.batch_year} className="flex flex-col-reverse gap-0.5 rounded-lg bg-paper px-3.5 py-3">
              <dt className="text-[13px] text-ink-2">Batch {batch.batch_year}</dt>
              <dd className="font-mono text-[22px] font-medium">{batch.count}</dd>
            </div>
          ))}
        </dl>
      ) : (
        <p className="text-[15px] text-ink-2">
          {waiting > 0 ? 'No students have signed in yet.' : 'No students yet. Import a class list so they can sign in.'}
        </p>
      )}
      {waiting > 0 && (
        <Link to="/not-signed-in" className="flex items-center justify-between gap-3 rounded-lg bg-warning-soft px-3.5 py-2.5 text-sm text-warning-ink hover:text-warning-ink">
          <span>
            <b className="font-semibold">
              {waiting} {waiting === 1 ? 'person' : 'people'}
            </b>{' '}
            on your class lists haven't signed in yet
          </span>
          <span className="font-semibold whitespace-nowrap">See who →</span>
        </Link>
      )}
      {department.upcoming_events.length > 0 && (
        <div className="flex flex-col">
          <span className="pt-1 text-[13px] font-semibold text-ink-2">Coming up in your department</span>
          <ul>
            {department.upcoming_events.map((event) => (
              <li key={event.id}>
                <Link to={`/events/${event.id}`} className="grid grid-cols-[92px_1fr] gap-3 py-2.5 text-ink hover:text-ink">
                  <span className="pt-0.5 font-mono text-xs text-ink-3">
                    {new Date(event.starts_at).toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', timeZone: COLLEGE_TIME_ZONE })}
                  </span>
                  <span className="flex flex-col gap-0.5">
                    <span className="text-[15px] font-semibold">{event.title}</span>
                    <span className="text-[13px] text-ink-3">{event.location}</span>
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="flex flex-wrap gap-x-5 gap-y-1">
        <Link to={`/departments/${department.code}`} className="inline-flex min-h-11 items-center text-sm font-semibold">
          Department page →
        </Link>
        <Link to="/import" className="inline-flex min-h-11 items-center text-sm font-semibold">
          Import students →
        </Link>
        <Link to="/staff" className="inline-flex min-h-11 items-center text-sm font-semibold">
          Add faculty →
        </Link>
      </div>
    </section>
  )
}

// addHODLink opens Add staff with HOD and the department already chosen.
function addHODLink(department: CollegeDepartment) {
  return `/admin/staff?role=hod&department=${encodeURIComponent(department.code)}`
}

function CollegeSection({ departments }: { departments: CollegeDepartment[] }) {
  const students = departments.reduce((sum, d) => sum + d.students, 0)
  const staff = departments.reduce((sum, d) => sum + d.staff, 0)
  return (
    <section aria-labelledby="college-h" className="flex flex-col gap-3.5 rounded-xl border border-line bg-surface px-5 py-5 lg:px-7 lg:py-6">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <h2 id="college-h" className="font-serif text-2xl font-medium">
          The college
        </h2>
        <span className="font-mono text-[13px] text-ink-3">
          {departments.length} {departments.length === 1 ? 'department' : 'departments'} · {students.toLocaleString('en-IN')} students · {staff} staff
        </span>
      </div>
      <table className="hidden w-full border-collapse text-[15px] lg:table">
        <thead>
          <tr className="text-xs text-ink-3">
            <th scope="col" className="px-3 pb-2 text-left font-semibold">Department</th>
            <th scope="col" className="px-3 pb-2 text-right font-semibold">Students</th>
            <th scope="col" className="px-3 pb-2 text-right font-semibold">Staff</th>
            <th scope="col" className="px-3 pb-2 text-left font-semibold">HOD</th>
          </tr>
        </thead>
        <tbody>
          {departments.map((d, i) => (
            <tr key={d.code} className={cn(i % 2 === 0 && 'bg-paper')}>
              <td className="rounded-l-lg px-3 py-2.5">
                <span className="inline-block w-[30px] font-mono text-xs text-ink-3">{d.code}</span>
                <Link to={`/departments/${d.code}`} className="font-semibold text-ink hover:text-rust">
                  {d.name}
                </Link>
              </td>
              <td className="px-3 py-2.5 text-right font-mono text-sm">{d.students}</td>
              <td className="px-3 py-2.5 text-right font-mono text-sm">{d.staff}</td>
              <td className="rounded-r-lg px-3 py-2.5">
                {d.hod ? <HODCell hod={d.hod} /> : <NoHOD department={d} />}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <ul className="flex flex-col gap-1 lg:hidden">
        {departments.map((d) => (
          <li key={d.code} className="flex items-center gap-3 rounded-lg bg-paper px-3.5 py-3">
            <span className="w-7 font-mono text-xs text-ink-3">{d.code}</span>
            <span className="flex grow flex-col gap-0.5">
              <Link to={`/departments/${d.code}`} className="text-[15px] font-semibold text-ink">
                {d.name}
              </Link>
              <span className="text-[13px] text-ink-3">
                {d.students} students · {d.staff} staff
              </span>
              {d.hod ? d.hod.state !== 'active' && <HODCell hod={d.hod} /> : <NoHOD department={d} />}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

// HODCell names the HOD, with a tag when they can't act yet: added but not
// signed in, or paused (#207). Only an active HOD's profile is linked; the
// others may not be listed in People.
function HODCell({ hod }: { hod: NonNullable<CollegeDepartment['hod']> }) {
  if (hod.state === 'active') {
    return (
      <Link to={`/people/${hod.username}`} className="font-medium text-ink hover:text-rust">
        {hod.full_name}
      </Link>
    )
  }
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      <span className="text-[13px] font-medium text-ink lg:text-[15px]">{hod.full_name}</span>
      <span
        className={cn(
          'rounded px-[7px] py-0.5 text-[11px] font-semibold whitespace-nowrap',
          hod.state === 'paused' ? 'bg-warning-soft text-warning-ink' : 'bg-well text-ink-2',
        )}
      >
        {hod.state === 'paused' ? 'Paused' : "Hasn't signed in yet"}
      </span>
    </span>
  )
}

function NoHOD({ department }: { department: CollegeDepartment }) {
  return (
    <span className="inline-flex flex-wrap items-center gap-2.5">
      <span className="rounded bg-warning-soft px-[7px] py-0.5 text-[11px] font-semibold whitespace-nowrap text-warning-ink">No HOD yet</span>
      <Link to={addHODLink(department)} className="inline-flex min-h-11 items-center text-sm font-semibold whitespace-nowrap lg:min-h-0">
        Add HOD
      </Link>
    </span>
  )
}

function ToDo({ data, now }: { data: Dashboard; now: Date }) {
  const rows = requestsRow(data, now, '/admin/requests')
  const noHOD = (data.college?.departments ?? []).filter((d) => !d.hod)
  if (noHOD.length > 0) {
    rows.push({
      count: noHOD.length,
      title: noHOD.length === 1 ? 'Department with no HOD' : 'Departments with no HOD',
      detail: noHOD.map((d) => d.name).join(', '),
      to: addHODLink(noHOD[0]),
      action: 'Add HOD',
    })
  }
  const waiting = data.lists?.waiting_count ?? 0
  if (waiting > 0) {
    rows.push({
      count: waiting,
      title: waiting === 1 ? "Person hasn't signed in yet" : "People haven't signed in yet",
      detail: 'added from class lists and staff invites',
      to: '/admin/not-signed-in',
      action: 'See who',
    })
  }
  const a = data.approvals
  const queued = (a?.pending_count ?? 0) + (a?.events_pending_count ?? 0)
  if (queued > 0) {
    rows.push({ count: queued, title: 'In the approval queue', detail: "usually the principal's to approve", to: '/approvals', action: 'Open' })
  }
  rows.push(...postsRow(data))

  return (
    <CountSection id="todo-h" title="To do" rows={rows} empty="Nothing needs you. Every department has an HOD and nobody is waiting to get in." />
  )
}

function RecentImports({ lists, now }: { lists: Lists; now: Date }) {
  return (
    <section aria-labelledby="imports-h" className="flex flex-col gap-2 rounded-xl border border-line bg-surface px-[18px] py-4 lg:px-7 lg:py-6">
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="imports-h" className="font-serif text-xl font-medium lg:text-[22px]">
          Recently added
        </h2>
        <Link to="/admin/import" className="text-sm font-semibold">
          Import →
        </Link>
      </div>
      <ul className="flex flex-col">
        {lists.recent_imports.map((run) => (
          <li key={run.imported_at} className="flex flex-col gap-0.5 py-2.5">
            <span className="text-[15px] font-semibold">
              {(run.batches ?? []).map((b) => `${b.department_code} ${b.batch_year}`).join(', ') || 'Class list'}
            </span>
            <span className="text-[13px] text-ink-2">
              {run.created} added{run.failed > 0 && ` · ${run.failed} not added`}
            </span>
            <span className="text-[13px] text-ink-3">
              {dayLabel(run.imported_at, now)}, {run.imported_by.full_name}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

function dayLabel(iso: string, now: Date): string {
  const days = collegeDay(now) - collegeDay(iso)
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  return new Date(iso).toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', timeZone: COLLEGE_TIME_ZONE })
}

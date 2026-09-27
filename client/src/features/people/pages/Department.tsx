import type { ReactNode } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { Avatar } from '../../../app/shell/Avatar'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useDepartmentOverview, useDirectory } from '../api'
import { BackLink } from '../components/BackLink'
import { roleLine } from '../labels'
import type { Entry, Overview } from '../types'

// How many faculty and coordinators show before "All 36 faculty →".
const SHOWN = 5

// Department is a Department's page: who leads it, who teaches in it and how
// many students it has, for anyone wondering "who do I ask in CS?".
export default function Department() {
  const { code = '' } = useParams()
  const upper = code.toUpperCase()
  const overview = useDepartmentOverview(upper)

  return (
    <div className="flex flex-col gap-4 lg:gap-[22px]">
      <BackLink />
      {overview.isPending ? (
        <>
          <LoadingStatus label="Loading the department" />
          <DepartmentSkeleton />
        </>
      ) : overview.isError ? (
        overview.error instanceof ApiRequestError && overview.error.status === 404 ? (
          <div className="max-w-[640px]">
            <EmptyState title={`No department called “${upper}”`}>
              <p>The link may be out of date. Every department is listed under People.</p>
              <Link to="/people?department=all" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
                Browse People
              </Link>
            </EmptyState>
          </div>
        ) : (
          <ErrorState message="This department could not be loaded." onRetry={() => overview.refetch()} />
        )
      ) : (
        <DepartmentView overview={overview.data} />
      )}
    </div>
  )
}

function DepartmentView({ overview }: { overview: Overview }) {
  const { department, hod, counts, staff } = overview
  const location = useLocation()
  const back = location.pathname
  const coordinators = useDirectory({ department: department.code, role: 'student_coordinator' })
  const coordinatorList = coordinators.data?.pages[0]?.data ?? []
  const coordinatorTotal = coordinators.data?.pages[0]?.meta?.total ?? 0
  const faculty = staff.filter((person) => person.username !== hod?.username && person.roles.includes('faculty'))
  const placement = staff.filter((person) => person.username !== hod?.username && person.roles.includes('placement_officer'))
  const people = (role: string) => `/people?department=${department.code}&role=${role}`
  // The faculty count includes an HOD who also teaches, who is listed above,
  // so the "All" link shows only when someone is left out.
  const hodTeaches = hod?.roles.includes('faculty') ? 1 : 0
  const facultyHidden = counts.faculty - hodTeaches > Math.min(faculty.length, SHOWN)

  return (
    <>
      <header className="flex flex-col gap-1.5 px-1 lg:max-w-[1120px] lg:gap-2 lg:px-0">
        <span className="self-start rounded bg-tag-department px-[7px] py-0.5 font-mono text-xs text-tag-department-ink lg:px-2 lg:text-[13px]">{department.code}</span>
        <h1 className="font-serif text-[27px] leading-[1.15] font-medium lg:text-[40px] lg:leading-[1.1] lg:tracking-[-0.6px]">{department.name}</h1>
        {department.description && <p className="text-[13px] text-ink-2 lg:max-w-[720px] lg:text-[15px]">{department.description}</p>}
        <p className="text-[13px] text-ink-2 lg:hidden">
          <span className="font-mono">{counts.students}</span> students · <span className="font-mono">{counts.faculty}</span> faculty
        </p>
      </header>

      <div className="grid items-start gap-4 lg:max-w-[1120px] lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-7">
        <div className="flex flex-col gap-4 lg:gap-[18px]">
          <Group title="Head of department">
            {hod ? <StaffRow person={hod} back={back} /> : <None>No head of department is listed.</None>}
          </Group>
          <Group
            title="Faculty"
            more={facultyHidden && <MoreLink to={people('faculty')}>All {counts.faculty} faculty →</MoreLink>}
          >
            {faculty.length > 0 ? faculty.slice(0, SHOWN).map((person) => <StaffRow key={person.username} person={person} back={back} />) : <None>No faculty with a visible profile yet.</None>}
          </Group>
          {placement.length > 0 && (
            <Group title="Placement office">
              {placement.map((person) => (
                <StaffRow key={person.username} person={person} back={back} />
              ))}
            </Group>
          )}
          {coordinatorList.length > 0 && (
            <Group
              title="Student coordinators"
              more={coordinatorTotal > SHOWN && <MoreLink to={people('student_coordinator')}>All {coordinatorTotal} coordinators →</MoreLink>}
            >
              {coordinatorList.slice(0, SHOWN).map((person) => (
                <StaffRow key={person.username} person={person} back={back} />
              ))}
            </Group>
          )}
        </div>
        <Numbers overview={overview} />
      </div>
    </>
  )
}

function Group({ title, children, more }: { title: string; children: ReactNode; more?: ReactNode }) {
  return (
    <section className="flex flex-col gap-1.5">
      <h2 className="px-1 text-[13px] font-semibold text-ink-3">{title}</h2>
      <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface [&>li+li]:shadow-[0_-1px_0_#efe9de]">{children}</ul>
      {more}
    </section>
  )
}

function None({ children }: { children: ReactNode }) {
  return <li className="px-3.5 py-4 text-sm text-ink-3">{children}</li>
}

function MoreLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to={to} className="inline-flex min-h-10 items-center self-start px-1 text-sm font-semibold">
      {children}
    </Link>
  )
}

// StaffRow shows what someone does (their headline) under their name, which
// helps more than repeating the Department on its own page.
function StaffRow({ person, back }: { person: Entry; back: string }) {
  return (
    <li>
      <Link
        to={`/people/${encodeURIComponent(person.username)}`}
        state={{ back }}
        className="flex min-h-14 items-center gap-3 px-3.5 py-1.5 text-ink hover:bg-paper hover:text-ink"
      >
        <Avatar name={person.full_name} />
        <span className="flex min-w-0 flex-col">
          <span className="truncate text-[15px] font-semibold">{person.full_name}</span>
          <span className="truncate text-[13px] text-ink-3">{person.headline ?? roleLine(person)}</span>
        </span>
      </Link>
    </li>
  )
}

// Numbers counts everyone, private profiles included: a number reveals no one.
function Numbers({ overview }: { overview: Overview }) {
  const { department, counts } = overview
  return (
    <aside className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-[18px] lg:p-[22px]">
      <h2 className="font-serif text-xl font-medium">In numbers</h2>
      <p className="flex items-baseline gap-2.5">
        <span className="font-mono text-[30px] font-medium">{counts.students}</span>
        <span className="text-sm text-ink-2">students</span>
      </p>
      {counts.students_by_batch.length > 0 && (
        <dl className="flex flex-col gap-1.5 rounded-lg bg-paper px-3 py-2.5 text-sm">
          {counts.students_by_batch.map((batch) => (
            <div key={batch.batch_year} className="flex justify-between">
              <dt className="text-ink-2">Batch {batch.batch_year}</dt>
              <dd className="font-mono">{batch.count}</dd>
            </div>
          ))}
        </dl>
      )}
      <p className="flex items-baseline gap-2.5">
        <span className="font-mono text-[30px] font-medium">{counts.faculty}</span>
        <span className="text-sm text-ink-2">faculty</span>
      </p>
      <Link to={`/people?department=${department.code}`} className="inline-flex min-h-9 items-center text-sm font-semibold">
        See everyone in {department.code} →
      </Link>
    </aside>
  )
}

function DepartmentSkeleton() {
  return (
    <div className="flex flex-col gap-4 lg:max-w-[1120px]">
      <Skeleton className="h-5 w-10" />
      <Skeleton className="h-9 w-80 max-w-full" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <div className="flex flex-col gap-2 rounded-xl border border-line bg-surface p-4">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-4 w-56" />
        <Skeleton className="h-4 w-48" />
      </div>
    </div>
  )
}

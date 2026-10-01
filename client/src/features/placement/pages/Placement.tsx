import { Plus } from 'lucide-react'
import { Link, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { dayAndDate } from '../../events/standing'
import { Tag } from '../../events/components/Tags'
import { DeadlineTile } from '../../jobs/components/DeadlineTile'
import { daysLeft, isUrgent } from '../../jobs/format'
import { typeLabel, type Opportunity } from '../../jobs/types'
import { useManaged, type ManagedStatus } from '../api'
import { Pipeline } from '../components/Pipeline'
import { describeEligibility } from '../eligibility'

const views: { value: ManagedStatus; label: string }[] = [
  { value: 'published', label: 'Open' },
  { value: 'draft', label: 'Drafts' },
  { value: 'closed', label: 'Closed' },
]

function isView(value: string | null): value is ManagedStatus {
  return views.some((view) => view.value === value)
}


// Placement is the office's list of every Opportunity: open ones, drafts and
// closed ones, each with its applicant counts. Desktop shows a table; phones
// a list.
export default function Placement() {
  const [params] = useSearchParams()
  const raw = params.get('status')
  const status: ManagedStatus = isView(raw) ? raw : 'published'
  const list = useManaged(status)
  const items = list.data?.pages.flatMap((page) => page.data) ?? []

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <header className="flex items-end justify-between gap-4 px-1 lg:px-0">
        <div className="flex flex-col gap-1.5">
          <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">Placement</h1>
          <p className="hidden text-[15px] text-ink-2 lg:block">Every drive the placement office runs. Anyone in the office can edit any of them.</p>
        </div>
        <Link to="/placement/new" className={cn(buttonStyles.primary, 'hidden hover:text-paper lg:inline-flex')}>
          <Plus aria-hidden="true" className="size-4" />
          New opportunity
        </Link>
        <Link
          to="/placement/new"
          aria-label="New opportunity"
          className="flex size-11 items-center justify-center rounded-full bg-ink text-paper hover:text-paper lg:hidden"
        >
          <Plus aria-hidden="true" className="size-5" />
        </Link>
      </header>

      <nav aria-label="Which opportunities" className="flex gap-1 self-start rounded-[10px] bg-well p-1">
        {views.map((view) => {
          const active = view.value === status
          return (
            <Link
              key={view.value}
              to={view.value === 'published' ? '/placement' : `?status=${view.value}`}
              replace
              aria-current={active ? 'true' : undefined}
              className={cn(
                'flex min-h-9 items-center rounded-[7px] px-3.5 text-sm lg:px-[18px]',
                active ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)] hover:text-ink' : 'text-ink-2 hover:text-ink',
              )}
            >
              {view.label}
            </Link>
          )
        })}
      </nav>

      {list.isPending ? (
        <>
          <LoadingStatus label="Loading opportunities" />
          <ListSkeleton />
        </>
      ) : list.isError && items.length === 0 ? (
        <ErrorState message="Opportunities could not be loaded." onRetry={() => list.refetch()} />
      ) : items.length === 0 ? (
        <Empty status={status} />
      ) : (
        <>
          <Table items={items} status={status} />
          <PhoneList items={items} />
          {list.hasNextPage && (
            <button
              type="button"
              onClick={() => list.fetchNextPage()}
              disabled={list.isFetchingNextPage}
              className="min-h-11 self-start px-1 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3"
            >
              {list.isFetchingNextPage ? 'Loading more…' : 'Show more'}
            </button>
          )}
          <p className="hidden text-[13px] text-ink-3 lg:block">
            Newest first. A drive moves to Closed when you close it; one whose deadline has passed says so.
          </p>
        </>
      )}
    </div>
  )
}

function Deadline({ item }: { item: Opportunity }) {
  const open = item.status === 'published' && new Date(item.apply_by) > new Date()
  return (
    <span className="flex flex-col gap-0.5">
      <span className="text-sm">{dayAndDate(item.apply_by)}</span>
      {open ? (
        <span className={cn('font-mono text-xs', isUrgent(item.apply_by) ? 'text-warning' : 'text-ink-3')}>{daysLeft(item.apply_by)}</span>
      ) : item.status === 'published' ? (
        <span className="font-mono text-xs text-ink-3">deadline passed</span>
      ) : null}
    </span>
  )
}

function Table({ items, status }: { items: Opportunity[]; status: ManagedStatus }) {
  return (
    <div className="hidden overflow-hidden rounded-xl border border-line bg-surface lg:block">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-paper">
          <tr className="text-left font-mono text-[11px] tracking-[1px] text-ink-3 uppercase">
            <th scope="col" className="px-3.5 py-2.5 font-medium">Opportunity</th>
            <th scope="col" className="w-[110px] px-3.5 py-2.5 font-medium">Type</th>
            <th scope="col" className="w-[220px] px-3.5 py-2.5 font-medium">Who can apply</th>
            <th scope="col" className="w-[150px] px-3.5 py-2.5 font-medium">Apply by</th>
            <th scope="col" className="w-[250px] px-3.5 py-2.5 font-medium">{status === 'draft' ? 'Last edited' : 'Applicants'}</th>
          </tr>
        </thead>
        <tbody className="[&>tr+tr]:shadow-[inset_0_1px_0_#efe9de]">
          {items.map((item) => (
            <tr key={item.id} className="relative hover:bg-paper">
              <td className="px-3.5 py-3">
                {/* The whole row opens it; the link covers the row. */}
                <Link to={`/placement/${item.id}`} className="flex flex-col gap-0.5 text-ink after:absolute after:inset-0 hover:text-ink">
                  <span className="text-[15px] font-semibold">{item.title}</span>
                  <span className="text-[13px] text-ink-2">{item.company}</span>
                </Link>
              </td>
              <td className="px-3.5 py-3">
                <Tag tone="placement">{typeLabel(item.opportunity_type)}</Tag>
              </td>
              <td className="px-3.5 py-3 text-[13px] text-ink-2">{describeEligibility(item.eligibility)}</td>
              <td className="px-3.5 py-3">
                <Deadline item={item} />
              </td>
              <td className="px-3.5 py-3">
                {status === 'draft' ? (
                  <span className="text-[13px] text-ink-3">{dayAndDate(item.updated_at)}</span>
                ) : (
                  <Pipeline counts={item.applicant_counts} className="w-[230px]" />
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function PhoneList({ items }: { items: Opportunity[] }) {
  return (
    <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:hidden [&>li+li]:shadow-[0_-1px_0_#efe9de]">
      {items.map((item) => (
        <li key={item.id}>
          <Link to={`/placement/${item.id}`} className="flex gap-3 px-3.5 py-3 text-ink hover:bg-paper hover:text-ink">
            <DeadlineTile iso={item.apply_by} />
            <span className="flex min-w-0 flex-1 flex-col gap-1.5">
              <span className="flex flex-col gap-0.5">
                <span className="text-base leading-[1.3] font-semibold">{item.title}</span>
                <span className="text-[13px] text-ink-3">
                  {item.company} · {describeEligibility(item.eligibility)}
                </span>
              </span>
              {item.status !== 'draft' && <Pipeline counts={item.applicant_counts} />}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  )
}

function Empty({ status }: { status: ManagedStatus }) {
  if (status === 'draft') return <EmptyState title="No drafts">Opportunities you save without publishing wait here.</EmptyState>
  if (status === 'closed') return <EmptyState title="Nothing closed yet">Drives you close show here, with their applicants.</EmptyState>
  return (
    <EmptyState title="Post the first drive">
      <p>Add a job, internship or training with its deadline and who can apply. Eligible students see it in Jobs as soon as you publish.</p>
      <Link to="/placement/new" className={cn(buttonStyles.primary, 'mt-3 hover:text-paper')}>
        <Plus aria-hidden="true" className="size-4" />
        New opportunity
      </Link>
    </EmptyState>
  )
}

function ListSkeleton() {
  return (
    <div aria-hidden="true" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4">
      {Array.from({ length: 4 }, (_, i) => (
        <div key={i} className="flex items-center gap-4">
          <Skeleton className="h-10 w-1/3" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-1/5" />
          <Skeleton className="h-4 w-1/6" />
        </div>
      ))}
    </div>
  )
}

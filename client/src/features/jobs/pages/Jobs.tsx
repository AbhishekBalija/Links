import { useEffect, useRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { Link, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useJobFeed } from '../api'
import { JobList, JobListSkeleton } from '../components/JobList'
import { isOpportunityType, opportunityTypes, typeLabel, type JobState } from '../types'

const views: { value: JobState; label: string }[] = [
  { value: 'open', label: 'Open' },
  { value: 'applied', label: 'Applied' },
  { value: 'closed', label: 'Closed' },
]

function isJobState(value: string | null): value is JobState {
  return views.some((view) => view.value === value)
}

// Jobs lists the Opportunities a student can apply to, soonest deadline
// first, what they applied to, and closed ones. The view and type live in
// the address bar, so back and reload keep them.
export default function Jobs() {
  const [params, setParams] = useSearchParams()
  const rawState = params.get('state')
  const state: JobState = isJobState(rawState) ? rawState : 'open'
  const rawType = params.get('type')
  const type = isOpportunityType(rawType) ? rawType : null
  const feed = useJobFeed(state, type)
  const jobs = feed.data?.pages.flatMap((page) => page.data) ?? []

  function setType(value: string | null) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (value) updated.set('type', value)
        else updated.delete('type')
        return updated
      },
      { replace: true },
    )
  }

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <header className="flex flex-col gap-1.5 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">Jobs</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">Jobs, internships and training you can apply to, soonest deadline first.</p>
      </header>

      <div className="flex items-center gap-2 lg:gap-3">
        <nav aria-label="Which jobs" className="flex gap-1 rounded-[10px] bg-well p-1">
          {views.map((view) => {
            const active = view.value === state
            return (
              <Link
                key={view.value}
                to={`?${new URLSearchParams({ ...(view.value !== 'open' && { state: view.value }), ...(type && { type }) })}`}
                replace
                aria-current={active ? 'true' : undefined}
                className={cn(
                  'flex min-h-9 items-center justify-center rounded-[7px] px-3.5 text-sm lg:px-[18px]',
                  active ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)] hover:text-ink' : 'text-ink-2 hover:text-ink',
                )}
              >
                {view.label}
              </Link>
            )
          })}
        </nav>
        <label className="relative ml-auto flex lg:ml-0">
          <span className="sr-only">Type of opportunity</span>
          <select
            value={type ?? ''}
            onChange={(e) => setType(e.target.value || null)}
            className={cn(
              'min-h-10 cursor-pointer appearance-none rounded-full border py-0 pr-9 pl-3.5 text-sm font-medium lg:min-h-11 lg:rounded-[10px]',
              type ? 'border-ink bg-ink text-paper' : 'border-[#d6ccbb] bg-surface text-ink',
            )}
          >
            <option value="" className="bg-surface text-ink">
              All types
            </option>
            {opportunityTypes.map((t) => (
              <option key={t.value} value={t.value} className="bg-surface text-ink">
                {t.label}
              </option>
            ))}
          </select>
          <ChevronDown aria-hidden="true" className={cn('pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2', type ? 'text-paper' : 'text-ink-3')} />
        </label>
      </div>

      {state === 'applied' && jobs.length > 0 && (
        <p className="px-1 text-[13px] text-ink-2 lg:text-sm">Everything you applied to. Statuses change when the placement office reviews.</p>
      )}

      {feed.isPending ? (
        <>
          <LoadingStatus label="Loading jobs" />
          <JobListSkeleton />
        </>
      ) : feed.isError && jobs.length === 0 ? (
        <ErrorState message="Jobs could not be loaded." onRetry={() => feed.refetch()} />
      ) : jobs.length === 0 ? (
        <Empty state={state} type={type ? typeLabel(type).toLowerCase() : null} />
      ) : (
        <div className="flex max-w-[1040px] flex-col gap-4">
          <JobList jobs={jobs} grouped={state === 'open'} />
          <More hasMore={feed.hasNextPage} loading={feed.isFetchingNextPage} failed={feed.isFetchNextPageError} onMore={() => feed.fetchNextPage()} />
        </div>
      )}
    </div>
  )
}

function Empty({ state, type }: { state: JobState; type: string | null }) {
  const kind = type ? `${type}s` : 'jobs'
  if (state === 'applied') {
    return (
      <EmptyState title="You haven't applied to anything yet">
        <p>Open a job and apply, or mark that you applied on the company's site, and it shows here with its status.</p>
        <Link to="/jobs" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
          See open jobs
        </Link>
      </EmptyState>
    )
  }
  if (state === 'closed') {
    return <EmptyState title={`No closed ${kind} yet`}>Opportunities you could apply to show here once they close.</EmptyState>
  }
  return (
    <EmptyState title={type ? `No ${kind} open for you` : 'Nothing open for you right now'}>
      <p>When the placement office posts a job, internship or training you can apply to, it shows here with its deadline.</p>
      <Link to="/jobs?state=closed" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
        See closed ones
      </Link>
    </EmptyState>
  )
}

type MoreProps = { hasMore: boolean; loading: boolean; failed: boolean; onMore: () => void }

// More loads the next page as the list's end scrolls near. The button does
// the same for keyboard users.
function More({ hasMore, loading, failed, onMore }: MoreProps) {
  const sentinel = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const node = sentinel.current
    if (!node || !hasMore || loading || failed) return
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) onMore()
    }, { rootMargin: '400px' })
    observer.observe(node)
    return () => observer.disconnect()
  }, [hasMore, loading, failed, onMore])

  if (failed) return <ErrorState message="More jobs could not be loaded." onRetry={onMore} />
  if (!hasMore) return null
  return (
    <div ref={sentinel} className="px-1">
      <button type="button" onClick={onMore} disabled={loading} className="min-h-11 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3">
        {loading ? 'Loading more jobs…' : 'Show more jobs'}
      </button>
    </div>
  )
}

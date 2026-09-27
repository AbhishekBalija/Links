import { useEffect, useRef } from 'react'
import { Check, ChevronDown } from 'lucide-react'
import { Link, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useEventFeed } from '../api'
import { EventList, EventListSkeleton } from '../components/EventList'
import { startTime } from '../format'
import { eventTypes, isEventType, typeLabel, type CampusEvent, type Show } from '../types'

const views: { value: Show; label: string }[] = [
  { value: 'upcoming', label: 'Upcoming' },
  { value: 'going', label: 'Going' },
  { value: 'past', label: 'Past' },
]

function isShow(value: string | null): value is Show {
  return views.some((view) => view.value === value)
}

// Events lists what's coming up for the reader, soonest first. The view and
// type live in the address bar, so back and reload keep them.
export default function Events() {
  const [params, setParams] = useSearchParams()
  const rawShow = params.get('show')
  const show: Show = isShow(rawShow) ? rawShow : 'upcoming'
  const rawType = params.get('type')
  const type = isEventType(rawType) ? rawType : null
  const feed = useEventFeed(show, type)
  const events = feed.data?.pages.flatMap((page) => page.data) ?? []
  // The next one they're going to that hasn't started yet.
  const now = new Date()
  const next =
    show === 'upcoming'
      ? events.find((event) => event.rsvp?.my_status === 'going' && event.status !== 'cancelled' && new Date(event.starts_at) > now)
      : undefined

  function set(key: string, value: string | null) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (value) updated.set(key, value)
        else updated.delete(key)
        return updated
      },
      { replace: true },
    )
  }

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <header className="flex flex-col gap-1.5 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">Events</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">What's coming up for you, soonest first.</p>
      </header>

      <div className="flex flex-col gap-2.5 lg:flex-row lg:items-center lg:gap-3">
        <nav aria-label="Which events" className="flex gap-1 rounded-[10px] bg-well p-1 lg:self-start">
          {views.map((view) => {
            const active = view.value === show
            return (
              <Link
                key={view.value}
                to={`?${new URLSearchParams({ ...(view.value !== 'upcoming' && { show: view.value }), ...(type && { type }) })}`}
                replace
                aria-current={active ? 'true' : undefined}
                className={cn(
                  'flex min-h-9 flex-1 items-center justify-center rounded-[7px] px-3.5 text-sm lg:flex-none lg:px-[18px]',
                  active ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)] hover:text-ink' : 'text-ink-2 hover:text-ink',
                )}
              >
                {view.label}
              </Link>
            )
          })}
        </nav>
        <label className="relative flex self-start">
          <span className="sr-only">Type of event</span>
          <select
            value={type ?? ''}
            onChange={(e) => set('type', e.target.value || null)}
            className={cn(
              'min-h-10 cursor-pointer appearance-none rounded-full border py-0 pr-9 pl-3.5 text-sm font-medium lg:min-h-11 lg:rounded-[10px]',
              type ? 'border-ink bg-ink text-paper' : 'border-[#d6ccbb] bg-surface text-ink',
            )}
          >
            <option value="" className="bg-surface text-ink">
              All types
            </option>
            {eventTypes.map((t) => (
              <option key={t.value} value={t.value} className="bg-surface text-ink">
                {t.label}
              </option>
            ))}
          </select>
          <ChevronDown aria-hidden="true" className={cn('pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2', type ? 'text-paper' : 'text-ink-3')} />
        </label>
      </div>

      {next && <NextForYou event={next} />}

      {feed.isPending ? (
        <>
          <LoadingStatus label="Loading events" />
          <EventListSkeleton />
        </>
      ) : feed.isError && events.length === 0 ? (
        <ErrorState message="Events could not be loaded." onRetry={() => feed.refetch()} />
      ) : events.length === 0 ? (
        <Empty show={show} type={type ? typeLabel(type).toLowerCase() : null} />
      ) : (
        <div className="flex max-w-[1040px] flex-col gap-4">
          <EventList events={events} past={show === 'past'} />
          <MoreEvents
            hasMore={feed.hasNextPage}
            loading={feed.isFetchingNextPage}
            failed={feed.isFetchNextPageError}
            onMore={() => feed.fetchNextPage()}
          />
        </div>
      )}
    </div>
  )
}

// NextForYou is the next Event the reader said they're going to, so a
// student opening Events on a phone sees their plan first.
function NextForYou({ event }: { event: CampusEvent }) {
  const day = new Date(event.starts_at).toLocaleDateString('en-IN', { weekday: 'short' })
  return (
    <Link
      to={`/events/${event.id}`}
      className="flex items-center gap-3 rounded-xl bg-success-soft px-3.5 py-3 text-ink hover:text-ink lg:hidden"
    >
      <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-success text-surface">
        <Check aria-hidden="true" className="size-4" strokeWidth={2.4} />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="text-xs font-semibold text-success-ink">
          Next for you · {day}, {startTime(event.starts_at)}
        </span>
        <span className="truncate text-[15px] font-semibold">{event.title}</span>
      </span>
    </Link>
  )
}

function Empty({ show, type }: { show: Show; type: string | null }) {
  const kind = type ? `${type} events` : 'events'
  if (show === 'going') {
    return (
      <EmptyState title="You haven't said Going to anything yet">
        <p>Open an event and answer Going, and it shows here.</p>
        <Link to="/events" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
          See upcoming events
        </Link>
      </EmptyState>
    )
  }
  if (show === 'past') {
    return <EmptyState title={`No past ${kind} yet`}>Events you were invited to show here once they're over.</EmptyState>
  }
  return (
    <EmptyState title={type ? `No ${kind} coming up` : 'Nothing coming up for you'}>
      <p>When an event is published for you, it shows here, soonest first.</p>
      <Link to="/events?show=past" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
        See past events
      </Link>
    </EmptyState>
  )
}

type MoreProps = { hasMore: boolean; loading: boolean; failed: boolean; onMore: () => void }

// MoreEvents loads the next page as the list's end scrolls near. The button
// does the same for keyboard users.
function MoreEvents({ hasMore, loading, failed, onMore }: MoreProps) {
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

  if (failed) return <ErrorState message="More events could not be loaded." onRetry={onMore} />
  if (!hasMore) return null
  return (
    <div ref={sentinel} className="px-1">
      <button type="button" onClick={onMore} disabled={loading} className="min-h-11 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3">
        {loading ? 'Loading more events…' : 'Show more events'}
      </button>
    </div>
  )
}

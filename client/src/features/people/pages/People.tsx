import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { EmptyState, ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { buttonStyles } from '../../announcements/buttons'
import { useAuthStore } from '../../auth/store'
import { useDashboard } from '../../home/api'
import { MIN_SEARCH, useDirectory } from '../api'
import { DesktopFilters } from '../components/DesktopFilters'
import { PersonList, PersonListSkeleton } from '../components/PersonList'
import { PhoneFilters } from '../components/PhoneFilters'
import { SearchBox } from '../components/SearchBox'
import { countLine } from '../labels'
import type { Filters } from '../types'

// ALL marks "every department" in the address bar. Without a department the
// list opens on the member's own Department, the one they look in most.
const ALL = 'all'

export default function People() {
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const isDesktop = useIsDesktop()
  const me = useAuthStore((s) => s.user?.profile)
  const dashboard = useDashboard()

  const own = dashboard.data?.user.department?.code
  const raw = params.get('department')
  // Wait for the member's Department before the first load, unless the
  // address already says which one; a failed Home load means "all".
  const ready = raw !== null || !dashboard.isPending
  const filters: Filters = {
    department: raw === ALL ? undefined : (raw ?? own ?? undefined),
    role: params.get('role') ?? undefined,
    batch: params.get('batch') ?? undefined,
    q: params.get('q') ?? undefined,
  }
  const q = filters.q && filters.q.trim().length >= MIN_SEARCH ? filters.q.trim() : ''
  const directory = useDirectory(filters, ready)
  const people = directory.data?.pages.flatMap((page) => page.data) ?? []
  const total = directory.data?.pages[0]?.meta?.total ?? 0
  const back = location.pathname + location.search

  // The search box updates the address bar after a short pause, so each
  // keystroke doesn't start a request.
  const [text, setText] = useState(filters.q ?? '')
  const typed = useRef(text)
  useEffect(() => {
    if (text === typed.current) return
    const timer = setTimeout(() => {
      typed.current = text
      change({ q: text || undefined })
    }, 250)
    return () => clearTimeout(timer)
  })

  function change(next: Partial<Filters>) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        for (const [key, value] of Object.entries(next)) {
          // Clearing the department means every department, not "back to mine".
          if (key === 'department') updated.set('department', value ?? ALL)
          else if (value) updated.set(key, value)
          else updated.delete(key)
        }
        return updated
      },
      { replace: true },
    )
  }

  function clearFilters() {
    change({ department: undefined, role: undefined, batch: undefined })
  }

  const filtered = Boolean(filters.department || filters.role || filters.batch)
  const scope = filters.department ? `in ${filters.department}` : 'across the whole college'

  return (
    <div className="flex flex-col gap-4 lg:gap-[22px]">
      <header className="flex flex-col gap-1.5 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">People</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">Everyone at the college who has made their profile visible.</p>
      </header>

      <div className="flex max-w-[900px] flex-col gap-3 lg:gap-3.5">
        <SearchBox value={text} onChange={setText} />
        {isDesktop ? <DesktopFilters filters={filters} onChange={change} /> : <PhoneFilters filters={filters} onChange={change} />}
      </div>

      {me && !me.public_profile_enabled && (
        <p role="note" className="max-w-[1040px] rounded-[10px] border border-line bg-surface px-3.5 py-3 text-sm leading-normal text-ink-2">
          You're not listed here because your profile is private, so only you can see it.
        </p>
      )}

      {!ready || directory.isPending ? (
        <>
          <LoadingStatus label="Loading people" />
          <PersonListSkeleton />
        </>
      ) : directory.isError && people.length === 0 ? (
        <ErrorState message="People could not be loaded." onRetry={() => directory.refetch()} />
      ) : people.length === 0 ? (
        <div className="max-w-[900px]">
          {q ? (
            <EmptyState title={filters.department ? `No one in ${filters.department} matches “${q}”` : `No one matches “${q}”`}>
              <p>{filtered ? 'Check the spelling, or search the whole college instead.' : 'Check the spelling, or try part of their name.'}</p>
              {filtered && (
                <button type="button" onClick={clearFilters} className={buttonStyles.secondary + ' mt-3'}>
                  Search the whole college
                </button>
              )}
            </EmptyState>
          ) : (
            <EmptyState title="No one matches these filters">
              <p>No one with a visible profile fits all of them. Try fewer filters.</p>
              <button type="button" onClick={clearFilters} className={buttonStyles.secondary + ' mt-3'}>
                Clear filters
              </button>
            </EmptyState>
          )}
        </div>
      ) : (
        <div className="flex max-w-[1040px] flex-col gap-2 lg:gap-3" aria-busy={directory.isFetching && !directory.isFetchingNextPage}>
          <ListLine
            text={q ? `${total === 1 ? '1 match' : `${total} matches`} for “${q}” ${scope}, best first` : `${countLine(total, filters)}, A to Z`}
            department={filters.department}
            filtered={Boolean(filters.role || filters.batch)}
            onShowAll={() => change({ department: undefined })}
            onClear={clearFilters}
          />
          <PersonList people={people} me={me?.username} q={q} back={back} />
          {!q && (
            <MorePeople
              hasMore={directory.hasNextPage}
              loading={directory.isFetchingNextPage}
              failed={directory.isFetchNextPageError}
              onMore={() => directory.fetchNextPage()}
            />
          )}
        </div>
      )}
    </div>
  )
}

type LineProps = {
  text: string
  department?: string
  filtered: boolean
  onShowAll: () => void
  onClear: () => void
}

// ListLine says what the list holds and offers the next step: widen to every
// department, or open this Department's page.
function ListLine({ text, department, filtered, onShowAll, onClear }: LineProps) {
  return (
    <div className="flex items-center justify-between gap-4 px-1 text-[13px] lg:px-0">
      <p className="text-ink-3">
        {text}
        {department && !filtered && (
          <span className="hidden lg:inline">
            {' · '}
            <button type="button" onClick={onShowAll} className="font-semibold text-rust hover:text-rust-deep">
              Show all departments
            </button>
          </span>
        )}
      </p>
      {filtered ? (
        <button type="button" onClick={onClear} className="min-h-9 shrink-0 font-semibold text-rust hover:text-rust-deep">
          Clear all
        </button>
      ) : (
        department && (
          <Link to={`/departments/${department}`} className="min-h-9 shrink-0 content-center font-semibold lg:text-sm">
            {department} department page →
          </Link>
        )
      )}
    </div>
  )
}

type MoreProps = { hasMore: boolean; loading: boolean; failed: boolean; onMore: () => void }

// MorePeople loads the next page as the list's end scrolls near. The button
// does the same for keyboard users.
function MorePeople({ hasMore, loading, failed, onMore }: MoreProps) {
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

  if (failed) return <ErrorState message="More people could not be loaded." onRetry={onMore} />
  if (!hasMore) return null
  return (
    <div ref={sentinel} className="px-1">
      <button type="button" onClick={onMore} disabled={loading} className="min-h-11 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3">
        {loading ? 'Loading more people…' : 'Show more people'}
      </button>
    </div>
  )
}

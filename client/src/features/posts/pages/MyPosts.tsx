import { Link, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useMine } from '../../announcements/api'
import { hrefFor, standing } from '../../announcements/status'
import type { Authored, MineFilter } from '../../announcements/types'
import { useMyEvents } from '../../events/api'
import { DateTile } from '../../events/components/DateTile'
import { PostTag, StandingTag } from '../../events/components/PostTags'
import { proposalStanding, type PostTab, type ProposalStanding } from '../../events/standing'
import type { CampusEvent, MineEventFilter } from '../../events/types'
import { useHasPosted } from '../api'
import { NewMenu } from '../components/NewMenu'
import { mergeNewestFirst } from '../merge'

type Kind = 'all' | 'announcement' | 'event'

// Post is one row of My posts: an announcement or an event, with where it
// stands put the same way for both.
type Post = { key: string; at: string; kind: 'announcement' | 'event'; title: string; href: string; standing: Pick<ProposalStanding, 'tag' | 'tone' | 'when'>; startsAt?: string }

const tabs: { value: PostTab; label: string; empty: string; announcements: MineFilter; events: MineEventFilter }[] = [
  { value: 'needs', label: 'Needs you', empty: 'Nothing needs you right now.', announcements: 'attention', events: 'attention' },
  { value: 'draft', label: 'Drafts', empty: 'No drafts.', announcements: 'draft', events: 'draft' },
  { value: 'waiting', label: 'Waiting', empty: 'Nothing is waiting for approval.', announcements: 'waiting', events: 'waiting' },
  { value: 'live', label: 'Live', empty: 'Nothing of yours is live right now.', announcements: 'live', events: 'live' },
  { value: 'ended', label: 'Ended', empty: 'Nothing has ended yet.', announcements: 'ended', events: 'ended' },
]

const kinds: { value: Kind; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'announcement', label: 'Announcements' },
  { value: 'event', label: 'Events' },
]

const isTab = (value: string | null): value is PostTab => tabs.some((t) => t.value === value)
const isKind = (value: string | null): value is Kind => kinds.some((k) => k.value === value)

function announcementPost(item: Authored): Post {
  const status = standing(item)
  // Sent back reads as "fix this", like an event with changes asked.
  const tone = status.needsAuthor ? 'warning' : status.tone
  return { key: `a-${item.id}`, at: item.created_at, kind: 'announcement', title: item.title, href: hrefFor(item), standing: { ...status, tone } }
}

function eventPost(event: CampusEvent): Post {
  return { key: `e-${event.id}`, at: event.created_at, kind: 'event', title: event.title, href: `/mine/events/${event.id}`, standing: proposalStanding(event), startsAt: event.starts_at }
}

// MyPosts is everything the author has written or proposed, by what it is
// waiting for. Rows only open things; each post's actions live on its page.
export default function MyPosts() {
  const [params, setParams] = useSearchParams()
  const rawTab = params.get('status')
  const rawKind = params.get('kind')
  const kind: Kind = isKind(rawKind) ? rawKind : 'all'

  // "Needs you" is always loaded: it gives the tab its count and decides
  // where the page opens.
  const needsAnnouncements = useMine('attention')
  const needsEvents = useMyEvents('attention')
  const hasPosted = useHasPosted()
  const needsCount =
    (needsAnnouncements.data?.pages.flatMap((p) => p.data).length ?? 0) +
    (needsEvents.data?.pages.flatMap((p) => p.data).filter((e) => proposalStanding(e).tab === 'needs').length ?? 0)
  const needsLoading = needsAnnouncements.isPending || needsEvents.isPending

  // Without a tab in the address, open on what needs the author, else Live.
  const tab: PostTab = isTab(rawTab) ? rawTab : needsCount > 0 ? 'needs' : 'live'
  const current = tabs.find((t) => t.value === tab) ?? tabs[0]

  const choose = (next: { status?: PostTab; kind?: Kind }) => {
    const query = new URLSearchParams(params)
    if (next.status) query.set('status', next.status)
    if (next.kind) {
      if (next.kind === 'all') query.delete('kind')
      else query.set('kind', next.kind)
    }
    setParams(query, { replace: true })
  }

  const firstTime = hasPosted.data === false

  return (
    <div className="flex flex-col gap-5 lg:gap-5">
      <header className="flex items-center justify-between gap-4 px-1 lg:items-end lg:px-0">
        <div className="flex flex-col gap-1.5">
          <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">
            <span className="lg:hidden">Mine</span>
            <span className="hidden lg:inline">My posts</span>
          </h1>
          <p className="hidden text-[15px] text-ink-2 lg:block">Your announcements and event proposals, and what each is waiting for.</p>
        </div>
        <NewMenu />
      </header>

      {firstTime ? (
        <div className="flex flex-col items-start gap-2 rounded-xl border border-line bg-surface px-6 py-8">
          <h2 className="font-serif text-xl font-medium">Nothing posted yet</h2>
          <p className="text-sm text-ink-2">
            Announcements you write and events you propose show up here, with what each is waiting for. Use <b className="font-semibold text-ink">New</b> to start.
          </p>
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <nav aria-label="Filter by status" className="-mx-4 flex gap-2 overflow-x-auto px-4 lg:mx-0 lg:gap-1 lg:rounded-[10px] lg:bg-well lg:p-1">
              {tabs.map((t) => {
                const active = t.value === tab
                return (
                  <button
                    key={t.value}
                    type="button"
                    aria-pressed={active}
                    onClick={() => choose({ status: t.value })}
                    className={cn(
                      'flex min-h-11 shrink-0 items-center gap-1.5 rounded-full border px-4 text-sm lg:min-h-9 lg:rounded-[7px] lg:border-0 lg:px-3.5',
                      active
                        ? 'border-ink bg-ink font-semibold text-paper lg:bg-surface lg:text-ink lg:shadow-[0_1px_2px_rgba(27,24,20,0.08)]'
                        : 'border-line bg-surface text-ink-2 hover:text-ink lg:bg-transparent',
                    )}
                  >
                    {t.label}
                    {t.value === 'needs' && needsCount > 0 && <span className="font-mono text-xs">{needsCount}</span>}
                  </button>
                )
              })}
            </nav>
            <div role="group" aria-label="Show" className="hidden gap-1 rounded-[10px] bg-well p-1 lg:ml-auto lg:flex">
              {kinds.map((k) => (
                <button
                  key={k.value}
                  type="button"
                  aria-pressed={k.value === kind}
                  onClick={() => choose({ kind: k.value })}
                  className={cn(
                    'min-h-9 rounded-[7px] px-3.5 text-sm',
                    k.value === kind ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2 hover:text-ink',
                  )}
                >
                  {k.label}
                </button>
              ))}
            </div>
          </div>

          {needsLoading && !isTab(rawTab) ? <ListSkeleton /> : <TabList key={tab} tab={current} kind={kind} />}
        </>
      )}
    </div>
  )
}

function TabList({ tab, kind }: { tab: (typeof tabs)[number]; kind: Kind }) {
  const announcements = useMine(tab.announcements)
  const events = useMyEvents(tab.events)
  // A rejected proposal can't be sent again, so it belongs under Ended; the
  // server groups it with "needs attention".
  const rejected = useMyEvents('attention')

  const announcementItems = (announcements.data?.pages.flatMap((p) => p.data) ?? [])
    // A live notice with an edit sent back shows under Needs you only.
    .filter((item) => tab.value === 'needs' || !standing(item).needsAuthor)
    .map(announcementPost)
  const eventItems = (events.data?.pages.flatMap((p) => p.data) ?? []).filter((e) => proposalStanding(e).tab === tab.value).map(eventPost)
  const rejectedItems = tab.value === 'ended' ? (rejected.data?.pages.flatMap((p) => p.data) ?? []).filter((e) => proposalStanding(e).tab === 'ended').map(eventPost) : []

  const sources = []
  if (kind !== 'event') sources.push({ items: announcementItems, complete: !announcements.hasNextPage })
  if (kind !== 'announcement') {
    sources.push({ items: eventItems, complete: !events.hasNextPage })
    if (tab.value === 'ended') sources.push({ items: rejectedItems, complete: !rejected.hasNextPage })
  }
  const merged = mergeNewestFirst(sources)

  const pending = (kind !== 'event' && announcements.isPending) || (kind !== 'announcement' && (events.isPending || (tab.value === 'ended' && rejected.isPending)))
  const failed = (kind !== 'event' && announcements.isError) || (kind !== 'announcement' && events.isError)
  const loadingMore = announcements.isFetchingNextPage || events.isFetchingNextPage || rejected.isFetchingNextPage

  function showMore() {
    if (announcements.hasNextPage && kind !== 'event') announcements.fetchNextPage()
    if (events.hasNextPage && kind !== 'announcement') events.fetchNextPage()
    if (rejected.hasNextPage && tab.value === 'ended' && kind !== 'announcement') rejected.fetchNextPage()
  }

  if (pending) return <ListSkeleton />
  if (failed && merged.items.length === 0) {
    return (
      <ErrorState
        message="Your posts could not be loaded."
        onRetry={() => {
          announcements.refetch()
          events.refetch()
        }}
      />
    )
  }
  if (merged.items.length === 0 && !merged.hasMore) {
    return <p className="rounded-xl border border-line bg-surface px-6 py-6 text-[15px] text-ink-2">{tab.empty}</p>
  }
  return (
    <>
      <ul className="flex max-w-[1080px] flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5">
        {merged.items.map((post) => (
          <li key={post.key} className="[&+li]:shadow-[0_-1px_0_var(--color-rail)] lg:[&+li]:shadow-none">
            <Row post={post} />
          </li>
        ))}
      </ul>
      {tab.value === 'needs' && merged.items.length > 0 && (
        <p className="px-1 text-[13px] text-ink-3 lg:px-0">Open one to read the note, fix it and send it again.</p>
      )}
      {merged.hasMore && (
        <button
          type="button"
          onClick={showMore}
          disabled={loadingMore}
          className="min-h-11 self-start px-1 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3"
        >
          {loadingMore ? 'Loading…' : 'Show older'}
        </button>
      )}
      {failed && merged.items.length > 0 && (
        <ErrorState
          message="Some of your posts could not be loaded."
          onRetry={() => {
            announcements.refetch()
            events.refetch()
          }}
        />
      )}
    </>
  )
}

function Row({ post }: { post: Post }) {
  return (
    <Link
      to={post.href}
      className="flex items-center gap-3.5 px-4 py-3 text-ink hover:text-ink lg:gap-4 lg:rounded-[10px] lg:py-2.5 lg:pr-3.5 lg:pl-2.5 lg:hover:bg-paper"
    >
      {post.startsAt ? <DateTile iso={post.startsAt} size="sm" /> : <span aria-hidden="true" className="hidden w-12 shrink-0 lg:block" />}
      <span className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="flex flex-wrap items-center gap-1.5">
          <PostTag kind={post.kind} />
          <StandingTag standing={post.standing} />
        </span>
        <span className="text-[15px] leading-snug font-semibold lg:text-base">{post.title}</span>
        <span className="text-[13px] text-ink-3 lg:hidden">{post.standing.when}</span>
      </span>
      <span className="hidden text-[13px] whitespace-nowrap text-ink-3 lg:block">{post.standing.when}</span>
    </Link>
  )
}

function ListSkeleton() {
  return (
    <div className="flex max-w-[1080px] flex-col gap-3 rounded-xl border border-line bg-surface p-4">
      <LoadingStatus label="Loading your posts" />
      {Array.from({ length: 4 }, (_, i) => (
        <div key={i} className="flex flex-col gap-1.5">
          <Skeleton className="h-3 w-32" />
          <Skeleton className="h-4 w-3/5" />
        </div>
      ))}
    </div>
  )
}

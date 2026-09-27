import { ChevronLeft } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useAuthStore } from '../../auth/store'
import { useEventReviews } from '../../events/api'
import { EventQueueRow } from '../../events/components/EventQueueRow'
import { EventReview } from '../../events/components/EventReview'
import type { EventQueueItem } from '../../events/types'
import { mergeOldestFirst } from '../../posts/merge'
import { useQueue } from '../api'
import { QueueRow } from '../components/queue/QueueRow'
import { QueueSkeleton } from '../components/queue/QueueSkeleton'
import { Review, type Banner } from '../components/queue/Review'
import type { QueueItem } from '../types'

// Waiting is one item in the queue: an announcement or an event proposal.
type Waiting = { id: string; at: string } & ({ kind: 'announcement'; item: QueueItem } | { kind: 'event'; item: EventQueueItem })

// ApprovalQueue is where reviewers decide on announcements and event
// proposals, oldest first. On desktop the list and the selected item sit side
// by side; on phones each item opens on its own page.
export default function ApprovalQueue() {
  const { id } = useParams()
  const isDesktop = useIsDesktop()
  const navigate = useNavigate()
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  const queue = useQueue()
  const events = useEventReviews()
  const [banner, setBanner] = useState<Banner | null>(null)
  const [focusTitle, setFocusTitle] = useState(false)
  // Both lists page separately, so they merge without skipping ahead.
  const merged = mergeOldestFirst<Waiting>([
    {
      items: (queue.data?.pages.flatMap((page) => page.data) ?? []).map((item) => ({ id: item.id, at: item.submitted_at, kind: 'announcement', item })),
      complete: !queue.hasNextPage,
    },
    {
      items: (events.data?.pages.flatMap((page) => page.data) ?? []).map((item) => ({ id: item.id, at: item.submitted_at ?? item.updated_at, kind: 'event', item })),
      complete: !events.hasNextPage,
    },
  ])
  const items = merged.items
  const selected = items.find((item) => item.id === id) ?? (isDesktop ? items[0] : undefined)

  // After a review, move on: the next item on desktop, back to the list on phones.
  function moveOn(reviewedId: string, next: Banner) {
    setBanner(next)
    const rest = items.filter((item) => item.id !== reviewedId)
    const after = items.slice(items.findIndex((item) => item.id === reviewedId) + 1)[0] ?? rest[0]
    setFocusTitle(true)
    if (isDesktop && after) navigate(`/approvals/${after.id}`, { replace: true })
    else navigate('/approvals', { replace: true })
  }

  const review = (waiting: Waiting) =>
    waiting.kind === 'announcement' ? (
      <Review key={waiting.id} item={waiting.item} focusTitle={focusTitle} onReviewed={(item, next) => moveOn(item.id, next)} />
    ) : (
      <EventReview key={waiting.id} item={waiting.item} focusTitle={focusTitle} onReviewed={moveOn} />
    )
  // The principal and admins see final approvals and Departments without an HOD.
  const subtitle =
    roles.includes('principal') || roles.includes('admin')
      ? 'Oldest first. Final approvals, and anything from departments without an HOD.'
      : 'Oldest first. Announcements and events that need your approval.'

  const header = (
    <header className="flex flex-col gap-1 px-1 lg:px-0">
      <h1 className="flex items-baseline gap-2.5 font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">
        <span className="lg:hidden">Approvals</span>
        <span className="hidden lg:inline">Approval queue</span>
        {items.length > 0 && <span className="font-mono text-sm tracking-normal text-rust lg:text-lg">{items.length}{merged.hasMore ? '+' : ''}</span>}
      </h1>
      <p className="hidden text-[15px] text-ink-2 lg:block">{subtitle}</p>
    </header>
  )

  if (queue.isPending || events.isPending) return <QueueSkeleton header={header} />
  if (queue.isError || events.isError) {
    return (
      <div className="flex flex-col gap-5">
        {header}
        <ErrorState
          message="The queue could not be loaded."
          onRetry={() => {
            queue.refetch()
            events.refetch()
          }}
        />
      </div>
    )
  }

  const bannerView = banner && (
    <div
      role="status"
      className={cn(
        'rounded-[10px] px-4 py-3 text-sm',
        banner.tone === 'done' ? 'bg-success-soft text-success-ink' : 'border border-line bg-surface text-ink',
      )}
    >
      {banner.text}
    </div>
  )

  if (items.length === 0) {
    return (
      <div className="flex flex-col gap-5">
        {header}
        {bannerView}
        <EmptyState title="Nothing waiting for you">
          When someone sends a notice or proposes an event for your approval, it shows up here, oldest first. Your Home page shows the count too.
        </EmptyState>
      </div>
    )
  }

  // Phone, one item open: its own page with the bar at the bottom.
  if (!isDesktop && selected) {
    return (
      <div className="group/page flex flex-col gap-3 pb-44">
        <Link to="/approvals" className="-ml-2 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-2 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-5" />
          Approvals
        </Link>
        {review(selected)}
      </div>
    )
  }

  return (
    <div className="group/page flex flex-col gap-5 pb-4 lg:pb-0">
      {header}
      {bannerView}
      <div className="grid items-start gap-5 lg:grid-cols-[360px_minmax(0,1fr)]">
        <nav aria-label="Waiting for you" className="flex flex-col gap-2.5 lg:sticky lg:top-6 lg:gap-0.5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-1.5">
          {items.map((waiting) =>
            waiting.kind === 'announcement' ? (
              <QueueRow key={waiting.id} item={waiting.item} selected={isDesktop && waiting.id === selected?.id} />
            ) : (
              <EventQueueRow key={waiting.id} item={waiting.item} selected={isDesktop && waiting.id === selected?.id} />
            ),
          )}
          {merged.hasMore && (
            <button
              type="button"
              onClick={() => {
                if (queue.hasNextPage) queue.fetchNextPage()
                if (events.hasNextPage) events.fetchNextPage()
              }}
              className="min-h-11 text-sm font-semibold text-rust"
            >
              {queue.isFetchingNextPage || events.isFetchingNextPage ? 'Loading…' : 'Show more'}
            </button>
          )}
        </nav>
        {isDesktop && selected && review(selected)}
      </div>
    </div>
  )
}

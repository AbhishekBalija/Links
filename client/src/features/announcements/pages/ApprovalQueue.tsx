import { ChevronLeft } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useQueue } from '../api'
import { QueueRow } from '../components/queue/QueueRow'
import { QueueSkeleton } from '../components/queue/QueueSkeleton'
import { Review, type Banner } from '../components/queue/Review'
import type { QueueItem } from '../types'

// ApprovalQueue is where approvers review announcements, oldest first. On
// desktop the list and the selected notice sit side by side; on phones each
// item opens on its own page.
export default function ApprovalQueue() {
  const { id } = useParams()
  const isDesktop = useIsDesktop()
  const navigate = useNavigate()
  const queue = useQueue()
  const [banner, setBanner] = useState<Banner | null>(null)
  const [focusTitle, setFocusTitle] = useState(false)
  const items = queue.data?.pages.flatMap((page) => page.data) ?? []
  const selected = items.find((item) => item.id === id) ?? (isDesktop ? items[0] : undefined)

  // After a review, move on: the next item on desktop, back to the list on phones.
  function moveOn(reviewed: QueueItem, next: Banner) {
    setBanner(next)
    const rest = items.filter((item) => item.id !== reviewed.id)
    const after = items.slice(items.findIndex((item) => item.id === reviewed.id) + 1)[0] ?? rest[0]
    setFocusTitle(true)
    if (isDesktop && after) navigate(`/approvals/${after.id}`, { replace: true })
    else navigate('/approvals', { replace: true })
  }

  const header = (
    <header className="flex flex-col gap-1 px-1 lg:px-0">
      <h1 className="flex items-baseline gap-2.5 font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">
        <span className="lg:hidden">Approvals</span>
        <span className="hidden lg:inline">Approval queue</span>
        {items.length > 0 && <span className="font-mono text-sm tracking-normal text-rust lg:text-lg">{items.length}{queue.hasNextPage ? '+' : ''}</span>}
      </h1>
      <p className="hidden text-[15px] text-ink-2 lg:block">Oldest first. Only announcements that need your approval.</p>
    </header>
  )

  if (queue.isPending) return <QueueSkeleton header={header} />
  if (queue.isError) {
    return (
      <div className="flex flex-col gap-5">
        {header}
        <ErrorState message="The queue could not be loaded." onRetry={() => queue.refetch()} />
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
        <EmptyState title="No announcements waiting for you">
          When someone sends a notice for your approval, it shows up here, oldest first. Your Home page shows the count too.
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
        <Review key={selected.id} item={selected} focusTitle={focusTitle} onReviewed={moveOn} />
      </div>
    )
  }

  return (
    <div className="group/page flex flex-col gap-5 pb-4 lg:pb-0">
      {header}
      {bannerView}
      <div className="grid items-start gap-5 lg:grid-cols-[360px_minmax(0,1fr)]">
        <nav aria-label="Waiting announcements" className="flex flex-col gap-2.5 lg:sticky lg:top-6 lg:gap-0.5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-1.5">
          {items.map((item) => (
            <QueueRow key={item.id} item={item} selected={isDesktop && item.id === selected?.id} />
          ))}
          {queue.hasNextPage && (
            <button type="button" onClick={() => queue.fetchNextPage()} className="min-h-11 text-sm font-semibold text-rust">
              {queue.isFetchingNextPage ? 'Loading…' : 'Show more'}
            </button>
          )}
        </nav>
        {isDesktop && selected && <Review key={selected.id} item={selected} focusTitle={focusTitle} onReviewed={moveOn} />}
      </div>
    </div>
  )
}

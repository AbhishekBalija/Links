import { useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { EmptyState, ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useNoticeFeed } from '../api'
import { useRefetchAtExpiry } from '../useRefetchAtExpiry'
import { CategoryFilter } from '../components/CategoryFilter'
import { NoticeList, NoticeListSkeleton } from '../components/NoticeList'
import { categories, isCategory } from '../types'

export default function Notices() {
  const [params] = useSearchParams()
  const raw = params.get('category')
  const category = isCategory(raw) ? raw : null
  const feed = useNoticeFeed(category)
  const notices = feed.data?.pages.flatMap((page) => page.data) ?? []
  useRefetchAtExpiry(notices.map((n) => n.expires_at), feed.refetch)
  const categoryLabel = categories.find((c) => c.value === category)?.label.toLowerCase()

  return (
    <div className="flex flex-col gap-5 lg:gap-6">
      <header className="flex flex-col gap-1.5 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">
          Notices
        </h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">Everything posted for you, newest first.</p>
      </header>

      <CategoryFilter current={category} />

      {feed.isPending ? (
        <>
          <LoadingStatus label="Loading notices" />
          <NoticeListSkeleton rows={6} />
        </>
      ) : feed.isError && notices.length === 0 ? (
        <ErrorState message="Notices could not be loaded." onRetry={() => feed.refetch()} />
      ) : notices.length === 0 ? (
        <EmptyState title={categoryLabel ? `No ${categoryLabel} notices` : 'No notices yet'}>
          {categoryLabel
            ? 'Nothing in this category is posted for you right now.'
            : 'When something is posted for you, it shows up here.'}
        </EmptyState>
      ) : (
        <>
          <NoticeList notices={notices} />
          <MoreNotices
            hasMore={feed.hasNextPage}
            loading={feed.isFetchingNextPage}
            failed={feed.isFetchNextPageError}
            onMore={() => feed.fetchNextPage()}
          />
        </>
      )}
    </div>
  )
}

type MoreProps = { hasMore: boolean; loading: boolean; failed: boolean; onMore: () => void }

// MoreNotices loads the next page when it scrolls into view. The button does
// the same for keyboard users and for browsers that scroll without firing it.
function MoreNotices({ hasMore, loading, failed, onMore }: MoreProps) {
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

  if (failed) {
    return <ErrorState message="Older notices could not be loaded." onRetry={onMore} />
  }
  if (!hasMore) {
    return <p className="px-1 text-[13px] text-ink-3">You're all caught up.</p>
  }
  return (
    <div ref={sentinel} className="flex items-center gap-3 px-1">
      <button
        type="button"
        onClick={onMore}
        disabled={loading}
        className="min-h-11 rounded-lg px-1 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3"
      >
        {loading ? 'Loading older notices…' : 'Load older notices'}
      </button>
      <span role="status" className="sr-only">
        {loading ? 'Loading older notices' : ''}
      </span>
    </div>
  )
}

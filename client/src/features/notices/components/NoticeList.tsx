import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Skeleton } from '../../../shared/ui/states'
import { useAuthStore } from '../../auth/store'
import { audienceLabel, expiry, timeAgo } from '../format'
import type { Notice } from '../types'
import { CategoryTag } from './CategoryTag'

// On phones each notice is its own card. From lg the list becomes one panel
// of rows, and when that panel is wide enough (a container query, so it works
// in any column) the rows line up as a table: tag, title, sender, time.
const listClass = 'flex flex-col gap-2.5 @container lg:gap-0 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-2'
const rowClass =
  'flex flex-col gap-1.5 rounded-[10px] border border-line bg-surface px-4 py-3.5 lg:rounded-lg lg:border-0 lg:bg-transparent lg:py-3 @3xl:grid @3xl:grid-cols-[112px_minmax(0,1fr)_220px_96px] @3xl:items-center @3xl:gap-5'

export function NoticeList({ notices }: { notices: Notice[] }) {
  const userId = useAuthStore((s) => s.user?.id)
  return (
    <ul className={listClass}>
      {notices.map((notice) => (
        <li key={notice.id}>
          <NoticeRow notice={notice} isMine={notice.publisher_id === userId} />
        </li>
      ))}
    </ul>
  )
}

function NoticeRow({ notice, isMine }: { notice: Notice; isMine: boolean }) {
  const posted = notice.published_at ?? notice.created_at
  const ends = expiry(notice.expires_at)
  const ago = timeAgo(posted)
  return (
    <Link to={`/notices/${notice.id}`} className={cn(rowClass, 'text-ink hover:text-ink lg:hover:bg-paper')}>
      <span className="flex items-center gap-2 @3xl:block">
        <CategoryTag category={notice.category} className="@3xl:text-xs" />
        <time dateTime={posted} className="text-[11px] text-ink-3 @3xl:hidden">
          {ago}
        </time>
      </span>
      <span className="flex flex-col gap-0.5">
        <span className="text-base font-semibold leading-snug @3xl:text-[15px]">{notice.title}</span>
        {ends?.soon && <span className="text-[13px] font-medium text-warning">Expires {ends.text}</span>}
      </span>
      <span className="text-[13px] text-ink-3 @3xl:text-ink-2">
        <span>{isMine ? 'You' : notice.publisher_name}</span>
        <span aria-hidden="true" className="@3xl:hidden">
          {' · '}
        </span>
        <span className="@3xl:block @3xl:text-ink-3">{audienceLabel(notice.audience)}</span>
      </span>
      <time dateTime={posted} className="hidden justify-self-end font-mono text-xs text-ink-3 @3xl:block">
        {ago}
      </time>
    </Link>
  )
}

export function NoticeListSkeleton({ rows }: { rows: number }) {
  return (
    <ul aria-hidden="true" className={listClass}>
      {Array.from({ length: rows }, (_, i) => (
        <li key={i} className={rowClass}>
          <Skeleton className="h-5 w-20" />
          <Skeleton className="h-4 w-4/5" />
          <Skeleton className="h-3.5 w-1/2" />
          <Skeleton className="hidden h-3 w-16 justify-self-end @3xl:block" />
        </li>
      ))}
    </ul>
  )
}

import { ChevronLeft } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useAuthStore } from '../../auth/store'
import { useNotice } from '../api'
import { useRefetchAtExpiry } from '../useRefetchAtExpiry'
import { CategoryTag } from '../components/CategoryTag'
import { audienceLabel, expiry, fullDate, timeAgo, type Expiry } from '../format'
import type { Notice } from '../types'

export default function NoticeDetail() {
  const { id = '' } = useParams()
  const notice = useNotice(id)
  // Once it expires, the refetch comes back 404 and the page says so.
  useRefetchAtExpiry([notice.data?.expires_at ?? null], notice.refetch)

  return (
    <div className="flex flex-col gap-4 lg:gap-7">
      <Link
        to="/notices"
        className="-ml-2 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-2 text-sm font-semibold"
      >
        <ChevronLeft aria-hidden="true" className="size-5 lg:size-4" />
        All notices
      </Link>

      {notice.isPending ? (
        <DetailSkeleton />
      ) : notice.isError ? (
        notice.error instanceof ApiRequestError && notice.error.status === 404 ? (
          <EmptyState title="This notice isn't available">
            It may have expired or been taken down, or it wasn't sent to you.{' '}
            <Link to="/notices" className="font-semibold">
              Back to notices
            </Link>
          </EmptyState>
        ) : (
          <ErrorState message="This notice could not be loaded." onRetry={() => notice.refetch()} />
        )
      ) : (
        <NoticeView notice={notice.data} />
      )}
    </div>
  )
}

function NoticeView({ notice }: { notice: Notice }) {
  const userId = useAuthStore((s) => s.user?.id)
  const posted = notice.published_at ?? notice.created_at
  const ends = expiry(notice.expires_at)
  const paragraphs = notice.body.split(/\n\s*\n/)

  return (
    <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-10">
      <article className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-11 lg:pt-9 lg:pb-11">
        <div className="flex items-center gap-2.5 text-xs">
          <CategoryTag category={notice.category} className="lg:text-xs" />
          <time dateTime={posted} className="text-ink-3">
            Posted {timeAgo(posted)}
          </time>
        </div>
        <h1 className="max-w-[720px] font-serif text-[28px] font-medium leading-[1.15] tracking-[-0.4px] lg:text-[40px] lg:leading-[1.12] lg:tracking-[-0.6px]">
          {notice.title}
        </h1>
        {/* On phones the details sit between the title and the body. */}
        <div className="lg:hidden">
          <Details notice={notice} isMine={notice.publisher_id === userId} posted={posted} ends={ends} compact />
        </div>
        <div className="flex max-w-[680px] flex-col gap-3 font-serif text-lg leading-relaxed text-prose lg:gap-3.5 lg:text-[19px]">
          {paragraphs.map((text, i) => (
            <p key={i} className="whitespace-pre-line">
              {text}
            </p>
          ))}
        </div>
      </article>
      <aside className="hidden lg:block">
        <Details notice={notice} isMine={notice.publisher_id === userId} posted={posted} ends={ends} />
      </aside>
    </div>
  )
}

type DetailsProps = {
  notice: Notice
  isMine: boolean
  posted: string
  ends: Expiry | null
  compact?: boolean
}

// Details is the "who, to whom, until when" of a notice: a small table on
// phones and a side panel on desktop.
function Details({ notice, isMine, posted, ends, compact }: DetailsProps) {
  const sender = isMine ? 'You' : notice.publisher_name
  const audience = audienceLabel(notice.audience)

  if (compact) {
    return (
      <dl className="flex flex-col gap-2.5 rounded-[10px] border border-line bg-surface px-4 py-3.5 text-[13px]">
        <Row label="Posted by">{sender}</Row>
        <Row label="Sent to">{audience}</Row>
        {ends && (
          <Row label="Expires" tone={ends.soon ? 'warning' : undefined}>
            <span className="font-mono font-medium">
              {shortDate(ends.at)} · {ends.text}
            </span>
          </Row>
        )}
      </dl>
    )
  }

  return (
    <div className="flex flex-col gap-[18px] rounded-xl border border-line bg-surface p-6">
      <h2 className="font-serif text-xl font-medium">About this notice</h2>
      <dl className="flex flex-col gap-[18px]">
        <Field label="Posted by">
          <span className="text-[15px] font-semibold">{sender}</span>
        </Field>
        <Field label="Sent to">
          <span className="flex flex-wrap gap-1.5">
            {audience.split('; ').map((part) => (
              <span key={part} className="rounded-full bg-paper px-2.5 py-1 text-[13px] font-medium">
                {part}
              </span>
            ))}
          </span>
        </Field>
        <Field label="Posted">
          <span className="font-mono text-sm">{fullDate(posted)}</span>
        </Field>
        {ends && (
          <div className={ends.soon ? 'flex flex-col gap-1 rounded-lg bg-warning-soft px-3.5 py-3' : 'flex flex-col gap-1'}>
            <dt className={ends.soon ? 'text-xs font-semibold text-warning-ink' : 'text-xs text-ink-3'}>Expires</dt>
            <dd className={ends.soon ? 'font-mono text-sm text-warning-ink' : 'font-mono text-sm'}>
              <span className="block">{fullDate(ends.at)}</span>
              <span className="block font-sans text-[13px] font-semibold">{capitalize(ends.text)}</span>
            </dd>
          </div>
        )}
      </dl>
    </div>
  )
}

function Row({ label, tone, children }: { label: string; tone?: 'warning'; children: ReactNode }) {
  return (
    <div className={tone === 'warning' ? 'flex justify-between gap-4 text-warning-ink' : 'flex justify-between gap-4'}>
      <dt className={tone === 'warning' ? '' : 'text-ink-3'}>{label}</dt>
      <dd className="text-right font-semibold">{children}</dd>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-ink-3">{label}</dt>
      <dd>{children}</dd>
    </div>
  )
}

function capitalize(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

function shortDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })
}

function DetailSkeleton() {
  return (
    <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-10">
      <LoadingStatus label="Loading notice" />
      <div className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-11 lg:py-9">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-9 w-4/5" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-11/12" />
        <Skeleton className="h-4 w-3/5" />
      </div>
      <div className="hidden flex-col gap-4 rounded-xl border border-line bg-surface p-6 lg:flex">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-4 w-40" />
      </div>
    </div>
  )
}

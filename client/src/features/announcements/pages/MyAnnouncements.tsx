import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useDepartments, useMine, useSentBack } from '../api'
import { describe, toRules } from '../audience'
import { buttonStyles } from '../buttons'
import { StatusTag } from '../components/StatusTag'
import { hrefFor, standing } from '../status'
import type { Authored, MineFilter } from '../types'

const tabs: { value: MineFilter | null; label: string; empty: string }[] = [
  { value: null, label: 'All', empty: 'Nothing else here.' },
  { value: 'draft', label: 'Drafts', empty: 'No drafts.' },
  { value: 'waiting', label: 'Waiting', empty: 'Nothing is waiting for approval.' },
  { value: 'live', label: 'Live', empty: 'Nothing of yours is live right now.' },
  { value: 'ended', label: 'Ended', empty: 'Nothing has expired or been withdrawn.' },
]

function isFilter(value: string | null): value is MineFilter {
  return tabs.some((tab) => tab.value === value)
}

// MyAnnouncements lists what the author has written. Anything sent back
// sits in one card at the top, the only place with a button; every other
// notice is a row that opens it, where its actions live.
export default function MyAnnouncements() {
  const [params] = useSearchParams()
  const raw = params.get('status')
  const filter = isFilter(raw) ? raw : null
  const sentBack = useSentBack()
  const list = useMine(filter)
  const departments = useDepartments()
  const codes = new Map((departments.data ?? []).map((d) => [d.id, d.code]))

  // Sent-back ones live only in the card, so "All" leaves them out.
  const items = (list.data?.pages.flatMap((page) => page.data) ?? []).filter((item) => !standing(item).needsAuthor)
  const attention = sentBack.data ?? []
  const firstTime = filter === null && !list.isPending && !sentBack.isPending && items.length === 0 && attention.length === 0 && !list.isError

  return (
    <div className="flex flex-col gap-5 lg:gap-6">
      <header className="flex items-center justify-between gap-4 px-1 lg:items-end lg:px-0">
        <div className="flex flex-col gap-1.5">
          <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[40px] lg:leading-tight lg:tracking-[-0.6px]">
            <span className="lg:hidden">Mine</span>
            <span className="hidden lg:inline">My announcements</span>
          </h1>
          <p className="hidden text-[15px] text-ink-2 lg:block">Open any notice to edit or withdraw it.</p>
        </div>
        <Link to="/mine/new" className={buttonStyles.primary + ' flex-none'}>
          <Plus aria-hidden="true" className="size-4" />
          <span className="lg:hidden">New</span>
          <span className="hidden lg:inline">New announcement</span>
        </Link>
      </header>

      {firstTime ? (
        <EmptyState title="You haven't posted anything yet">
          Announcements you write, and where each one stands, show up here. Use <b className="font-semibold text-ink">New</b> to write your first.
        </EmptyState>
      ) : (
        <>
          {attention.length > 0 && <SentBackCard items={attention} codes={codes} />}
          {sentBack.isError && <ErrorState message="Notices sent back to you could not be loaded." onRetry={() => sentBack.refetch()} />}

          <nav aria-label="Filter by status" className="-mx-4 flex gap-2 overflow-x-auto px-4 lg:mx-0 lg:gap-1 lg:self-start lg:rounded-[10px] lg:bg-well lg:p-1">
            {tabs.map((tab) => {
              const active = tab.value === filter
              return (
                <Link
                  key={tab.label}
                  to={tab.value ? `?status=${tab.value}` : '.'}
                  aria-current={active ? 'true' : undefined}
                  className={cn(
                    'flex min-h-11 shrink-0 items-center rounded-full border px-4 text-sm lg:min-h-9 lg:rounded-[7px] lg:border-0 lg:px-[18px]',
                    active
                      ? 'border-ink bg-ink font-semibold text-paper hover:text-paper lg:bg-surface lg:text-ink lg:shadow-[0_1px_2px_rgba(27,24,20,0.08)] lg:hover:text-ink'
                      : 'border-line bg-surface text-ink-2 hover:text-ink lg:bg-transparent',
                  )}
                >
                  {tab.label}
                </Link>
              )
            })}
          </nav>

          {list.isPending ? (
            <ListSkeleton />
          ) : list.isError && items.length === 0 ? (
            <ErrorState message="Your announcements could not be loaded." onRetry={() => list.refetch()} />
          ) : items.length === 0 ? (
            <p className="rounded-xl border border-line bg-surface px-6 py-6 text-[15px] text-ink-2">{tabs.find((t) => t.value === filter)?.empty}</p>
          ) : (
            <>
              <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5">
                {items.map((item) => (
                  <li key={item.id} className="[&+li]:shadow-[0_-1px_0_var(--color-rail)] lg:[&+li]:shadow-none">
                    <Row item={item} codes={codes} />
                  </li>
                ))}
              </ul>
              {list.hasNextPage && (
                <button
                  type="button"
                  onClick={() => list.fetchNextPage()}
                  disabled={list.isFetchingNextPage}
                  className="min-h-11 self-start px-1 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3"
                >
                  {list.isFetchingNextPage ? 'Loading…' : 'Show older'}
                </button>
              )}
              {list.isFetchNextPageError && <ErrorState message="Older announcements could not be loaded." onRetry={() => list.fetchNextPage()} />}
            </>
          )}
        </>
      )}
    </div>
  )
}

function Row({ item, codes }: { item: Authored; codes: Map<string, string> }) {
  const status = standing(item)
  return (
    <Link
      to={hrefFor(item)}
      className="flex flex-col gap-1.5 px-4 py-3 text-ink hover:text-ink lg:grid lg:min-h-[60px] lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center lg:gap-5 lg:rounded-lg lg:py-2.5 lg:hover:bg-paper"
    >
      <span className="lg:order-2">
        <StatusTag text={status.tag} tone={status.tone} />
      </span>
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="text-base leading-snug font-semibold lg:text-[15px]">{item.title}</span>
        <span className="text-[13px] text-ink-3">
          {describe(toRules(item.audience), codes)} · {status.when}
        </span>
      </span>
    </Link>
  )
}

// SentBackCard is what needs the author: new notices and edits that were
// sent back, each with the note and one button to fix it.
function SentBackCard({ items, codes }: { items: Authored[]; codes: Map<string, string> }) {
  const [expanded, setExpanded] = useState(false)
  // A phone shows one, so the rest of the list stays near the top.
  const shownAtFirst = useIsDesktop() ? 3 : 1
  const shown = expanded ? items : items.slice(0, shownAtFirst)
  const hidden = items.length - shown.length
  return (
    <section aria-labelledby="sent-back-h" className="flex flex-col rounded-xl border-[1.5px] border-danger bg-surface px-4 pt-3.5 pb-1 lg:px-6 lg:pt-4">
      <h2 id="sent-back-h" className="text-[13px] font-semibold text-danger">
        Sent back to you · {items.length}
      </h2>
      <ul>
        {shown.map((item) => {
          const isEdit = item.status === 'published'
          const note = isEdit ? item.edit?.review_note : item.review_note
          const who = (isEdit ? item.edit?.approver : item.approver) ?? 'Approver'
          return (
            <li
              key={item.id}
              className="flex flex-col gap-2.5 py-3.5 lg:grid lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end lg:gap-6 [&+li]:shadow-[0_-1px_0_#ead3da]"
            >
              <div className="flex flex-col gap-1">
                <span className="text-xs text-ink-3">
                  {isEdit ? 'Your edit to a live notice · readers still see the old version' : describe(toRules(item.audience), codes)}
                </span>
                <Link to={`/mine/${item.id}/edit`} className="font-serif text-lg font-medium text-ink hover:text-ink lg:text-[21px]">
                  {isEdit ? (item.edit?.title ?? item.title) : item.title}
                </Link>
                <p className="text-[15px] leading-normal text-ink-2">
                  <b className="font-semibold text-ink">{who}:</b> {note}
                </p>
              </div>
              <Link to={`/mine/${item.id}/edit`} className={buttonStyles.secondary}>
                {isEdit ? 'Edit again' : 'Edit and resubmit'}
              </Link>
            </li>
          )
        })}
      </ul>
      {hidden > 0 && (
        <button type="button" onClick={() => setExpanded(true)} className="min-h-11 self-start text-sm font-semibold text-rust hover:text-rust-deep">
          Show {hidden} more
        </button>
      )}
    </section>
  )
}

function ListSkeleton() {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4">
      <LoadingStatus label="Loading your announcements" />
      {Array.from({ length: 4 }, (_, i) => (
        <div key={i} className="flex flex-col gap-1.5">
          <Skeleton className="h-4 w-3/5" />
          <Skeleton className="h-3 w-2/5" />
        </div>
      ))}
    </div>
  )
}

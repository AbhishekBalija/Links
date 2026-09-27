import { ChevronLeft, Download } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Avatar } from '../../../app/shell/Avatar'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { useAnswerList, useEvent, useExportAnswers } from '../api'
import { dayMonth } from '../standing'
import { answerLabels, type Answer } from '../types'

const tabs: Answer[] = ['going', 'interested', 'not_going']

// EventPeople is everyone who answered an event, one answer at a time, for
// its organisers. The server decides who may see it.
export default function EventPeople() {
  const { id = '' } = useParams()
  const list = useAnswerList(id)
  const event = useEvent(id)
  const exportAnswers = useExportAnswers(id, event.data?.title ?? 'event')
  const [tab, setTab] = useState<Answer>('going')
  const counts = list.data?.counts
  const shown = (list.data?.people ?? []).filter((person) => person.status === tab)
  const organiser = list.data !== undefined && (list.data.people.length > 0 || (counts && counts.going + counts.interested + counts.not_going === 0))

  return (
    <div className="flex max-w-[720px] flex-col gap-4 lg:gap-5">
      <header className="flex items-center justify-between gap-3">
        <Link to={`/events/${id}`} className="-ml-1.5 inline-flex min-h-11 items-center gap-1 rounded-lg px-1.5 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-5 lg:size-4" />
          Event
        </Link>
        {organiser && (
          <button type="button" onClick={() => exportAnswers.mutate()} disabled={exportAnswers.isPending} className={buttonStyles.secondary}>
            <Download aria-hidden="true" className="size-4" />
            {exportAnswers.isPending ? 'Preparing…' : 'CSV'}
          </button>
        )}
      </header>
      <div className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium lg:text-[38px] lg:leading-tight">Who's coming</h1>
        {event.data && <p className="text-[15px] text-ink-2">{event.data.title}</p>}
      </div>
      {exportAnswers.isError && (
        <p role="alert" className="rounded-lg bg-danger-soft px-4 py-3 text-sm font-semibold text-danger-ink">
          {exportAnswers.error instanceof ApiRequestError ? exportAnswers.error.message : "The CSV couldn't be downloaded. Try again."}
        </p>
      )}

      {list.isPending ? (
        <div className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4">
          <LoadingStatus label="Loading the answers" />
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-9 w-full" />
          ))}
        </div>
      ) : list.isError ? (
        list.error instanceof ApiRequestError && list.error.status === 404 ? (
          <EmptyState title="This event isn't available">It may not be for you, or the link is out of date.</EmptyState>
        ) : (
          <ErrorState message="The answers could not be loaded." onRetry={() => list.refetch()} />
        )
      ) : !organiser ? (
        <EmptyState title="Only the organisers see who answered">The event page shows how many are going.</EmptyState>
      ) : (
        <>
          <div role="group" aria-label="Answer" className="flex gap-1 self-stretch rounded-[10px] bg-well p-1 lg:self-start">
            {tabs.map((value) => (
              <button
                key={value}
                type="button"
                aria-pressed={value === tab}
                onClick={() => setTab(value)}
                className={cn(
                  'flex min-h-9 flex-1 items-center justify-center gap-1.5 rounded-[7px] px-3 text-sm lg:flex-none lg:px-4',
                  value === tab ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2 hover:text-ink',
                )}
              >
                {answerLabels[value]}
                <span className="font-mono text-xs">{counts?.[value] ?? 0}</span>
              </button>
            ))}
          </div>
          {shown.length === 0 && !list.hasNextPage ? (
            <p className="rounded-xl border border-line bg-surface px-5 py-5 text-[15px] text-ink-2">Nobody has said {answerLabels[tab]} yet.</p>
          ) : (
            <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface">
              {shown.map((person) => (
                <li key={person.user_id} className="[&+li]:shadow-[0_-1px_0_var(--color-rail)]">
                  <div className="flex min-h-14 items-center gap-3 px-4 py-2">
                    <Avatar name={person.full_name} />
                    <span className="flex-1 text-[15px] font-semibold">{person.full_name}</span>
                    <span className="font-mono text-xs text-ink-3">{dayMonth(person.responded_at)}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
          {list.hasNextPage && (
            <button
              type="button"
              onClick={() => list.fetchNextPage()}
              disabled={list.isFetchingNextPage}
              className="min-h-11 self-start px-1 text-sm font-semibold text-rust hover:text-rust-deep disabled:text-ink-3"
            >
              {list.isFetchingNextPage ? 'Loading…' : 'Show more'}
            </button>
          )}
          <p className="px-1 text-[13px] text-ink-3 lg:px-0">Earliest answer first. The CSV includes emails and USNs, so each export is logged.</p>
        </>
      )}
    </div>
  )
}

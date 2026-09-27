import { ChevronRight, Download } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { useAnswers, useExportAnswers } from '../api'
import { publishedLine } from '../organiser'
import { dayAndDate, dayMonth } from '../standing'
import { answerLabels, typeLabel, type AnswerSummary, type CampusEvent } from '../types'
import { CancelDialog } from './CancelDialog'
import { DateTile } from './DateTile'
import { EventFacts, Fact } from './EventFacts'
import { StandingTag } from './PostTags'

// OrganiserView is an event as the people running it see it: where it
// stands, what they can still change, and who is coming.
export function OrganiserView({ event }: { event: CampusEvent }) {
  const answers = useAnswers(event.id)
  const over = new Date(event.ends_at) <= new Date()
  const tag =
    event.status === 'cancelled'
      ? ({ tag: 'Cancelled', tone: 'danger' } as const)
      : over
        ? ({ tag: 'Over', tone: 'neutral' } as const)
        : ({ tag: 'Live', tone: 'live' } as const)

  return (
    <div className="grid max-w-[1140px] items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-6">
      <div className="flex flex-col gap-4">
        {event.status === 'cancelled' && (
          <div role="status" className="flex flex-col gap-1 rounded-xl bg-danger-soft px-4 py-3.5 text-danger-ink">
            <span className="text-[15px] font-bold">Cancelled{event.cancelled_at && ` on ${dayAndDate(event.cancelled_at)}`}</span>
            {event.cancel_reason && <span className="text-sm leading-[1.45]">“{event.cancel_reason}”</span>}
          </div>
        )}
        <article className="flex flex-col gap-4 lg:gap-[18px] lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
          <div className="flex flex-wrap items-center gap-2">
            <StandingTag standing={tag} />
            <span className="text-xs text-ink-3">{publishedLine(event)}</span>
          </div>
          <div className="flex items-start gap-3.5 lg:gap-[18px]">
            <DateTile iso={event.starts_at} size="lg" />
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-ink-3">{typeLabel(event.event_type)}</span>
              <h1 className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[34px] lg:leading-[1.12] lg:tracking-[-0.5px]">{event.title}</h1>
            </div>
          </div>
          <EventFacts event={event}>
            <Fact label="Organised by">
              {event.proposer_name}
              {event.faculty_mentor && ` · mentor ${event.faculty_mentor.full_name}`}
            </Fact>
          </EventFacts>
        </article>
        <OrganiserBar event={event} over={over} going={answers.data?.counts.going ?? 0} />
      </div>
      <WhoIsComing event={event} answers={answers.data} pending={answers.isPending} failed={answers.isError} onRetry={() => answers.refetch()} />
    </div>
  )
}

// OrganiserBar holds the two things organisers can do once an event is out:
// change its logistics, or cancel it.
function OrganiserBar({ event, over, going }: { event: CampusEvent; over: boolean; going: number }) {
  const [cancelling, setCancelling] = useState(false)
  if (event.status === 'cancelled') {
    return <ActionBar sentence="The people invited see it's cancelled, with your reason, when they open it." />
  }
  if (over) return <ActionBar sentence="It's over. The answers stay here for your records." />
  return (
    <>
      <ActionBar
        start={
          <button type="button" onClick={() => setCancelling(true)} className={buttonStyles.secondary}>
            Cancel event
          </button>
        }
        sentence="Place, time, seats and description can still change."
      >
        <Link to={`/events/${event.id}/edit`} className={buttonStyles.primary}>
          Edit details
        </Link>
      </ActionBar>
      <CancelDialog
        open={cancelling}
        eventId={event.id}
        title="Cancel this event?"
        reasonLabel="Reason (everyone invited sees this)"
        keepLabel="Keep event"
        confirmLabel="Cancel event"
        onClose={() => setCancelling(false)}
        onCancelled={() => setCancelling(false)}
      >
        It leaves the Events list.{' '}
        {going > 0 ? `The ${going} ${going === 1 ? 'person' : 'people'} going see` : 'Anyone who opens it sees'} it's cancelled, with your reason. This can't be undone.
      </CancelDialog>
    </>
  )
}

const PREVIEW = 4

// WhoIsComing is the answer counts, the first few people and the export.
function WhoIsComing({ event, answers, pending, failed, onRetry }: {
  event: CampusEvent
  answers: AnswerSummary | undefined
  pending: boolean
  failed: boolean
  onRetry: () => void
}) {
  const exportAnswers = useExportAnswers(event.id, event.title)
  const counts = answers?.counts
  const total = counts ? counts.going + counts.interested + counts.not_going : 0
  const people = (answers?.people ?? []).slice(0, PREVIEW)
  const exportError =
    exportAnswers.error instanceof ApiRequestError ? exportAnswers.error.message : exportAnswers.error ? "The CSV couldn't be downloaded. Try again." : null

  return (
    <aside aria-labelledby="coming-h" className="flex flex-col gap-3.5 rounded-xl border border-line bg-surface p-4 lg:p-5">
      <h2 id="coming-h" className="font-serif text-[21px] font-medium">
        Who's coming
      </h2>
      {pending ? (
        <div className="flex flex-col gap-2.5">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      ) : failed || !counts ? (
        <div role="alert" className="flex flex-col items-start gap-2 text-sm text-danger-ink">
          The answers could not be loaded.
          <button type="button" onClick={onRetry} className="min-h-10 font-semibold text-rust hover:text-rust-deep">
            Try again
          </button>
        </div>
      ) : (
        <>
          <dl className="grid grid-cols-3 gap-2">
            {(['going', 'interested', 'not_going'] as const).map((key) => (
              <div key={key} className="flex flex-col-reverse rounded-[10px] bg-paper px-3 py-2.5">
                <dt className="mt-1 text-xs text-ink-2">{key === 'not_going' ? "can't go" : key}</dt>
                <dd className="font-serif text-[26px] leading-none font-medium">{counts[key]}</dd>
              </div>
            ))}
          </dl>
          {event.capacity !== null && (
            <p className="font-mono text-xs text-ink-3">
              {Math.max(event.capacity - counts.going, 0)} of {event.capacity} seats left
            </p>
          )}
          {total === 0 ? (
            <p className="text-sm text-ink-2">No answers yet. Everyone invited sees it in their Events list.</p>
          ) : (
            <ul className="flex flex-col">
              {people.map((person) => (
                <li key={person.user_id} className="flex min-h-11 items-center gap-3 text-sm [&+li]:shadow-[0_-1px_0_var(--color-rail)]">
                  <span className="flex-1 font-semibold">{person.full_name}</span>
                  <span className="text-xs text-ink-3">{answerLabels[person.status]}</span>
                  <span className="w-12 text-right font-mono text-xs text-ink-3">{dayMonth(person.responded_at)}</span>
                </li>
              ))}
            </ul>
          )}
          <div className="flex items-center justify-between gap-3">
            {total > 0 ? (
              <Link to={`/events/${event.id}/people`} className="inline-flex min-h-11 items-center gap-1 text-sm font-semibold">
                All {total} {total === 1 ? 'answer' : 'answers'}
                <ChevronRight aria-hidden="true" className="size-4" />
              </Link>
            ) : (
              <span />
            )}
            <button type="button" onClick={() => exportAnswers.mutate()} disabled={exportAnswers.isPending || total === 0} className={buttonStyles.secondary}>
              <Download aria-hidden="true" className="size-4" />
              {exportAnswers.isPending ? 'Preparing…' : 'Export CSV'}
            </button>
          </div>
          {exportError && (
            <p role="alert" className="text-[13px] font-semibold text-danger">
              {exportError}
            </p>
          )}
          <p className="text-xs text-ink-3">The CSV includes emails and USNs, so each export is logged.</p>
        </>
      )}
    </aside>
  )
}

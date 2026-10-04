import { Check, ChevronLeft } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { audienceLabel } from '../../notices/format'
import { useAnswer, useAnswers, useEvent } from '../api'
import { useAuthStore } from '../../auth/store'
import { useDashboard } from '../../home/api'
import { AnswerControl } from '../components/AnswerControl'
import { OrganiserView } from '../components/OrganiserView'
import { DateTile } from '../components/DateTile'
import { Tag } from '../components/Tags'
import { answersClosed, isFull, startTime, whenLine } from '../format'
import { isOrganiser, organiserName } from '../organiser'
import { typeLabel, type AnswerSummary, type CampusEvent } from '../types'

// EventDetail is one Event: what, when and where, then the answer bar, the
// one place to act. On phones the bar sits at the bottom of the screen.
export default function EventDetail() {
  const { id = '' } = useParams()
  const event = useEvent(id)

  return (
    <div className="flex flex-col gap-3 pb-40 lg:gap-5 lg:pb-0">
      <BackLink />
      {event.isPending ? (
        <>
          <LoadingStatus label="Loading the event" />
          <DetailSkeleton />
        </>
      ) : event.isError ? (
        event.error instanceof ApiRequestError && event.error.status === 404 ? (
          <div className="max-w-[640px]">
            <EmptyState title="This event isn't available">
              <p>It may not be for you, or the link is out of date.</p>
              <Link to="/events" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
                See your events
              </Link>
            </EmptyState>
          </div>
        ) : (
          <ErrorState message="This event could not be loaded." onRetry={() => event.refetch()} />
        )
      ) : (
        <EventPage event={event.data} />
      )}
    </div>
  )
}

function BackLink() {
  const state = useLocation().state as { back?: string } | null
  return (
    <Link to={state?.back ?? '/events'} className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
      <ChevronLeft aria-hidden="true" className="size-5" />
      Events
    </Link>
  )
}

// EventPage shows organisers their tools, and everyone else the event with
// its answer bar.
function EventPage({ event }: { event: CampusEvent }) {
  const user = useAuthStore((s) => s.user)
  // Only an HOD needs their Department to tell if the event is theirs.
  const isHOD = Boolean(user?.roles.includes('hod'))
  const dashboard = useDashboard(isHOD)
  if (isHOD && dashboard.isPending) return <DetailSkeleton />
  const live = event.status === 'published' || event.status === 'cancelled'
  if (user && live && isOrganiser(event, user, dashboard.data?.user.department ?? null)) {
    return <OrganiserView event={event} />
  }
  return <EventView event={event} />
}

function EventView({ event }: { event: CampusEvent }) {
  const answers = useAnswers(event.id, event.status === 'published' || event.status === 'cancelled')
  const when = whenLine(event.starts_at, event.ends_at)
  const audience = audienceLabel(
    (event.audience ?? []).map((rule) => ({
      department_id: rule.department_id ?? null,
      department_code: rule.department_code ?? null,
      batch_year: rule.batch_year ?? null,
      role: rule.role ?? null,
    })),
  )

  return (
    <div className="flex max-w-[920px] flex-col gap-3.5 lg:gap-4">
      {event.status === 'cancelled' && (
        <div role="status" className="flex flex-col gap-1 rounded-xl bg-danger-soft px-4 py-3.5 text-danger-ink">
          <span className="text-[15px] font-bold">
            Cancelled
            {event.cancelled_at && ` on ${new Date(event.cancelled_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })}`}
          </span>
          {event.cancel_reason && <span className="text-sm leading-[1.45]">“{event.cancel_reason}”</span>}
        </div>
      )}

      <article className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-10 lg:pt-8 lg:pb-9">
        <header className="flex items-start gap-3.5 lg:gap-5">
          <DateTile iso={event.starts_at} size="lg" />
          <div className="flex flex-col gap-1.5 lg:gap-2">
            <span className="flex gap-1.5">
              <Tag>{typeLabel(event.event_type)}</Tag>
              {event.department && <Tag tone="department">{event.department.code}</Tag>}
            </span>
            <h1 className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-[1.1] lg:tracking-[-0.6px]">
              {event.title}
            </h1>
          </div>
        </header>

        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 rounded-xl border border-line bg-surface px-4 py-3.5 text-sm lg:gap-x-[18px] lg:gap-y-2.5 lg:border-0 lg:bg-transparent lg:p-0 lg:text-[15px]">
          <Fact label="When">
            {when.date && <span className="font-semibold">{when.date} </span>}
            <span className="font-mono text-[13px] lg:text-sm">{when.time}</span>
          </Fact>
          <Fact label="Where">{event.location}</Fact>
          <Fact label="For">{audience}</Fact>
          <Fact label="Organised by">
            {organiserName(event)}
            {event.faculty_mentor && ` · mentor ${event.faculty_mentor.full_name}`}
          </Fact>
        </dl>

        {event.description && (
          <div className="flex max-w-[640px] flex-col gap-3 font-serif text-[17px] leading-[1.55] text-prose lg:text-lg lg:leading-[1.6]">
            {event.description.split(/\n\s*\n/).map((paragraph, i) => (
              <p key={i} className="whitespace-pre-line">
                {paragraph}
              </p>
            ))}
          </div>
        )}
      </article>

      {event.status === 'published' &&
        (answers.isPending ? (
          <Skeleton className="h-[72px] w-full rounded-xl" />
        ) : answers.data ? (
          <AnswerBar event={event} summary={answers.data} />
        ) : null)}
    </div>
  )
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-ink-3">{label}</dt>
      <dd>{children}</dd>
    </>
  )
}

const answerWords = { going: 'Going', interested: 'Interested', not_going: "Can't go" }

// AnswerBar says where the reader stands and lets them change it. Answers
// close once the event starts.
function AnswerBar({ event, summary }: { event: CampusEvent; summary: AnswerSummary }) {
  const answer = useAnswer(event.id)
  const { counts, my_status: mine } = summary
  const full = isFull(event.capacity, counts.going)
  const closed = answersClosed(event)
  const failed = answer.error instanceof ApiRequestError ? answer.error.message : answer.error ? 'Your answer could not be saved. Try again.' : null

  let text: ReactNode
  if (closed) {
    text = (
      <>
        Started at {startTime(event.starts_at)}, so answers are closed.
        {mine && (
          <>
            {' '}
            You said <b className="font-semibold text-ink">{answerWords[mine]}</b>.
          </>
        )}
      </>
    )
  } else if (mine === 'going') {
    text = (
      <span className="inline-flex items-center gap-2 font-semibold text-success-ink">
        <Check aria-hidden="true" className="size-4" strokeWidth={2.4} />
        You're going. Choose another answer to change it.
      </span>
    )
  } else if (full) {
    text = (
      <>
        <b className="font-semibold text-ink">Full:</b> all {event.capacity} seats are taken. You can still say you're interested.
      </>
    )
  } else {
    text = (
      <>
        <b className="font-semibold text-ink">Are you going?</b> {counts.going} going
        {counts.interested > 0 && `, ${counts.interested} interested`}.{event.capacity === null ? ' No seat limit.' : ` ${event.capacity - counts.going} seats left.`}
      </>
    )
  }

  return (
    <footer className="fixed inset-x-0 bottom-0 z-40 flex flex-col gap-2.5 bg-surface px-4 pt-3 pb-[max(env(safe-area-inset-bottom),20px)] shadow-[0_-1px_0_var(--color-line)] lg:static lg:flex-row lg:items-center lg:justify-between lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:p-3.5 lg:pl-4 lg:shadow-none">
      <div className="flex flex-col gap-1">
        <p className="text-[13px] text-ink-2 lg:text-[15px]">{text}</p>
        {failed && (
          <p role="alert" className="text-[13px] font-semibold text-danger">
            {failed}
          </p>
        )}
      </div>
      {!closed && (
        <div className="lg:shrink-0">
          <div className="lg:hidden">
            <AnswerControl value={mine} full={full} busy={answer.isPending} onAnswer={(value) => answer.mutate(value)} grow />
          </div>
          <div className="hidden lg:block">
            <AnswerControl value={mine} full={full} busy={answer.isPending} onAnswer={(value) => answer.mutate(value)} />
          </div>
        </div>
      )}
    </footer>
  )
}

function DetailSkeleton() {
  return (
    <div className="flex max-w-[920px] flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-10">
      <div className="flex gap-4">
        <Skeleton className="h-20 w-14 rounded-[10px] lg:w-[72px]" />
        <div className="flex flex-1 flex-col gap-2.5 pt-1">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-7 w-4/5" />
        </div>
      </div>
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-4 w-1/2" />
      <Skeleton className="h-4 w-3/5" />
    </div>
  )
}

import { ChevronLeft } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { useDeleteDraft, useEvent } from '../api'
import { CancelDialog } from '../components/CancelDialog'
import { DateTile } from '../components/DateTile'
import { EventFacts } from '../components/EventFacts'
import { PostTag, StandingTag } from '../components/PostTags'
import { proposalHistory } from '../history'
import { dayAndDate, dayMonth, latestReview, proposalStanding, reviewerAt } from '../standing'
import { typeLabel, type CampusEvent } from '../types'

// MyEvent is one of the author's own proposals: what it says, where it
// stands, and the one bar of actions its status allows.
export default function MyEvent() {
  const { id = '' } = useParams()
  const event = useEvent(id)

  return (
    <div className="group/page flex flex-col gap-4 pb-44 lg:gap-5 lg:pb-0">
      <Link to="/mine" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5 lg:size-4" />
        My posts
      </Link>
      {event.isPending ? (
        <ViewSkeleton />
      ) : event.isError ? (
        event.error instanceof ApiRequestError && event.error.status === 404 ? (
          <EmptyState title="This proposal isn't available">It may have been deleted, or it isn't yours.</EmptyState>
        ) : (
          <ErrorState message="This proposal could not be loaded." onRetry={() => event.refetch()} />
        )
      ) : (
        <View event={event.data} />
      )}
    </div>
  )
}

function View({ event }: { event: CampusEvent }) {
  const standing = proposalStanding(event)
  const history = proposalHistory(event)

  return (
    <div className="grid max-w-[1140px] items-start gap-4 lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-6">
      <div className="flex flex-col gap-4">
        <article className="flex flex-col gap-4 lg:gap-[18px] lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
          <div className="flex flex-wrap items-center gap-2">
            <PostTag kind="event" />
            <StandingTag standing={standing} />
            <span className="text-xs text-ink-3">{typeLabel(event.event_type)}</span>
          </div>
          <div className="flex items-start gap-3.5 lg:gap-[18px]">
            <DateTile iso={event.starts_at} size="lg" />
            <h1 className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[34px] lg:leading-[1.12] lg:tracking-[-0.5px]">{event.title}</h1>
          </div>
          <EventFacts event={event} />
        </article>
        <Bar event={event} />
      </div>

      <aside aria-labelledby="history-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 lg:p-5">
        <h2 id="history-h" className="font-serif text-[21px] font-medium">
          History
        </h2>
        <ol className="flex flex-col gap-3.5">
          {history.map((entry, i) => (
            <li key={i} className="flex gap-3.5">
              <span className="w-16 shrink-0 pt-0.5 font-mono text-xs text-ink-3">
                {dayMonth(entry.at)}
              </span>
              <span className="flex min-w-0 flex-col gap-1">
                <span className="text-sm">
                  <b className="font-semibold">{entry.who}</b>{' '}
                  <span className={cn(entry.tone === 'warning' && 'text-warning-ink', entry.tone === 'danger' && 'text-danger-ink')}>{entry.what}</span>
                </span>
                {entry.note && <span className="rounded-lg bg-paper px-3 py-2.5 text-sm leading-[1.45] text-prose">{entry.note}</span>}
              </span>
            </li>
          ))}
        </ol>
      </aside>
    </div>
  )
}

// Bar holds what the author can do at each status, and says why when there
// is nothing to do.
function Bar({ event }: { event: CampusEvent }) {
  const navigate = useNavigate()
  const deleteDraft = useDeleteDraft()
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [cancelling, setCancelling] = useState(false)
  const [failed, setFailed] = useState('')
  const review = latestReview(event)
  const edit = (label: string) => (
    <Link to={`/mine/events/${event.id}/edit`} className={buttonStyles.primary}>
      {label}
    </Link>
  )
  const cancelButton = (
    <button type="button" onClick={() => setCancelling(true)} className={buttonStyles.secondary}>
      Cancel proposal
    </button>
  )
  const cancelDialog = (
    <CancelDialog
      open={cancelling}
      eventId={event.id}
      title="Cancel this proposal?"
      reasonLabel="Reason (the reviewers see this)"
      keepLabel="Keep proposal"
      confirmLabel="Cancel proposal"
      onClose={() => setCancelling(false)}
      onCancelled={() => {
        setCancelling(false)
        navigate('/mine?status=ended', { replace: true })
      }}
    >
      It stops here, and the reviewers no longer see it. This can't be undone.
    </CancelDialog>
  )

  if (confirmingDelete) {
    return (
      <ActionBar
        tone="danger"
        labelledBy="delete-h"
        onEscape={() => setConfirmingDelete(false)}
        sentence={
          <span id="delete-h">
            <b className="font-semibold">Delete this draft?</b> It can't be brought back.
            {failed && <span className="block font-semibold">{failed}</span>}
          </span>
        }
      >
        <button type="button" onClick={() => setConfirmingDelete(false)} className={buttonStyles.secondary + ' flex-1 bg-surface lg:flex-none'}>
          Keep it
        </button>
        <button
          type="button"
          disabled={deleteDraft.isPending}
          onClick={async () => {
            setFailed('')
            try {
              await deleteDraft.mutateAsync(event.id)
              navigate('/mine?status=draft', { replace: true })
            } catch {
              setFailed("It couldn't be deleted. Try again.")
            }
          }}
          className={buttonStyles.danger}
        >
          {deleteDraft.isPending ? 'Deleting…' : 'Delete draft'}
        </button>
      </ActionBar>
    )
  }

  switch (event.status) {
    case 'draft':
      return (
        <ActionBar
          start={
            <button type="button" onClick={() => setConfirmingDelete(true)} className={buttonStyles.secondary}>
              Delete draft
            </button>
          }
          sentence="Only you can see it."
        >
          {edit('Edit')}
        </ActionBar>
      )
    case 'submitted':
    case 'hod_approved': {
      const hodStage = event.status === 'submitted'
      const since = dayAndDate((hodStage ? event.submitted_at : review?.decided_at) ?? event.updated_at)
      return (
        <>
          <ActionBar
            start={cancelButton}
            sentence={
              <>
                With <b className="font-semibold text-ink">{reviewerAt(hodStage ? 'hod' : 'final', event)}</b> since {since}.
                {hodStage && ' After them, the principal.'} You can cancel it meanwhile.
              </>
            }
          />
          {cancelDialog}
        </>
      )
    }
    case 'hod_changes_requested':
    case 'final_changes_requested':
      return (
        <>
          <ActionBar start={cancelButton} sentence={<>Fix what they asked; it goes back to <b className="font-semibold text-ink">them</b>.</>}>
            {edit('Edit and resubmit')}
          </ActionBar>
          {cancelDialog}
        </>
      )
    case 'hod_rejected':
    case 'final_rejected': {
      const who = reviewerAt(event.status === 'hod_rejected' ? 'hod' : 'final', event)
      return (
        <ActionBar sentence={`${who.charAt(0).toUpperCase()}${who.slice(1)} said no, so it can't be sent again. Their note is in the history.`}>
          <Link to="/mine/events/new" className={buttonStyles.primary}>
            Propose a new one
          </Link>
        </ActionBar>
      )
    }
    case 'published':
      if (new Date(event.ends_at) <= new Date()) {
        return <ActionBar sentence="It's over." />
      }
      return (
        <ActionBar sentence="Published. Its page shows who's coming.">
          <Link to={`/events/${event.id}`} state={{ back: `/mine/events/${event.id}` }} className={buttonStyles.primary}>
            Open event page
          </Link>
        </ActionBar>
      )
    case 'cancelled':
      return (
        <ActionBar
          sentence={
            <>
              Cancelled{event.cancelled_at && ` on ${dayAndDate(event.cancelled_at)}`}.{' '}
              {event.published_at ? 'The people invited see your reason.' : 'The reviewers no longer see it.'}
            </>
          }
        />
      )
  }
}

function ViewSkeleton() {
  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-6">
      <LoadingStatus label="Loading the proposal" />
      <div className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-9">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-9 w-4/5" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </div>
      <Skeleton className="hidden h-48 rounded-xl lg:block" />
    </div>
  )
}

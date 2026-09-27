import { ChevronLeft } from 'lucide-react'
import { useState } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { ComposeSkeleton } from '../../announcements/components/compose/ComposeSkeleton'
import { LeaveDialog } from '../../announcements/components/LeaveDialog'
import { useAnswers, useEvent, useUpdateEvent } from '../api'
import { SeatLimit, WhenAndWhere } from '../components/FormFields'
import { checkLogistics, logisticsChanges } from '../logistics'
import { fromEvent } from '../proposal'
import { serverErrors, useEventForm } from '../useEventForm'
import type { CampusEvent } from '../types'

// EventEdit changes a published event's logistics: when, where, seats and
// description. Everything else needs the event cancelled and proposed again.
export default function EventEdit() {
  const { id = '' } = useParams()
  const event = useEvent(id)
  const answers = useAnswers(id)

  if (event.isPending || answers.isPending) return <ComposeSkeleton />
  if (event.error instanceof ApiRequestError && event.error.status === 404) {
    return <EmptyState title="This event isn't available">It may not be yours to edit, or the link is out of date.</EmptyState>
  }
  if (event.isError || answers.isError) {
    return (
      <ErrorState
        message="The event could not be loaded."
        onRetry={() => {
          event.refetch()
          answers.refetch()
        }}
      />
    )
  }
  // Only a published event that isn't over takes these edits.
  if (event.data.status !== 'published' || new Date(event.data.ends_at) <= new Date()) {
    return <Navigate to={`/events/${id}`} replace />
  }
  return <EditForm event={event.data} going={answers.data.counts.going} />
}

function EditForm({ event, going }: { event: CampusEvent; going: number }) {
  const navigate = useNavigate()
  const update = useUpdateEvent()
  const { form, set, errors, setErrors, blocker, allowLeaving } = useEventForm(() => fromEvent(event))
  const [failure, setFailure] = useState('')

  const back = () => {
    allowLeaving()
    navigate(`/events/${event.id}`, { replace: true })
  }

  async function save() {
    setFailure('')
    const found = checkLogistics(form, event, new Date(), going)
    setErrors(found)
    if (Object.keys(found).length > 0) return
    const changes = logisticsChanges(form, event)
    if (Object.keys(changes).length === 0) return back()
    try {
      await update.mutateAsync({ id: event.id, input: changes })
      back()
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400 && err.details) {
        setErrors(serverErrors(err.details))
      } else if (err instanceof ApiRequestError && err.status === 409) {
        setFailure("It can't be changed any more: it may have started or been cancelled. Reload to see it.")
      } else {
        setFailure("It couldn't be saved. Check your connection and try again; your changes are still here.")
      }
    }
  }

  return (
    <div className="group/page flex max-w-[760px] flex-col gap-5 pb-44 lg:pb-0">
      <header className="flex flex-col gap-1">
        <Link to={`/events/${event.id}`} className="-ml-1.5 inline-flex min-h-10 items-center gap-1.5 self-start rounded-lg px-1.5 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-4" />
          Event
        </Link>
        <h1 className="px-1 font-serif text-[26px] font-medium lg:px-0 lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Edit details</h1>
        <p className="px-1 text-[15px] leading-normal text-ink-2 lg:px-0">
          <b className="font-semibold text-ink">{event.title}</b> is published, so only these can change. Changes show straight away; nobody approves them again.
        </p>
      </header>

      <section aria-label="Details" className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-[30px] lg:pt-[26px] lg:pb-[30px]">
        <WhenAndWhere
          form={form}
          errors={errors}
          set={set}
          between={
            <div className="flex flex-col gap-2">
              <SeatLimit
                form={form}
                errors={errors}
                set={set}
                heading={<span className="text-[13px] font-semibold text-ink-2">Seats</span>}
                hint={
                  !errors.capacity && <span className="text-[13px] text-ink-3">
                    {going === 1 ? '1 is going.' : `${going} are going.`} A limit can't be lower than that.
                  </span>
                }
              />
            </div>
          }
        />
        <p className="rounded-[10px] bg-paper px-4 py-3 text-[13px] text-ink-2">To change the title, type or who's invited, cancel it and propose it again.</p>
      </section>

      {failure && (
        <p role="alert" className="rounded-lg bg-danger-soft px-4 py-3 text-sm font-semibold text-danger-ink">
          {failure}
        </p>
      )}

      <ActionBar sentence="Everyone invited sees the change when they open it.">
        <button type="button" onClick={back} className={buttonStyles.secondary}>
          Discard
        </button>
        <button type="button" onClick={save} disabled={update.isPending} className={buttonStyles.primary}>
          {update.isPending ? 'Saving…' : 'Save changes'}
        </button>
      </ActionBar>

      <LeaveDialog
        what="event"
        open={blocker.state === 'blocked'}
        canSaveDraft={false}
        saving={false}
        onSaveDraft={() => {}}
        onKeepEditing={() => blocker.reset?.()}
        onDiscard={() => {
          allowLeaving()
          blocker.proceed?.()
        }}
      />
    </div>
  )
}

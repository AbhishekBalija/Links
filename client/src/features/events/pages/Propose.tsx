import { ChevronLeft } from 'lucide-react'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, Navigate, useBlocker, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { useAuthStore } from '../../auth/store'
import { useDashboard } from '../../home/api'
import { useDepartments } from '../../announcements/api'
import { presetsFor } from '../../announcements/audience'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { AudiencePicker } from '../../announcements/components/AudiencePicker'
import { ComposeSkeleton } from '../../announcements/components/compose/ComposeSkeleton'
import { Segmented } from '../../announcements/components/compose/Fields'
import { errorRing, inputClass } from '../../announcements/components/compose/styles'
import { LeaveDialog } from '../../announcements/components/LeaveDialog'
import { DateTime, FieldError, Labelled } from '../components/FormFields'
import type { Department } from '../../announcements/types'
import { useDepartmentOverview } from '../../people/api'
import { useCreateEvent, useEvent, useSubmitEvent, useUpdateEvent } from '../api'
import { checkProposal, durationLabel, emptyProposal, fromEvent, toInput, type ProposalErrors, type ProposalForm } from '../proposal'
import { routeFor } from '../route'
import { latestReview, reviewerAt } from '../standing'
import { eventTypes, type CampusEvent, type EventStatus } from '../types'

// Statuses the proposer can still edit; anything else opens its own page.
const editable: EventStatus[] = ['draft', 'hod_changes_requested', 'final_changes_requested']

// Propose writes a new event proposal, or edits a draft or one sent back.
export default function Propose() {
  const { id } = useParams()
  const existing = useEvent(id ?? '', Boolean(id))
  const dashboard = useDashboard()
  const departments = useDepartments()

  if ((id && existing.isPending) || dashboard.isPending || departments.isPending) return <ComposeSkeleton />
  if (id && existing.error instanceof ApiRequestError && existing.error.status === 404) {
    return (
      <EmptyState title="This proposal isn't available">
        It may have been deleted, or it isn't yours.
      </EmptyState>
    )
  }
  if ((id && existing.isError) || dashboard.isError || departments.isError) {
    return (
      <ErrorState
        message="The proposal form could not be loaded."
        onRetry={() => {
          existing.refetch()
          dashboard.refetch()
          departments.refetch()
        }}
      />
    )
  }
  const item = existing.data
  if (item && !editable.includes(item.status)) return <Navigate to={`/mine/events/${item.id}`} replace />
  return (
    <ProposeForm
      key={item?.id ?? 'new'}
      item={item}
      department={item?.department ?? dashboard.data?.user.department ?? null}
      departments={departments.data ?? []}
    />
  )
}

// Maps the server's field names onto the form's.
const serverFields: Record<string, keyof ProposalErrors> = {
  title: 'title',
  description: 'description',
  event_type: 'event_type',
  location: 'location',
  starts_at: 'starts',
  ends_at: 'ends',
  capacity: 'capacity',
  audience: 'audience',
}

function ProposeForm({ item, department, departments }: {
  item: CampusEvent | undefined
  department: { id: string; code: string } | null
  departments: Department[]
}) {
  const navigate = useNavigate()
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  const overview = useDepartmentOverview(department?.code)
  const privileged = roles.includes('principal') || roles.includes('admin')
  // The placement office proposes training events only.
  const trainingOnly = !privileged && roles.includes('placement_officer') && !roles.some((r) => ['hod', 'faculty', 'student_coordinator'].includes(r))
  const types = trainingOnly ? eventTypes.filter((t) => t.value === 'training') : eventTypes

  const [initial] = useState<ProposalForm>(() =>
    item ? fromEvent(item) : emptyProposal(department ? presetsFor(department)[2].audience : [], trainingOnly ? 'training' : null),
  )
  const [form, setForm] = useState(initial)
  const [errors, setErrors] = useState<ProposalErrors>({})
  const [failure, setFailure] = useState('')
  const leaving = useRef(false)
  const titleRef = useRef<HTMLInputElement>(null)

  const create = useCreateEvent()
  const update = useUpdateEvent()
  const submit = useSubmitEvent()
  const saving = create.isPending || update.isPending || submit.isPending

  const dirty = JSON.stringify(form) !== JSON.stringify(initial)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  function set<K extends keyof ProposalForm>(key: K, value: ProposalForm[K]) {
    setForm((f) => {
      const next = { ...f, [key]: value }
      // A new start date carries the end date along while they matched.
      if (key === 'startDate' && (f.endDate === '' || f.endDate === f.startDate)) next.endDate = value as string
      return next
    })
    const field: keyof ProposalErrors | undefined =
      key === 'startDate' || key === 'startTime' ? 'starts' : key === 'endDate' || key === 'endTime' ? 'ends' : key === 'limitSeats' ? 'capacity' : (key as keyof ProposalErrors)
    setErrors((e) => ({ ...e, [field]: undefined }))
  }

  const resubmitting = item?.status === 'hod_changes_requested' || item?.status === 'final_changes_requested' ? item.status : undefined
  const route = routeFor({
    roles,
    eventType: form.event_type ?? 'other',
    department,
    hasHOD: Boolean(overview.data?.hod),
    resubmitting,
  })
  const routeKnown = !department || !overview.isPending

  // save creates or updates the proposal, then submits it unless it stays a
  // draft. What was typed stays in the form if anything fails.
  async function save(asDraft: boolean): Promise<string | null> {
    setFailure('')
    const found = checkProposal(form, new Date(), { submitting: !asDraft })
    setErrors(found)
    if (Object.keys(found).length > 0) {
      if (found.title) titleRef.current?.focus()
      return null
    }
    const input = toInput(form, department)
    try {
      let saved: CampusEvent
      if (!item) {
        saved = await create.mutateAsync({ ...input, draft: asDraft })
      } else {
        saved = await update.mutateAsync({ id: item.id, input })
        if (!asDraft) saved = await submit.mutateAsync(item.id)
      }
      return saved.id
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400 && err.details) {
        const mapped: ProposalErrors = {}
        for (const [field, message] of Object.entries(err.details)) {
          const key = serverFields[field]
          if (key && typeof message === 'string') mapped[key] = sentence(message)
        }
        setErrors(mapped)
        setFailure(Object.keys(mapped).length > 0 ? '' : 'This proposal needs a change the form cannot show. Check the details and try again.')
      } else if (err instanceof ApiRequestError && err.status === 409) {
        setFailure('This proposal changed somewhere else. Reload it to see where it stands.')
      } else if (err instanceof ApiRequestError && err.status === 403) {
        setFailure("You can't propose events any more. Ask an admin if that's a mistake.")
      } else {
        setFailure("It couldn't be saved. Check your connection and try again; what you typed is still here.")
      }
      return null
    }
  }

  async function handleSubmit() {
    const savedId = await save(false)
    if (!savedId) return
    leaving.current = true
    navigate(`/mine/events/${savedId}`, { replace: true })
  }

  async function handleDraft() {
    const savedId = await save(true)
    if (!savedId) return
    leaving.current = true
    if (blocker.state === 'blocked') blocker.proceed()
    else navigate('/mine?status=draft', { replace: true })
  }

  const errorCount = Object.values(errors).filter(Boolean).length
  const review = latestReview(item ?? {})
  const heading = !item ? 'Propose an event' : item.status === 'draft' ? 'Edit draft' : 'Edit and resubmit'
  const duration = durationLabel(form)

  return (
    <div className="group/page flex flex-col gap-5 pb-44 lg:pb-0">
      <header className="flex flex-col gap-1">
        <Link to={item ? `/mine/events/${item.id}` : '/mine'} className="-ml-1.5 inline-flex min-h-10 items-center gap-1.5 self-start rounded-lg px-1.5 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-4" />
          {item ? 'Back to the proposal' : 'My posts'}
        </Link>
        <h1 className="px-1 font-serif text-[26px] font-medium lg:px-0 lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">{heading}</h1>
      </header>

      {resubmitting && review?.note && (
        <div role="note" className="rounded-xl bg-warning-soft px-5 py-4 text-[15px] leading-normal text-warning-ink">
          <b className="font-semibold">
            {review.reviewer_name}, {reviewerAt(review.stage, { department }).replace(/^the /, '')}, asked for changes:
          </b>{' '}
          {review.note}
        </div>
      )}

      {errorCount > 0 && (
        <div role="alert" className="rounded-[10px] bg-danger-soft px-3.5 py-3 text-sm text-danger-ink">
          <b className="font-bold">{errorCount === 1 ? '1 thing to fix' : `${errorCount} things to fix`}</b> before it can be saved. What you typed is kept.
        </div>
      )}

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_380px] lg:gap-6">
        <section aria-label="The event" className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-[30px] lg:pt-[26px] lg:pb-[30px]">
          <PhoneHeading>What</PhoneHeading>
          <div className="flex flex-col gap-2">
            <span id="type-label" className="text-[13px] font-semibold text-ink-2">
              What kind of event
            </span>
            <div role="group" aria-labelledby="type-label" className="flex flex-wrap gap-2 lg:gap-1.5">
              {types.map((type) => {
                const active = form.event_type === type.value
                return (
                  <button
                    key={type.value}
                    type="button"
                    aria-pressed={active}
                    onClick={() => set('event_type', type.value)}
                    className={cn(
                      'min-h-10 rounded-full border px-3.5 text-sm lg:px-3',
                      active ? 'border-ink bg-ink font-semibold text-paper' : 'border-input bg-surface text-ink hover:border-ink-3',
                      errors.event_type && !active && 'border-danger',
                    )}
                  >
                    {type.label}
                  </button>
                )
              })}
            </div>
            <FieldError message={errors.event_type} />
          </div>

          <Labelled label="Title" error={errors.title}>
            {(props) => (
              <input
                {...props}
                ref={titleRef}
                type="text"
                maxLength={200}
                value={form.title}
                onChange={(e) => set('title', e.target.value)}
                className={cn(inputClass, 'text-[15px] lg:font-serif lg:text-2xl lg:font-medium', errors.title && errorRing)}
              />
            )}
          </Labelled>

          <PhoneHeading>When and where</PhoneHeading>
          <DateTime
            label="Starts"
            date={form.startDate}
            time={form.startTime}
            onDate={(v) => set('startDate', v)}
            onTime={(v) => set('startTime', v)}
            error={errors.starts}
          />
          <DateTime
            label="Ends"
            date={form.endDate}
            time={form.endTime}
            onDate={(v) => set('endDate', v)}
            onTime={(v) => set('endTime', v)}
            error={errors.ends}
            after={duration && <span className="font-mono text-[13px] text-ink-3">{duration}</span>}
          />

          <Labelled label="Where" error={errors.location}>
            {(props) => (
              <input
                {...props}
                type="text"
                maxLength={200}
                value={form.location}
                onChange={(e) => set('location', e.target.value)}
                className={cn(inputClass, 'text-[15px]', errors.location && errorRing)}
              />
            )}
          </Labelled>

          <Labelled label="About the event" optional error={errors.description}>
            {(props) => (
              <textarea
                {...props}
                rows={5}
                maxLength={5000}
                value={form.description}
                onChange={(e) => set('description', e.target.value)}
                className={cn(inputClass, 'resize-y font-serif text-[17px] leading-relaxed text-prose lg:text-lg', errors.description && errorRing)}
              />
            )}
          </Labelled>
        </section>

        <aside className="flex flex-col gap-4">
          <PhoneHeading>Who</PhoneHeading>
          <section aria-labelledby="invited-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 lg:p-5">
            <h2 id="invited-h" className="hidden font-serif text-[21px] font-medium lg:block">
              Who's invited
            </h2>
            <AudiencePicker
              label="Who's invited"
              value={form.audience}
              onChange={(a) => set('audience', a)}
              department={department}
              departments={departments}
              hideCollege={Boolean(department) && !privileged}
              error={errors.audience}
            />
            {department && !privileged && (
              <p className="text-[13px] text-ink-2">
                An event in <b className="font-semibold text-ink">{department.code}</b>, your department. Only the principal, admins and the placement office propose college-wide events.
              </p>
            )}
          </section>

          <section aria-labelledby="seats-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface py-4 pr-4 pl-5">
            <div className="flex items-center justify-between gap-3">
              <span className="flex flex-col gap-0.5">
                <span id="seats-h" className="text-sm font-semibold">
                  Seats
                </span>
                <span className="text-xs text-ink-2">Going stops at the limit</span>
              </span>
              <Segmented
                label="Seats"
                options={[
                  { value: 'none', label: 'No limit' },
                  { value: 'limit', label: 'Limit' },
                ]}
                value={form.limitSeats ? 'limit' : 'none'}
                onChange={(v) => set('limitSeats', v === 'limit')}
              />
            </div>
            {form.limitSeats && (
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-2.5">
                  <input
                    type="text"
                    inputMode="numeric"
                    aria-label="Seat limit"
                    aria-invalid={errors.capacity ? true : undefined}
                    value={form.capacity}
                    onChange={(e) => set('capacity', e.target.value)}
                    className={cn('min-h-11 w-[110px] rounded-lg border border-input bg-paper px-3 font-mono text-[15px] outline-none focus-visible:border-rust', errors.capacity && errorRing)}
                  />
                  <span className="text-[13px] text-ink-2">people can say Going. Others can still say Interested.</span>
                </div>
                <FieldError message={errors.capacity} />
              </div>
            )}
          </section>
        </aside>
      </div>

      {failure && (
        <p role="alert" className="rounded-lg bg-danger-soft px-4 py-3 text-sm font-semibold text-danger-ink">
          {failure}
        </p>
      )}

      <ActionBar
        sentence={
          <span>
            <span className="hidden font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase lg:block lg:text-left">What happens next</span>
            <span className="lg:text-ink">{routeKnown ? route.sentence : 'Checking who reviews this…'}</span>
          </span>
        }
      >
        <button type="button" onClick={handleDraft} disabled={saving} className={buttonStyles.secondary}>
          Save draft
        </button>
        <button type="button" onClick={handleSubmit} disabled={saving || !routeKnown} className={buttonStyles.primary}>
          {saving ? 'Saving…' : route.button}
        </button>
      </ActionBar>

      <LeaveDialog
        what="event"
        open={blocker.state === 'blocked'}
        canSaveDraft
        saving={saving}
        onSaveDraft={handleDraft}
        onKeepEditing={() => blocker.reset?.()}
        onDiscard={() => {
          leaving.current = true
          blocker.proceed?.()
        }}
      />
    </div>
  )
}

// The server's messages are lower-case fragments ("must be in the future").
function sentence(message: string) {
  return message.charAt(0).toUpperCase() + message.slice(1) + (message.endsWith('.') ? '' : '.')
}

// PhoneHeading groups the long form into steps on phones.
function PhoneHeading({ children }: { children: ReactNode }) {
  return <h2 className="-mb-1 px-1 font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase lg:hidden">{children}</h2>
}

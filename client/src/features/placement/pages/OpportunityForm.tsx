import { ChevronLeft, Lock } from 'lucide-react'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, Navigate, useBlocker, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { useDepartments } from '../../announcements/api'
import { batchOptions } from '../../announcements/audience'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { ComposeSkeleton } from '../../announcements/components/compose/ComposeSkeleton'
import { Segmented } from '../../announcements/components/compose/Fields'
import { errorRing, inputClass } from '../../announcements/components/compose/styles'
import { LeaveDialog } from '../../announcements/components/LeaveDialog'
import type { Department } from '../../announcements/types'
import { DateTime, FieldError, Labelled } from '../../events/components/FormFields'
import { ConfirmDialog } from '../../jobs/components/ConfirmDialog'
import { opportunityTypes, type Opportunity } from '../../jobs/types'
import { useCreateOpportunity, useManagedOne, usePublishing, useUpdateOpportunity } from '../api'
import { eligibilitySummary, fromRules, type EligibilityChoice } from '../eligibility'
import { COLLEGE_TIME_ZONE } from '../../../shared/time/college'
import {
  applyBy,
  checkOpportunity,
  emptyOpportunity,
  fromOpportunity,
  toInput,
  type OpportunityErrors,
  type OpportunityForm as Form,
} from '../opportunityForm'

// OpportunityFormPage writes a new Opportunity or edits one. A closed one
// can't change, so it opens its own page instead.
export default function OpportunityFormPage() {
  const { id } = useParams()
  const existing = useManagedOne(id)
  const departments = useDepartments()

  if ((id && existing.isPending) || departments.isPending) return <ComposeSkeleton />
  if (id && existing.error instanceof ApiRequestError && existing.error.status === 404) {
    return <EmptyState title="This opportunity doesn't exist">The link may be wrong.</EmptyState>
  }
  if ((id && existing.isError) || departments.isError) {
    return (
      <ErrorState
        message="The form could not be loaded."
        onRetry={() => {
          existing.refetch()
          departments.refetch()
        }}
      />
    )
  }
  const item = existing.data
  if (item && item.status === 'closed') return <Navigate to={`/placement/${item.id}`} replace />
  return <FormView key={item?.id ?? 'new'} item={item} departments={departments.data ?? []} />
}

type FieldKey = keyof OpportunityErrors

// Which error a field's change clears.
function errorFor(key: keyof Form): FieldKey {
  if (key === 'applyDate' || key === 'applyTime') return 'apply_by'
  if (key === 'application_mode') return 'external_url'
  return key as FieldKey
}

function FormView({ item, departments }: { item: Opportunity | undefined; departments: Department[] }) {
  const navigate = useNavigate()
  const live = item?.status === 'published'
  const [initial] = useState<Form>(() => (item ? fromOpportunity(item) : emptyOpportunity()))
  const [form, setForm] = useState(initial)
  const [errors, setErrors] = useState<OpportunityErrors>({})
  const [failure, setFailure] = useState('')
  const [confirming, setConfirming] = useState(false)
  const leaving = useRef(false)
  const titleRef = useRef<HTMLInputElement>(null)
  // Rules set some other way than the chips are kept unless the officer changes them.
  const customRules = item ? fromRules(item.eligibility) === null : false
  const [touchedEligibility, setTouchedEligibility] = useState(false)

  const create = useCreateOpportunity()
  const update = useUpdateOpportunity()
  const publishing = usePublishing()
  const saving = create.isPending || update.isPending || publishing.isPending

  const dirty = JSON.stringify(form) !== JSON.stringify(initial)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)
  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  function set<K extends keyof Form>(key: K, value: Form[K]) {
    setForm((f) => ({ ...f, [key]: value }))
    setErrors((e) => ({ ...e, [errorFor(key)]: undefined }))
    if (key === 'eligibility') setTouchedEligibility(true)
  }

  const who = eligibilitySummary(form.eligibility, departments)

  // save creates or updates, then publishes when asked. What was typed stays
  // in the form if anything fails.
  async function save(publish: boolean): Promise<string | null> {
    setFailure('')
    const found = checkOpportunity(form, new Date(), { publishing: publish || live })
    setErrors(found)
    if (Object.keys(found).length > 0) {
      if (found.title) titleRef.current?.focus()
      return null
    }
    const input = toInput(form)
    // A published one keeps how students apply, and unchanged rules stay as set.
    const body = {
      ...input,
      ...(live ? { application_mode: undefined } : {}),
      ...(customRules && !touchedEligibility ? { eligibility: undefined } : {}),
    }
    try {
      let saved = item ? await update.mutateAsync({ id: item.id, input: body }) : await create.mutateAsync(input)
      if (publish) saved = await publishing.mutateAsync({ id: saved.id, action: 'publish' })
      return saved.id
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400 && err.details) {
        setErrors(serverErrors(err.details))
        setFailure('Something needs changing before it can be saved. Check the fields marked in red.')
      } else if (err instanceof ApiRequestError && err.status === 409) {
        setFailure('It changed somewhere else in the meantime. Open it again to see where it stands.')
      } else if (err instanceof ApiRequestError && err.status === 403) {
        setFailure("You can't manage opportunities any more. Ask an admin if that's a mistake.")
      } else {
        setFailure("It couldn't be saved. Check your connection and try again; what you typed is still here.")
      }
      return null
    }
  }

  async function finish(publish: boolean) {
    const savedId = await save(publish)
    setConfirming(false)
    if (!savedId) return
    leaving.current = true
    if (blocker.state === 'blocked') blocker.proceed()
    else navigate(`/placement/${savedId}`, { replace: true })
  }

  function askToPublish() {
    const found = checkOpportunity(form, new Date(), { publishing: true })
    setErrors(found)
    if (Object.keys(found).length > 0) return
    setConfirming(true)
  }

  const errorCount = Object.values(errors).filter(Boolean).length
  const due = applyBy(form)
  const heading = !item ? 'New opportunity' : item.status === 'draft' ? 'Edit draft' : 'Edit opportunity'
  const backTo = item ? `/placement/${item.id}` : '/placement'

  return (
    <div className="group/page flex flex-col gap-5 pb-44 lg:pb-0">
      <header className="flex flex-col gap-1">
        <Link to={backTo} className="-ml-1.5 inline-flex min-h-10 items-center gap-1.5 self-start rounded-lg px-1.5 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-4" />
          {item ? item.title : 'Placement'}
        </Link>
        <h1 className="px-1 font-serif text-[26px] font-medium lg:px-0 lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">{heading}</h1>
      </header>

      {errorCount > 0 && (
        <div role="alert" className="rounded-[10px] bg-danger-soft px-3.5 py-3 text-sm text-danger-ink">
          <b className="font-bold">{errorCount === 1 ? '1 thing to fix' : `${errorCount} things to fix`}</b> before it can be saved. What you typed is kept.
        </div>
      )}

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_400px] lg:gap-6">
        <section aria-label="The opportunity" className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-[30px] lg:pt-[26px] lg:pb-[30px]">
          <PhoneHeading>What</PhoneHeading>
          <div className="flex flex-col gap-2">
            <span id="type-label" className="text-[13px] font-semibold text-ink-2">
              Type
            </span>
            <div role="group" aria-labelledby="type-label" className="flex flex-wrap gap-2">
              {opportunityTypes.map((type) => (
                <Chip key={type.value} active={form.opportunity_type === type.value} invalid={Boolean(errors.opportunity_type)} onClick={() => set('opportunity_type', type.value)}>
                  {type.label}
                </Chip>
              ))}
            </div>
            <FieldError message={errors.opportunity_type} />
          </div>

          <Labelled label="Role" error={errors.title}>
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
          <Labelled label="Company" error={errors.company}>
            {(props) => (
              <input {...props} type="text" maxLength={200} value={form.company} onChange={(e) => set('company', e.target.value)} className={cn(inputClass, 'text-[15px]', errors.company && errorRing)} />
            )}
          </Labelled>
          <div className="flex flex-col gap-4 lg:flex-row lg:gap-3">
            <div className="lg:w-[260px]">
              <Labelled label="Location" optional error={errors.location}>
                {(props) => (
                  <input {...props} type="text" maxLength={200} value={form.location} onChange={(e) => set('location', e.target.value)} className={cn(inputClass, 'text-[15px]', errors.location && errorRing)} />
                )}
              </Labelled>
            </div>
            <div className="flex flex-col gap-1.5 lg:w-[260px]">
              <Labelled label="Stipend or CTC" optional error={errors.compensation}>
                {(props) => (
                  <input {...props} type="text" maxLength={200} value={form.compensation} onChange={(e) => set('compensation', e.target.value)} className={cn(inputClass, 'text-[15px]', errors.compensation && errorRing)} />
                )}
              </Labelled>
              <span className="text-xs text-ink-3">As students should read it: 4.5 LPA, ₹20,000 a month</span>
            </div>
          </div>
          <DateTime
            label="Apply by"
            date={form.applyDate}
            time={form.applyTime}
            onDate={(v) => set('applyDate', v)}
            onTime={(v) => set('applyTime', v)}
            error={errors.apply_by}
            after={live ? <span className="text-xs text-ink-3">Only later than now</span> : undefined}
          />
          <Labelled label="About the role" optional error={errors.description}>
            {(props) => (
              <textarea
                {...props}
                rows={5}
                maxLength={10000}
                value={form.description}
                onChange={(e) => set('description', e.target.value)}
                className={cn(inputClass, 'resize-y font-serif text-[17px] leading-relaxed text-prose lg:text-lg', errors.description && errorRing)}
              />
            )}
          </Labelled>
        </section>

        <aside className="flex flex-col gap-4">
          <PhoneHeading>Who can apply</PhoneHeading>
          <WhoCanApply
            value={form.eligibility}
            onChange={(v) => set('eligibility', v)}
            departments={departments}
            summary={customRules && !touchedEligibility ? 'the groups already set for it (change them here to replace them)' : who}
            error={errors.eligibility}
          />
          <PhoneHeading>How students apply</PhoneHeading>
          <section aria-labelledby="how-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 lg:p-5">
            <h2 id="how-h" className="hidden font-serif text-[21px] font-medium lg:block">
              How students apply
            </h2>
            {live ? (
              <>
                <p className="flex min-h-11 items-center gap-2 rounded-lg bg-rail px-3 text-sm text-ink-2">
                  <Lock aria-hidden="true" className="size-4" />
                  <b className="font-semibold text-ink">{form.application_mode === 'internal' ? 'In LINKS' : "On the company's site"}</b>
                </p>
                <p className="text-[13px] text-ink-2">This can't change after publishing, so applications already made stay valid.</p>
              </>
            ) : (
              <>
                <Segmented
                  label="How students apply"
                  full
                  options={[
                    { value: 'internal', label: 'In LINKS' },
                    { value: 'external', label: 'On company site' },
                  ]}
                  value={form.application_mode}
                  onChange={(v) => set('application_mode', v === 'external' ? 'external' : 'internal')}
                />
                {form.application_mode === 'internal' ? (
                  <p className="text-[13px] text-ink-2">Students apply with one tap. You see them in the applicant list and move them through shortlisting.</p>
                ) : null}
              </>
            )}
            {form.application_mode === 'external' && (
              <Labelled label="Company's apply link" error={errors.external_url}>
                {(props) => (
                  <input
                    {...props}
                    type="url"
                    inputMode="url"
                    placeholder="https://"
                    value={form.external_url}
                    onChange={(e) => set('external_url', e.target.value)}
                    className={cn(inputClass, 'font-mono text-[13px]', errors.external_url && errorRing)}
                  />
                )}
              </Labelled>
            )}
            {form.application_mode === 'external' && !errors.external_url && (
              <p className="text-xs text-ink-3">Students mark in LINKS that they applied, so you still see who did.</p>
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
            <span className="hidden font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase lg:block lg:text-left">
              {live ? 'Editing a live opportunity' : 'What happens next'}
            </span>
            <span className="lg:text-ink">
              {live ? (
                'Changes show to students right away. Applications already made are kept.'
              ) : (
                <>
                  Saved as a draft until you publish. Publishing shows it in Jobs for <b className="font-semibold">{who}</b>.
                </>
              )}
            </span>
          </span>
        }
      >
        {live ? (
          <>
            <Link to={backTo} className={buttonStyles.secondary}>
              Discard changes
            </Link>
            <button type="button" onClick={() => finish(false)} disabled={saving} className={buttonStyles.primary}>
              {saving ? 'Saving…' : 'Save changes'}
            </button>
          </>
        ) : (
          <>
            <button type="button" onClick={() => finish(false)} disabled={saving} className={buttonStyles.secondary}>
              Save draft
            </button>
            <button type="button" onClick={askToPublish} disabled={saving} className={buttonStyles.primary}>
              Publish
            </button>
          </>
        )}
      </ActionBar>

      <ConfirmDialog
        open={confirming}
        title={`Publish to ${who}?`}
        cancelLabel="Keep editing"
        confirmLabel="Publish"
        busyLabel="Publishing…"
        busy={saving}
        onCancel={() => setConfirming(false)}
        onConfirm={() => finish(true)}
      >
        <p>
          It shows in their Jobs right away
          {due && (
            <>
              , open until <b className="font-semibold text-ink">{due.toLocaleDateString('en-IN', { weekday: 'short', day: 'numeric', month: 'short', timeZone: COLLEGE_TIME_ZONE })}, {due.toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit', hour12: true, timeZone: COLLEGE_TIME_ZONE }).toLowerCase()}</b>
            </>
          )}
          . After publishing you can edit the details and move the deadline later, but not change how students apply.
        </p>
      </ConfirmDialog>

      <LeaveDialog
        what="opportunity"
        open={blocker.state === 'blocked'}
        canSaveDraft={!live}
        saving={saving}
        onSaveDraft={() => finish(false)}
        onKeepEditing={() => blocker.reset?.()}
        onDiscard={() => {
          leaving.current = true
          blocker.proceed?.()
        }}
      />
    </div>
  )
}

// The four most recent batches are the students still on campus.
const batches = batchOptions().slice(0, 4).reverse()

function WhoCanApply({ value, onChange, departments, summary, error }: {
  value: EligibilityChoice
  onChange: (value: EligibilityChoice) => void
  departments: Department[]
  summary: string
  error?: string
}) {
  function toggle<T>(list: T[], item: T) {
    return list.includes(item) ? list.filter((x) => x !== item) : [...list, item]
  }
  return (
    <section aria-labelledby="who-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 lg:p-5">
      <h2 id="who-h" className="hidden font-serif text-[21px] font-medium lg:block">
        Who can apply
      </h2>
      <div role="radiogroup" aria-label="Who can apply" className="flex flex-col gap-0.5">
        <label className={cn('flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm', value.everyone && 'bg-well font-semibold')}>
          <input type="radio" name="who" checked={value.everyone} onChange={() => onChange({ ...value, everyone: true })} className="size-4 accent-ink" />
          Every student
        </label>
        <label className={cn('flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm', !value.everyone && 'bg-well font-semibold')}>
          <input type="radio" name="who" checked={!value.everyone} onChange={() => onChange({ ...value, everyone: false })} className="size-4 accent-ink" />
          Choose departments and batches
        </label>
      </div>
      {!value.everyone && (
        <>
          <div className="flex flex-col gap-2">
            <span id="dept-label" className="text-[13px] font-semibold text-ink-2">
              Departments
            </span>
            <div role="group" aria-labelledby="dept-label" className="flex flex-wrap gap-2">
              <Chip active={value.departments.length === 0} onClick={() => onChange({ ...value, departments: [] })}>
                All
              </Chip>
              {departments.map((d) => (
                <Chip key={d.id} active={value.departments.includes(d.id)} onClick={() => onChange({ ...value, departments: toggle(value.departments, d.id) })}>
                  {d.code}
                </Chip>
              ))}
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span id="batch-label" className="text-[13px] font-semibold text-ink-2">
              Batch
            </span>
            <div role="group" aria-labelledby="batch-label" className="flex flex-wrap gap-2">
              <Chip active={value.batches.length === 0} onClick={() => onChange({ ...value, batches: [] })}>
                All
              </Chip>
              {batches.map((year) => (
                <Chip key={year} active={value.batches.includes(year)} onClick={() => onChange({ ...value, batches: toggle(value.batches, year) })}>
                  {String(year)}
                </Chip>
              ))}
            </div>
          </div>
        </>
      )}
      <p className="rounded-lg bg-paper px-3 py-2.5 text-sm" aria-live="polite">
        Shows in Jobs for <b className="font-semibold">{summary}</b>.
      </p>
      <FieldError message={error} />
    </section>
  )
}

function Chip({ active, invalid, onClick, children }: { active: boolean; invalid?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        'min-h-10 rounded-full border px-3.5 text-sm',
        active ? 'border-ink bg-ink font-semibold text-paper' : 'border-input bg-surface text-ink hover:border-ink-3',
        invalid && !active && 'border-danger',
      )}
    >
      {children}
    </button>
  )
}

// PhoneHeading groups the long form into steps on phones.
function PhoneHeading({ children }: { children: ReactNode }) {
  return <h2 className="-mb-1 px-1 font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase lg:hidden">{children}</h2>
}

// serverErrors maps a 400's field details onto the form's errors.
function serverErrors(details: Record<string, unknown>): OpportunityErrors {
  const fields: Record<string, FieldKey> = {
    opportunity_type: 'opportunity_type',
    title: 'title',
    company: 'company',
    description: 'description',
    location: 'location',
    compensation: 'compensation',
    apply_by: 'apply_by',
    external_url: 'external_url',
    application_mode: 'external_url',
    eligibility: 'eligibility',
  }
  const errors: OpportunityErrors = {}
  for (const [field, message] of Object.entries(details)) {
    const key = fields[field]
    if (key && typeof message === 'string') errors[key] = message.charAt(0).toUpperCase() + message.slice(1) + (message.endsWith('.') ? '' : '.')
  }
  return errors
}

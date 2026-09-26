import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { Link, Navigate, useBlocker, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useModal } from '../../../shared/ui/useModal'
import { useAuthStore } from '../../auth/store'
import { useDashboard } from '../../home/api'
import { CategoryTag } from '../../notices/components/CategoryTag'
import { categories, type Category } from '../../notices/types'
import { useAuthored, useCreate, useDepartments, usePreview, useSubmit, useUpdate } from '../api'
import { describe, presetsFor, toRules } from '../audience'
import { buttonStyles } from '../buttons'
import { ActionBar } from '../components/ActionBar'
import { AudiencePicker } from '../components/AudiencePicker'
import { LeaveDialog } from '../components/LeaveDialog'
import { approverPhrase } from '../status'
import type { Authored, Department, Draft, Preview, RuleInput } from '../types'

// Everyone who can post may use any category, except the placement officer,
// who posts placement notices only (ADR 0017, CanPost on the server).
const allCategoryRoles = ['principal', 'admin', 'hod', 'faculty', 'student_coordinator']

type Mode = 'new' | 'draft' | 'published'
type Errors = Partial<Record<'title' | 'body' | 'audience' | 'expires_at' | 'category', string>>

// Compose writes a new Announcement or edits one of the author's own.
export default function Compose() {
  const { id } = useParams()
  const existing = useAuthored(id)
  const dashboard = useDashboard()
  const departments = useDepartments()

  if ((id && existing.isPending) || dashboard.isPending || departments.isPending) {
    return <ComposeSkeleton />
  }
  if ((id && existing.isError) || dashboard.isError || departments.isError) {
    return (
      <ErrorState
        message="The composer could not be loaded."
        onRetry={() => {
          existing.refetch()
          dashboard.refetch()
          departments.refetch()
        }}
      />
    )
  }
  const item = existing.data
  // Waiting or ended announcements can't be edited; show them instead.
  if (item && (item.status === 'pending' || item.status === 'withdrawn' || item.edit?.status === 'pending')) {
    return <Navigate to={`/mine/${item.id}`} replace />
  }
  return (
    <ComposeForm
      key={item?.id ?? 'new'}
      item={item}
      department={dashboard.data?.user.department ?? null}
      departments={departments.data ?? []}
    />
  )
}

function initialDraft(item: Authored | undefined, category: Category, audience: RuleInput[]): Draft {
  if (!item) return { title: '', body: '', category, audience, expires_at: null }
  // A sent-back edit starts from the author's edit, not the live text.
  const source = item.edit?.status === 'rejected' ? item.edit : item
  return {
    title: source.title,
    body: source.body,
    category: source.category,
    audience: toRules(source.audience),
    expires_at: source.expires_at,
  }
}

function ComposeForm({ item, department, departments }: {
  item: Authored | undefined
  department: { id: string; code: string } | null
  departments: Department[]
}) {
  const navigate = useNavigate()
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  const isDesktop = useIsDesktop()
  const allowed = roles.some((r) => allCategoryRoles.includes(r)) ? categories : categories.filter((c) => c.value === 'placement')
  const mode: Mode = !item ? 'new' : item.status === 'published' ? 'published' : 'draft'
  const canSaveDraft = mode !== 'published'

  const defaultAudience = department ? presetsFor(department)[2].audience : []
  const [initial] = useState(() => initialDraft(item, allowed.some((c) => c.value === 'department') ? 'department' : allowed[0].value, defaultAudience))
  const [draft, setDraft] = useState<Draft>(initial)
  const [errors, setErrors] = useState<Errors>({})
  const [failure, setFailure] = useState('')
  const [view, setView] = useState<'write' | 'preview'>('write')
  const [sheetOpen, setSheetOpen] = useState(false)
  const leaving = useRef(false)
  const titleRef = useRef<HTMLInputElement>(null)
  const bodyRef = useRef<HTMLTextAreaElement>(null)

  const create = useCreate()
  const update = useUpdate()
  const submit = useSubmit()
  const saving = create.isPending || update.isPending || submit.isPending
  const preview = usePreview(draft.category, draft.audience)
  const codes = new Map(departments.map((d) => [d.id, d.code]))

  const dirty = JSON.stringify(draft) !== JSON.stringify(initial)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)

  // Closing the tab with unsaved text asks the browser's own question.
  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  function set<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((d) => ({ ...d, [key]: value }))
    setErrors((e) => ({ ...e, [key]: undefined }))
  }

  function check(): boolean {
    const found: Errors = {}
    if (!draft.title.trim()) found.title = 'Add a title so readers know what this is about.'
    if (!draft.body.trim()) found.body = 'Write the notice itself.'
    setErrors(found)
    if (found.title) titleRef.current?.focus()
    else if (found.body) bodyRef.current?.focus()
    return !found.title && !found.body
  }

  // save posts, saves or submits. The text stays in the form if it fails.
  async function save(asDraft: boolean): Promise<string | null> {
    setFailure('')
    if (!check()) return null
    const input = { ...draft, title: draft.title.trim(), body: draft.body.trim() }
    try {
      let result: Authored
      if (!item) {
        result = await create.mutateAsync({ ...input, draft: asDraft })
      } else {
        result = await update.mutateAsync({ id: item.id, input })
        if (mode === 'draft' && !asDraft) result = await submit.mutateAsync(item.id)
      }
      return result.id
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400 && err.details) {
        setErrors(err.details as Errors)
        setFailure('Some details need fixing before this can be sent.')
      } else if (err instanceof ApiRequestError && err.status === 409) {
        setFailure('This announcement changed somewhere else. Reload it to see its current state.')
      } else {
        setFailure("It couldn't be saved. Check your connection and try again; your text is still here.")
      }
      return null
    }
  }

  async function handleSend() {
    const savedId = await save(false)
    if (!savedId) return
    leaving.current = true
    navigate(`/mine/${savedId}`, { replace: true })
  }

  async function handleDraft() {
    const savedId = await save(true)
    if (!savedId) return
    leaving.current = true
    if (blocker.state === 'blocked') blocker.proceed()
    else navigate('/mine?status=draft', { replace: true })
  }

  const outcome = outcomeFor(preview.data, mode)
  const heading = mode === 'new' ? 'New announcement' : mode === 'published' ? 'Edit announcement' : 'Edit draft'
  const note = item?.edit?.status === 'rejected' ? item.edit.review_note : item?.status === 'rejected' ? item.review_note : undefined
  const reviewer = item?.edit?.approver ?? item?.approver ?? 'Approver'

  return (
    <div className="group/page flex flex-col gap-5 pb-40 lg:pb-0">
      <header className="flex items-end justify-between gap-4">
        <div className="flex items-center gap-1 lg:flex-col lg:items-start lg:gap-1">
          <Link to="/mine" aria-label="Close" className="flex size-11 items-center justify-center rounded-full text-ink hover:text-ink lg:hidden">
            <X aria-hidden="true" className="size-5" />
          </Link>
          <Link to="/mine" className="hidden min-h-10 items-center gap-1.5 text-sm font-semibold lg:inline-flex">
            <ChevronLeft aria-hidden="true" className="size-4" />
            My announcements
          </Link>
          <h1 className="font-serif text-[22px] font-medium lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">{heading}</h1>
        </div>
        <Segmented
          label="Mode"
          options={[{ value: 'write', label: 'Write' }, { value: 'preview', label: 'Preview' }]}
          value={view}
          onChange={(v) => setView(v as 'write' | 'preview')}
          className="hidden lg:flex"
        />
      </header>
      <Segmented
        label="Mode"
        options={[{ value: 'write', label: 'Write' }, { value: 'preview', label: 'Preview' }]}
        value={view}
        onChange={(v) => setView(v as 'write' | 'preview')}
        className="lg:hidden"
        full
      />

      {note && (
        <div role="note" className="rounded-xl bg-danger-soft px-5 py-4 text-[15px] leading-normal text-danger-ink">
          <b className="font-semibold">{reviewer} sent this back:</b> {note}
        </div>
      )}

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_400px] lg:gap-6">
        {view === 'write' ? (
          <section aria-label="Writing" className="flex flex-col gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-8 lg:pt-7 lg:pb-8">
            <Field label="Category" error={errors.category}>
              <Segmented
                label="Category"
                options={allowed.map((c) => ({ value: c.value, label: c.label, short: c.short }))}
                value={draft.category}
                onChange={(v) => set('category', v as Category)}
                full={!isDesktop}
              />
            </Field>
            <TextField
              label="Title"
              error={errors.title}
              input={(props) => (
                <input
                  {...props}
                  ref={titleRef}
                  type="text"
                  value={draft.title}
                  onChange={(e) => set('title', e.target.value)}
                  maxLength={200}
                  className={cn(inputClass, 'font-serif text-xl font-medium lg:text-2xl', errors.title && errorRing)}
                />
              )}
            />
            <TextField
              label="Notice"
              error={errors.body}
              hint="A blank line starts a new paragraph. Use Preview to see it as readers will."
              input={(props) => (
                <textarea
                  {...props}
                  ref={bodyRef}
                  rows={isDesktop ? 10 : 6}
                  value={draft.body}
                  onChange={(e) => set('body', e.target.value)}
                  className={cn(inputClass, 'resize-y font-serif text-[17px] leading-relaxed text-prose lg:text-lg', errors.body && errorRing)}
                />
              )}
            />
          </section>
        ) : (
          <PreviewArticle draft={draft} audience={describe(draft.audience, codes)} />
        )}

        <aside className="flex flex-col gap-4">
          {isDesktop ? (
            <section aria-labelledby="who-h" className="flex flex-col gap-2.5 rounded-xl border border-line bg-surface p-5">
              <h2 id="who-h" className="font-serif text-[21px] font-medium">Who sees it</h2>
              <AudiencePicker
                value={draft.audience}
                onChange={(a) => set('audience', a)}
                department={department}
                departments={departments}
                category={draft.category}
                error={errors.audience}
              />
            </section>
          ) : (
            <PhoneRows
              audience={describe(draft.audience, codes)}
              reach={preview.data?.reach}
              expiry={draft.expires_at}
              audienceError={errors.audience}
              onOpenAudience={() => setSheetOpen(true)}
              onExpiry={(v) => set('expires_at', v)}
              expiryError={errors.expires_at}
            />
          )}
          {isDesktop && <ExpiryField value={draft.expires_at} onChange={(v) => set('expires_at', v)} error={errors.expires_at} />}
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
            {outcome.sentence}
          </span>
        }
      >
        {canSaveDraft && (
          <button type="button" onClick={handleDraft} disabled={saving} className={buttonStyles.secondary}>
            Save draft
          </button>
        )}
        <button type="button" onClick={handleSend} disabled={saving || !preview.data} className={buttonStyles.primary}>
          {saving ? 'Sending…' : outcome.button}
        </button>
      </ActionBar>

      {!isDesktop && (
        <AudienceSheet open={sheetOpen} onClose={() => setSheetOpen(false)}>
          <AudiencePicker
            value={draft.audience}
            onChange={(a) => set('audience', a)}
            department={department}
            departments={departments}
            category={draft.category}
          />
        </AudienceSheet>
      )}

      <LeaveDialog
        open={blocker.state === 'blocked'}
        canSaveDraft={canSaveDraft}
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

// outcomeFor turns the publishing preview into the sentence beside the
// buttons and the main button's label.
function outcomeFor(preview: Preview | undefined, mode: Mode) {
  if (!preview) return { sentence: 'Checking who approves this…', button: mode === 'published' ? 'Save changes' : 'Send' }
  const people = `${preview.reach.toLocaleString('en-IN')} ${preview.reach === 1 ? 'person sees' : 'people see'} it`
  const approver = approverPhrase(preview.approver)
  if (mode === 'published') {
    return preview.publishes_directly
      ? { sentence: <><b className="font-semibold text-ink">Your changes go live now.</b> {people} straight away.</>, button: 'Publish changes' }
      : { sentence: <>Your changes go to <b className="font-semibold text-ink">{approver}</b>. Readers keep the current version until then.</>, button: 'Submit changes' }
  }
  return preview.publishes_directly
    ? { sentence: <><b className="font-semibold text-ink">Publishes now.</b> {people} straight away.</>, button: 'Publish' }
    : { sentence: <>Goes to <b className="font-semibold text-ink">{approver}</b> for approval. Readers see it once it's approved.</>, button: 'Submit for approval' }
}

const inputClass = 'w-full rounded-[10px] border border-line bg-surface px-4 py-3 text-ink outline-none focus-visible:border-rust lg:bg-paper'
const errorRing = 'border-[1.5px] border-danger focus-visible:border-danger'

function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold text-ink-2">{label}</span>
      {children}
      {error && <p className="text-[13px] font-semibold text-danger">{error}</p>}
    </div>
  )
}

type InputProps = { id: string; 'aria-invalid'?: boolean; 'aria-describedby'?: string }

function TextField({ label, error, hint, input }: { label: string; error?: string; hint?: string; input: (props: InputProps) => ReactNode }) {
  const id = useId()
  const describedBy = [error && `${id}-error`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-[13px] font-semibold text-ink-2">
        {label}
      </label>
      {input({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy })}
      {error && (
        <p id={`${id}-error`} className="text-[13px] font-semibold text-danger">
          {error}
        </p>
      )}
      {hint && (
        <p id={`${id}-hint`} className="hidden text-xs text-ink-3 lg:block">
          {hint}
        </p>
      )}
    </div>
  )
}

function Segmented({ label, options, value, onChange, className, full }: {
  label: string
  options: { value: string; label: string; short?: string }[]
  value: string
  onChange: (value: string) => void
  className?: string
  full?: boolean
}) {
  return (
    <div role="group" aria-label={label} className={cn('flex gap-1 self-start rounded-[10px] bg-well p-1', full && 'self-stretch', className)}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={option.value === value}
          onClick={() => onChange(option.value)}
          className={cn(
            'min-h-9 rounded-[7px] px-4 text-sm',
            full && 'flex-1 px-2',
            option.value === value ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2 hover:text-ink',
          )}
        >
          {full && option.short ? option.short : option.label}
        </button>
      ))}
    </div>
  )
}

// Expiry is a date; the notice leaves the feed at the end of that day, in the
// author's time zone.
function toEndOfDay(date: string): string {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y, m - 1, d, 23, 59, 59).toISOString()
}

function toDateInput(iso: string | null): string {
  if (!iso) return ''
  const date = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function friendlyDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-IN', { weekday: 'short', day: 'numeric', month: 'short' })
}

function ExpiryField({ value, onChange, error }: { value: string | null; onChange: (v: string | null) => void; error?: string }) {
  const id = useId()
  return (
    <section aria-label="Expiry" className="flex flex-col gap-2 rounded-xl border border-line bg-surface py-3.5 pr-3 pl-5">
      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <label htmlFor={id} className="text-sm font-semibold">
            Expires <span className="font-normal text-ink-3">(optional)</span>
          </label>
          <span className="text-xs text-ink-2">
            {value ? (
              <>
                Leaves the feed after <b className="font-semibold">{friendlyDate(value)}</b>
              </>
            ) : (
              'Stays in the feed until you withdraw it.'
            )}
          </span>
        </div>
        <span className="flex items-center gap-0.5">
          <input
            id={id}
            type="date"
            value={toDateInput(value)}
            min={toDateInput(new Date().toISOString())}
            onChange={(e) => onChange(e.target.value ? toEndOfDay(e.target.value) : null)}
            className="min-h-10 rounded-[7px] border border-input bg-paper px-2.5 font-mono text-sm"
          />
          {value && (
            <button
              type="button"
              aria-label="Remove expiry"
              onClick={() => onChange(null)}
              className="flex size-10 items-center justify-center rounded-lg text-ink-3 hover:bg-well hover:text-ink"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          )}
        </span>
      </div>
      {error && <p className="text-[13px] font-semibold text-danger">{error}</p>}
    </section>
  )
}

function PhoneRows({ audience, reach, expiry, audienceError, expiryError, onOpenAudience, onExpiry }: {
  audience: string
  reach: number | undefined
  expiry: string | null
  audienceError?: string
  expiryError?: string
  onOpenAudience: () => void
  onExpiry: (v: string | null) => void
}) {
  const id = useId()
  return (
    <div className="flex flex-col rounded-[10px] border border-line bg-surface">
      <button type="button" onClick={onOpenAudience} className="flex min-h-14 items-center justify-between gap-3 px-4 py-2 text-left">
        <span className="flex flex-col">
          <span className="text-xs text-ink-3">Who sees it</span>
          <span className="text-[15px] font-semibold">{audience}</span>
          {audienceError && <span className="text-[13px] font-semibold text-danger">{audienceError}</span>}
        </span>
        <span className="flex items-center gap-1 font-mono text-xs text-ink-3">
          {reach !== undefined && reach.toLocaleString('en-IN')}
          <ChevronRight aria-hidden="true" className="size-4" />
        </span>
      </button>
      <div className="flex min-h-14 items-center justify-between gap-3 px-4 py-2 shadow-[0_-1px_0_var(--color-line)]">
        <label htmlFor={id} className="flex flex-col">
          <span className="text-xs text-ink-3">Expires (optional)</span>
          <span className="text-[15px]">{expiry ? <>After <b className="font-semibold">{friendlyDate(expiry)}</b></> : 'No end date'}</span>
          {expiryError && <span className="text-[13px] font-semibold text-danger">{expiryError}</span>}
        </label>
        <input
          id={id}
          type="date"
          value={toDateInput(expiry)}
          min={toDateInput(new Date().toISOString())}
          onChange={(e) => onExpiry(e.target.value ? toEndOfDay(e.target.value) : null)}
          className="min-h-10 w-[136px] rounded-[7px] border border-input bg-paper px-2 font-mono text-[13px]"
        />
      </div>
    </div>
  )
}

function AudienceSheet({ open, onClose, children }: { open: boolean; onClose: () => void; children: ReactNode }) {
  const ref = useModal(open)
  return (
    <dialog
      ref={ref}
      aria-labelledby="sheet-h"
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-0 mt-auto max-h-[85dvh] w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45"
    >
      <div className="flex max-h-[85dvh] flex-col">
        <header className="flex items-center justify-between py-3 pr-3 pl-5">
          <h2 id="sheet-h" className="font-serif text-2xl font-medium">Who sees it</h2>
          <button type="button" aria-label="Close" onClick={onClose} className="flex size-11 items-center justify-center rounded-full text-ink-3">
            <X aria-hidden="true" className="size-5" />
          </button>
        </header>
        <div className="flex-1 overflow-y-auto px-4 pb-3">{children}</div>
        <footer className="bg-surface px-4 pt-3 pb-6 shadow-[0_-1px_0_var(--color-line)]">
          <button type="button" onClick={onClose} className={buttonStyles.primary + ' min-h-12 w-full'}>
            Done
          </button>
        </footer>
      </div>
    </dialog>
  )
}

function PreviewArticle({ draft, audience }: { draft: Draft; audience: string }) {
  const paragraphs = draft.body.trim() ? draft.body.split(/\n\s*\n/) : []
  return (
    <section aria-label="Preview" className="flex flex-col gap-2">
      <p className="px-1 text-xs text-ink-3">This is how {audience} will see it.</p>
      <article className="flex flex-col gap-4 rounded-xl border border-line bg-surface px-5 py-5 lg:px-10 lg:py-9">
        <div className="flex items-center gap-2.5 text-xs">
          <CategoryTag category={draft.category} className="lg:text-xs" />
          <span className="text-ink-3">Posted when it goes live</span>
        </div>
        <h2 className="font-serif text-[26px] leading-tight font-medium tracking-[-0.4px] lg:text-[36px]">
          {draft.title.trim() || <span className="text-ink-3">Your title</span>}
        </h2>
        <div className="flex max-w-[680px] flex-col gap-3 font-serif text-[17px] leading-relaxed text-prose lg:text-[19px]">
          {paragraphs.length === 0 ? (
            <p className="text-ink-3">Your notice will appear here.</p>
          ) : (
            paragraphs.map((text, i) => (
              <p key={i} className="whitespace-pre-line">
                {text}
              </p>
            ))
          )}
        </div>
      </article>
    </section>
  )
}

function ComposeSkeleton() {
  return (
    <div className="flex flex-col gap-5">
      <LoadingStatus label="Loading the composer" />
      <Skeleton className="h-9 w-64" />
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_400px]">
        <div className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-6">
          <Skeleton className="h-9 w-72" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
        <div className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-5">
          <Skeleton className="h-6 w-32" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    </div>
  )
}

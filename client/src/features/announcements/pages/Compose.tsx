import { ChevronLeft, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Link, Navigate, useBlocker, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { ErrorState } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useAuthStore } from '../../auth/store'
import { useDashboard } from '../../home/api'
import { categories, type Category } from '../../notices/types'
import { useAuthored, useCreate, useDepartments, usePreview, useSubmit, useUpdate } from '../api'
import { describe, presetsFor, toRules } from '../audience'
import { buttonStyles } from '../buttons'
import { ActionBar } from '../components/ActionBar'
import { AudiencePicker } from '../components/AudiencePicker'
import { AudienceSheet } from '../components/compose/AudienceSheet'
import { ComposeSkeleton } from '../components/compose/ComposeSkeleton'
import { ExpiryField } from '../components/compose/ExpiryField'
import { Field, Segmented, TextField } from '../components/compose/Fields'
import { outcomeFor, type Mode } from '../components/compose/outcome'
import { PhoneRows } from '../components/compose/PhoneRows'
import { PreviewArticle } from '../components/compose/PreviewArticle'
import { errorRing, inputClass } from '../components/compose/styles'
import { LeaveDialog } from '../components/LeaveDialog'
import type { Authored, Department, Draft, RuleInput } from '../types'

// Everyone who can post may use any category, except the placement officer,
// who posts placement notices only (ADR 0017, CanPost on the server).
const allCategoryRoles = ['principal', 'admin', 'hod', 'faculty', 'student_coordinator']

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

import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { ActionBar } from '../../announcements/components/ActionBar'
import { Chip } from '../../announcements/components/queue/Chip'
import { buttonStyles } from '../../announcements/buttons'
import type { Banner } from '../../announcements/components/queue/Review'
import { waited } from '../../announcements/status'
import { useDecide } from '../api'
import { FixEmail } from '../../lists/components/RowActions'
import { whyItsHere } from '../reason'
import type { AccessRequest } from '../types'

type Mode = 'idle' | 'confirm' | 'reject'

// RequestReview shows one Access request and the bar to approve or reject it.
export function RequestReview({ request, focusName, onDecided }: {
  request: AccessRequest
  focusName: boolean
  onDecided: (request: AccessRequest, banner: Banner) => void
}) {
  const [mode, setMode] = useState<Mode>('idle')
  const [note, setNote] = useState('')
  const [noteError, setNoteError] = useState('')
  const [failure, setFailure] = useState('')
  // A reported row's email can be fixed right here (#174).
  const [fixing, setFixing] = useState(false)
  const decide = useDecide()
  const nameRef = useRef<HTMLHeadingElement>(null)
  const id = request.student_identity
  const name = request.profile?.full_name ?? request.email ?? 'This person'
  const first = name.split(' ')[0]

  useEffect(() => {
    if (focusName) nameRef.current?.focus()
  }, [focusName])

  async function submit(decision: 'approve' | 'reject') {
    setFailure('')
    if (decision === 'reject' && !note.trim()) {
      setNoteError("Write a short reason. It's kept with the decision.")
      return
    }
    try {
      await decide.mutateAsync({ id: request.id, decision, note: note.trim() })
      onDecided(request, {
        tone: 'done',
        text:
          decision === 'approve'
            ? `${name} can sign in now. They're in ${id?.department_name ?? 'their department'}, batch ${id?.batch_year ?? ''}.`.replace(', batch .', '.')
            : `${name}'s request was rejected. Your note is kept with the decision.`,
      })
    } catch (err) {
      // 409: someone approved it first; 404: someone rejected it, or it left
      // the viewer's scope.
      if (err instanceof ApiRequestError && (err.status === 409 || err.status === 404)) {
        onDecided(request, { tone: 'info', text: `Someone else already decided ${name}'s request. The list has moved on.` })
      } else {
        setFailure("That didn't go through. Check your connection and try again.")
      }
    }
  }

  return (
    <section aria-label="Selected request" className="flex min-w-0 flex-col gap-3">
      <article className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
        <div className="flex flex-wrap items-center gap-2">
          <span className="rounded bg-[#e3e8eb] px-1.5 py-0.5 text-[11px] leading-4 font-semibold text-[#2f4250]">Access request</span>
          {request.reported_at && <Chip tone="warning">Reported on first sign-in</Chip>}
          <span className="font-mono text-xs text-ink-3">{waited(request.created_at).text}</span>
        </div>
        <h2
          ref={nameRef}
          tabIndex={-1}
          className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] outline-none lg:text-[32px] lg:tracking-[-0.5px]"
        >
          {name}
        </h2>
        <dl className="grid grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-2.5 text-[15px] lg:grid-cols-[120px_minmax(0,1fr)] lg:gap-x-4">
          <dt className="text-ink-3">Email</dt>
          {fixing ? (
            <dd>
              <FixEmail
                person={{ user_id: request.id, full_name: name, email: request.email ?? '' }}
                onCancel={() => setFixing(false)}
                onDone={(outcome) =>
                  onDecided(request, {
                    tone: outcome.tone === 'ok' ? 'done' : 'info',
                    text: outcome.tone === 'ok' ? `${outcome.text} The row is off this list and waits for their first sign-in.` : outcome.text,
                  })
                }
              />
            </dd>
          ) : (
            <dd className="flex flex-wrap items-baseline gap-x-3 font-semibold break-all">
              {request.email}
              {request.reported_at && (
                <button type="button" onClick={() => setFixing(true)} className="min-h-8 text-sm font-semibold text-ink underline underline-offset-[3px]">
                  Fix email
                </button>
              )}
            </dd>
          )}
          {id && (
            <>
              <dt className="text-ink-3">USN</dt>
              <dd className="font-mono text-sm">{id.usn}</dd>
              <dt className="text-ink-3">Department</dt>
              <dd className="font-semibold">{id.department_name ?? id.department_code}</dd>
              <dt className="text-ink-3">Batch</dt>
              <dd className="font-mono text-sm">{id.batch_year}</dd>
            </>
          )}
        </dl>
        <div className="flex max-w-[620px] flex-col gap-1.5 rounded-[10px] bg-surface px-4 py-3.5 lg:bg-paper">
          <span className="text-[13px] font-semibold text-ink-2">Why it's here</span>
          <span className="text-[15px] leading-normal">{whyItsHere(request)}</span>
        </div>
        {id && (
          <p className="max-w-[620px] text-sm text-ink-3">
            Check the USN against the department records. Approving gives them the student role in {id.department_name}.
          </p>
        )}
      </article>

      {fixing ? null : mode === 'confirm' ? (
        <ActionBar
          tone="confirm"
          labelledBy="approve-h"
          onEscape={() => setMode('idle')}
          sentence={
            <span id="approve-h">
              <b className="font-semibold text-ink">
                Let {name} in as a {id?.department_code ?? ''} student{id ? `, batch ${id.batch_year}` : ''}?
              </b>{' '}
              They can sign in straight away.
              {failure && <span className="block font-semibold text-danger">{failure}</span>}
            </span>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary + ' flex-1 lg:flex-none'}>
            Cancel
          </button>
          {/* Focus lands here, so Enter confirms. */}
          <button type="button" autoFocus disabled={decide.isPending} onClick={() => submit('approve')} className={buttonStyles.primary}>
            {decide.isPending ? 'Approving…' : 'Approve'}
          </button>
        </ActionBar>
      ) : mode === 'reject' ? (
        <ActionBar
          stacked
          onEscape={() => setMode('idle')}
          sentence={
            <label className="flex flex-col gap-1.5">
              <span className="text-[13px] font-semibold text-ink-2">Reject {first}'s request? Note</span>
              <textarea
                autoFocus
                rows={2}
                value={note}
                placeholder="For example: USN not in our records for this batch."
                onChange={(e) => {
                  setNote(e.target.value)
                  setNoteError('')
                }}
                aria-invalid={noteError ? true : undefined}
                className={cn(
                  'field-sizing-content max-h-40 min-h-16 rounded-lg border bg-paper px-3 py-2.5 text-[15px] leading-normal text-ink outline-none',
                  noteError ? 'border-[1.5px] border-danger' : 'border-input focus-visible:border-rust',
                )}
              />
              <span className={cn('text-xs', noteError ? 'text-[13px] font-semibold text-danger' : 'text-ink-3')}>
                {noteError || failure || "It's kept with the decision. The request is removed, so a corrected class list can add them."}
              </span>
            </label>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary}>
            Cancel
          </button>
          <button type="button" disabled={decide.isPending} onClick={() => submit('reject')} className={buttonStyles.primary}>
            {decide.isPending ? 'Rejecting…' : 'Reject request'}
          </button>
        </ActionBar>
      ) : (
        <ActionBar
          start={
            <button type="button" onClick={() => setMode('reject')} className={buttonStyles.secondary}>
              Reject
            </button>
          }
          sentence={
            <>
              Approving lets <b className="font-semibold text-ink">{first}</b> sign in with {request.email}.
            </>
          }
        >
          <button type="button" onClick={() => setMode('confirm')} className={buttonStyles.primary}>
            Approve
          </button>
        </ActionBar>
      )}
    </section>
  )
}

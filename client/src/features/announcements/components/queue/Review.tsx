import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../../shared/api/types'
import { CategoryTag } from '../../../notices/components/CategoryTag'
import { audienceLabel } from '../../../notices/format'
import { usePreview, useReview } from '../../api'
import { toRules } from '../../audience'
import { buttonStyles } from '../../buttons'
import { hasExpired, waited } from '../../status'
import type { QueueItem } from '../../types'
import { ActionBar } from '../ActionBar'
import { Chip } from './Chip'

// Banner is the message shown after a review: done, or someone else acted.
export type Banner = { tone: 'done' | 'info'; text: string }

type Mode = 'idle' | 'confirm' | 'sendback'

// Review shows one waiting notice in full and the bar to decide on it.
export function Review({ item, focusTitle, onReviewed }: {
  item: QueueItem
  focusTitle: boolean
  onReviewed: (item: QueueItem, banner: Banner) => void
}) {
  const [mode, setMode] = useState<Mode>('idle')
  const [note, setNote] = useState('')
  const [noteError, setNoteError] = useState('')
  const [failure, setFailure] = useState('')
  const [compare, setCompare] = useState(false)
  const review = useReview()
  const preview = usePreview(item.category, toRules(item.audience))
  const titleRef = useRef<HTMLHeadingElement>(null)
  const wait = waited(item.submitted_at)
  const expired = hasExpired(item)
  const audience = audienceLabel(item.audience)
  const reach = preview.data ? `about ${preview.data.reach.toLocaleString('en-IN')} ${preview.data.reach === 1 ? 'person' : 'people'}` : ''
  const paragraphs = item.body.split(/\n\s*\n/)

  useEffect(() => {
    if (focusTitle) titleRef.current?.focus()
  }, [focusTitle])

  async function decide(decision: 'approve' | 'reject') {
    setFailure('')
    if (decision === 'reject' && !note.trim()) {
      setNoteError('Add a note so they know what to change.')
      return
    }
    try {
      await review.mutateAsync({ id: item.id, decision, note: note.trim() })
      onReviewed(item, {
        tone: 'done',
        text:
          decision === 'approve'
            ? `Published. "${item.title}" is live for ${audience}.`
            : `Sent back to ${item.publisher_name} with your note.`,
      })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 409) {
        onReviewed(item, { tone: 'info', text: `Someone else already reviewed "${item.title}". The queue has been refreshed.` })
      } else {
        setFailure("That didn't go through. Check your connection and try again.")
      }
    }
  }

  return (
    <section aria-label="Selected announcement" className="flex min-w-0 flex-col gap-4">
      <article className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
        <div className="flex flex-wrap items-center gap-2">
          <CategoryTag category={item.category} className="lg:text-xs" />
          {item.kind === 'edit' && <Chip>Edit to a live notice</Chip>}
          {expired && <Chip tone="warning">Expired while waiting</Chip>}
          <span className={cn('font-mono text-xs', wait.long ? 'text-warning' : 'text-ink-3')}>{wait.text}</span>
        </div>
        <h2
          ref={titleRef}
          tabIndex={-1}
          className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] outline-none lg:text-[32px] lg:tracking-[-0.5px]"
        >
          {item.title}
        </h2>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
          <dt className="text-ink-3">From</dt>
          <dd className="font-semibold">{item.publisher_name}</dd>
          <dt className="text-ink-3">To</dt>
          <dd>
            <b className="font-semibold">{audience}</b> {reach && <span className="font-mono text-xs text-ink-3">{reach}</span>}
          </dd>
          <dt className="text-ink-3">Expires</dt>
          <dd className="font-mono text-[13px]">
            {item.expires_at ? new Date(item.expires_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' }) : 'No end date'}
          </dd>
        </dl>

        {item.kind === 'edit' && item.live && (
          <div className="flex flex-col items-start justify-between gap-2.5 rounded-lg bg-paper px-3.5 py-2.5 text-sm text-ink-2 lg:flex-row lg:items-center">
            <span>An edit to a live notice. Readers keep seeing the current version until you approve.</span>
            <button
              type="button"
              aria-pressed={compare}
              onClick={() => setCompare((c) => !c)}
              className={cn(
                'min-h-9 shrink-0 rounded-[7px] border px-3 text-[13px] font-semibold',
                compare ? 'border-ink bg-ink text-paper' : 'border-input bg-surface text-ink',
              )}
            >
              Compare with the live version
            </button>
          </div>
        )}
        {expired && (
          <p className="rounded-lg bg-warning-soft px-3.5 py-2.5 text-sm text-warning-ink">
            <b className="font-semibold">Its expiry date passed while it waited.</b> It can't be published now. Send it back so the author can pick a new date.
          </p>
        )}

        {compare && item.live ? (
          <div className="flex flex-col gap-3">
            <Version label="Live now" title={item.live.title} body={item.live.body} muted />
            <Version label="Proposed" title={item.title} body={item.body} />
          </div>
        ) : (
          <div className="flex max-w-[640px] flex-col gap-3 font-serif text-[17px] leading-relaxed text-prose lg:text-lg">
            {paragraphs.map((text, i) => (
              <p key={i} className="whitespace-pre-line">
                {text}
              </p>
            ))}
          </div>
        )}
      </article>

      {mode === 'confirm' ? (
        <ActionBar
          tone="confirm"
          labelledBy="approve-h"
          onEscape={() => setMode('idle')}
          sentence={
            <span id="approve-h">
              <b className="font-semibold text-ink">Publish to {reach || audience} now?</b> {audience} will see it straight away.
              <span className="mt-0.5 hidden text-xs text-ink-3 lg:block">Enter to approve · Esc to cancel</span>
              {failure && <span className="block font-semibold text-danger">{failure}</span>}
            </span>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary + ' flex-1 lg:flex-none'}>
            Cancel
          </button>
          {/* Focus lands here, so Enter confirms. */}
          <button type="button" autoFocus disabled={review.isPending} onClick={() => decide('approve')} className={buttonStyles.primary}>
            {review.isPending ? 'Publishing…' : 'Approve and publish'}
          </button>
        </ActionBar>
      ) : mode === 'sendback' ? (
        <ActionBar
          stacked
          onEscape={() => setMode('idle')}
          sentence={
            <label className="flex flex-col gap-1.5">
              <span className="text-[13px] font-semibold text-ink-2">What should they change?</span>
              <textarea
                autoFocus
                rows={2}
                value={note}
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
                {noteError || failure || 'They see this note with their notice and can fix and resubmit it.'}
              </span>
            </label>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary}>
            Cancel
          </button>
          <button type="button" disabled={review.isPending} onClick={() => decide('reject')} className={buttonStyles.primary}>
            {review.isPending ? 'Sending…' : 'Send back'}
          </button>
        </ActionBar>
      ) : (
        <ActionBar
          start={
            <button type="button" onClick={() => setMode('sendback')} className={buttonStyles.secondary}>
              Send back
            </button>
          }
          sentence={
            expired ? (
              "It expired before review, so it can't be approved."
            ) : (
              <>
                Approving publishes it to <b className="font-semibold text-ink">{audience}</b> now.
              </>
            )
          }
        >
          <button type="button" disabled={expired} onClick={() => setMode('confirm')} className={buttonStyles.primary}>
            Approve
          </button>
        </ActionBar>
      )}
    </section>
  )
}

function Version({ label, title, body, muted }: { label: string; title: string; body: string; muted?: boolean }) {
  return (
    <div className={cn('flex flex-col gap-1.5 rounded-lg px-4 py-3.5', muted ? 'bg-paper text-ink-3' : 'border-[1.5px] border-ink')}>
      <span className="text-xs font-semibold tracking-wide uppercase">{label}</span>
      <p className={cn('font-serif text-lg font-medium', !muted && 'text-ink')}>{title}</p>
      <p className={cn('font-serif text-[17px] leading-relaxed whitespace-pre-line', !muted && 'text-prose')}>{body}</p>
    </div>
  )
}

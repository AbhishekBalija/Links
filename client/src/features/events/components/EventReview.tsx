import { useEffect, useId, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { useAuthStore } from '../../auth/store'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { Chip } from '../../announcements/components/queue/Chip'
import { waited } from '../../announcements/status'
import { useEventDecision, type EventDecision } from '../api'
import { approvalCopy } from '../review'
import { typeLabel, type EventQueueItem } from '../types'
import { DateTile } from './DateTile'
import { EventFacts, Fact } from './EventFacts'
import { PostTag } from './PostTags'

export type ReviewBanner = { tone: 'done' | 'info'; text: string }

type Mode = 'idle' | 'confirm' | 'sendback'

// EventReview shows one waiting event proposal in full, with the bar to
// decide on it. "Send back" asks whether the proposer may fix it (changes)
// or not (reject), and always needs a note.
export function EventReview({ item, focusTitle, onReviewed }: {
  item: EventQueueItem
  focusTitle: boolean
  onReviewed: (id: string, banner: ReviewBanner) => void
}) {
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  const copy = approvalCopy(item, roles)
  const decision = useEventDecision()
  const [mode, setMode] = useState<Mode>('idle')
  const [outcome, setOutcome] = useState<'request_changes' | 'reject'>('request_changes')
  const [note, setNote] = useState('')
  const [noteError, setNoteError] = useState('')
  const [failure, setFailure] = useState('')
  const titleRef = useRef<HTMLHeadingElement>(null)
  const radioName = useId()
  const wait = waited(item.submitted_at ?? item.updated_at)
  const firstName = item.proposer_name.split(' ')[0]

  useEffect(() => {
    if (focusTitle) titleRef.current?.focus()
  }, [focusTitle])

  async function decide(choice: EventDecision) {
    setFailure('')
    if (choice !== 'approve' && !note.trim()) {
      setNoteError(choice === 'reject' ? 'Add a note so they know why.' : 'Add a note so they know what to fix.')
      return
    }
    try {
      await decision.mutateAsync({ item, decision: choice, note: choice === 'approve' ? '' : note.trim() })
      const text =
        choice === 'approve'
          ? copy.publishes
            ? `Published. "${item.title}" is live for everyone invited.`
            : `Approved. "${item.title}" goes on to final approval.`
          : choice === 'reject'
            ? `Rejected. ${item.proposer_name} sees your note.`
            : `Sent back to ${item.proposer_name} with your note.`
      onReviewed(item.id, { tone: 'done', text })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 409) {
        onReviewed(item.id, { tone: 'info', text: `"${item.title}" was already decided, or it has started. The queue has been refreshed.` })
      } else {
        setFailure("That didn't go through. Check your connection and try again.")
      }
    }
  }

  return (
    <section aria-label="Selected event" className="flex min-w-0 flex-col gap-4">
      <article className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
        <div className="flex flex-wrap items-center gap-2">
          <PostTag kind="event" />
          <Chip>{typeLabel(item.event_type)}</Chip>
          {item.stage === 'final' && <Chip>Final approval</Chip>}
          <span className={cn('font-mono text-xs', wait.long ? 'text-warning' : 'text-ink-3')}>{wait.text}</span>
        </div>
        <div className="flex items-start gap-3.5 lg:gap-[18px]">
          <DateTile iso={item.starts_at} size="lg" />
          <h2
            ref={titleRef}
            tabIndex={-1}
            className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] outline-none lg:text-[32px] lg:tracking-[-0.5px]"
          >
            {item.title}
          </h2>
        </div>
        {copy.earlier && <p className="rounded-lg bg-well px-3.5 py-2.5 text-sm text-ink-2 lg:bg-paper">{copy.earlier}</p>}
        <EventFacts event={item}>
          <Fact label="Proposed by">
            {item.proposer_name}
            {item.faculty_mentor && ` · mentor ${item.faculty_mentor.full_name}`}
          </Fact>
        </EventFacts>
      </article>

      {mode === 'confirm' ? (
        <ActionBar
          tone="confirm"
          labelledBy="publish-h"
          onEscape={() => setMode('idle')}
          sentence={
            <span id="publish-h">
              <b className="font-semibold text-ink">Publish it now?</b> Everyone invited sees it straight away.
              <span className="mt-0.5 hidden text-xs text-ink-3 lg:block">Enter to approve · Esc to cancel</span>
              {failure && <span className="block font-semibold text-danger">{failure}</span>}
            </span>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary + ' flex-1 lg:flex-none'}>
            Cancel
          </button>
          {/* Focus lands here, so Enter confirms. */}
          <button type="button" autoFocus disabled={decision.isPending} onClick={() => decide('approve')} className={buttonStyles.primary}>
            {decision.isPending ? 'Publishing…' : copy.button}
          </button>
        </ActionBar>
      ) : mode === 'sendback' ? (
        <ActionBar
          stacked
          onEscape={() => setMode('idle')}
          sentence={
            <div className="flex flex-col gap-3 text-ink">
              <fieldset className="flex flex-col gap-1.5">
                <legend className="mb-1.5 text-[13px] font-semibold text-ink-2">Send back</legend>
                {(
                  [
                    ['request_changes', 'Ask for changes', `${firstName} can fix it and send it again.`],
                    ['reject', 'Reject', "Final. It can't be sent again."],
                  ] as const
                ).map(([value, label, hint]) => (
                  <label key={value} className={cn('flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm', outcome === value && 'bg-well')}>
                    <input type="radio" name={radioName} checked={outcome === value} onChange={() => setOutcome(value)} className="size-4 accent-ink" />
                    <span className="flex flex-col lg:flex-row lg:items-baseline lg:gap-2">
                      <span className="font-semibold">{label}</span>
                      <span className="text-[13px] text-ink-3">{hint}</span>
                    </span>
                  </label>
                ))}
              </fieldset>
              <label className="flex flex-col gap-1.5">
                <span className="text-[13px] font-semibold text-ink-2">
                  {outcome === 'reject' ? 'Why not?' : 'What should they change?'} <span className="font-normal text-ink-3">{firstName} sees this note.</span>
                </span>
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
                {(noteError || failure) && <span className="text-[13px] font-semibold text-danger">{noteError || failure}</span>}
              </label>
            </div>
          }
        >
          <button type="button" onClick={() => setMode('idle')} className={buttonStyles.secondary}>
            Back
          </button>
          <button
            type="button"
            disabled={decision.isPending}
            onClick={() => decide(outcome)}
            className={outcome === 'reject' ? buttonStyles.danger : buttonStyles.primary}
          >
            {decision.isPending ? 'Sending…' : outcome === 'reject' ? 'Reject' : 'Send back'}
          </button>
        </ActionBar>
      ) : (
        <ActionBar
          start={
            <button type="button" onClick={() => setMode('sendback')} className={buttonStyles.secondary}>
              Send back…
            </button>
          }
          sentence={
            <>
              {copy.sentence}
              {failure && <span className="block font-semibold text-danger">{failure}</span>}
            </>
          }
        >
          <button
            type="button"
            disabled={decision.isPending}
            onClick={() => (copy.publishes ? setMode('confirm') : decide('approve'))}
            className={buttonStyles.primary}
          >
            {decision.isPending ? 'Approving…' : copy.button}
          </button>
        </ActionBar>
      )}
    </section>
  )
}

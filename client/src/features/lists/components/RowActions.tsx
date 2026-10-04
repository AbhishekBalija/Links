import { CircleAlert, Ellipsis } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { useModal } from '../../../shared/ui/useModal'
import { buttonStyles } from '../../announcements/buttons'
import { useFixEmail, useRemoveRow } from '../api'
import { fixRefusal, whoLine } from '../logic'
import type { WaitingPerson } from '../types'

export type Outcome = { tone: 'ok' | 'late'; text: string }

// RowMenu is a row's "…" button. On desktop it opens a small menu; the
// chosen action then opens inline under the row (onChoose).
export function RowMenu({ person, onChoose }: { person: WaitingPerson; onChoose: (mode: 'fix' | 'remove') => void }) {
  const [open, setOpen] = useState(false)
  const menuId = useId()
  const wrapper = useRef<HTMLDivElement>(null)
  const button = useRef<HTMLButtonElement>(null)

  // A click outside or Escape closes the menu; Escape returns focus.
  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (!wrapper.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      setOpen(false)
      button.current?.focus()
    }
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const choose = (mode: 'fix' | 'remove') => {
    setOpen(false)
    onChoose(mode)
  }

  return (
    <div ref={wrapper} className="relative">
      <button
        ref={button}
        type="button"
        aria-label={`More for ${person.full_name}`}
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex size-10 items-center justify-center rounded-lg text-ink-2 hover:bg-paper"
      >
        <Ellipsis aria-hidden="true" className="size-[18px]" />
      </button>
      {open && (
        <div id={menuId} className="absolute top-[calc(100%+4px)] right-0 z-30 flex w-60 flex-col rounded-xl border border-line bg-surface p-1.5 shadow-[0_12px_32px_rgba(27,24,20,0.14)]">
          <button type="button" autoFocus onClick={() => choose('fix')} className="min-h-11 rounded-lg px-3 text-left text-[15px] font-semibold hover:bg-paper focus-visible:bg-paper">
            Fix email
          </button>
          <button type="button" onClick={() => choose('remove')} className="min-h-11 rounded-lg px-3 text-left text-[15px] font-semibold text-danger-ink hover:bg-paper focus-visible:bg-paper">
            Remove from the list
          </button>
        </div>
      )}
    </div>
  )
}

// RowSheet is the phone's version: the row's actions, then the email form,
// in a bottom sheet.
export function RowSheet({ person, onClose, onDone }: { person: WaitingPerson; onClose: () => void; onDone: (outcome: Outcome) => void }) {
  const [mode, setMode] = useState<'choose' | 'fix' | 'remove'>('choose')
  const ref = useModal(true)
  const id = useId()
  return (
    <dialog
      ref={ref}
      aria-labelledby={`${id}-h`}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45"
    >
      <div className="flex flex-col gap-2 px-4 pt-5 pb-7">
        <div id={`${id}-h`} className="mx-1 mb-1 flex flex-col gap-0.5">
          <span className="text-base font-semibold">{person.full_name}</span>
          <span className="font-mono text-xs break-all text-ink-3">
            {whoLine(person)} · {person.email}
          </span>
        </div>
        {mode === 'choose' && (
          <>
            <button type="button" onClick={() => setMode('fix')} className="min-h-[52px] rounded-[10px] px-4 text-left text-base font-semibold">
              Fix email
            </button>
            <button type="button" onClick={() => setMode('remove')} className="min-h-[52px] rounded-[10px] px-4 text-left text-base font-semibold text-danger-ink">
              Remove from the list
            </button>
            <button type="button" onClick={onClose} className={buttonStyles.secondary + ' mt-1.5 min-h-12'}>
              Cancel
            </button>
          </>
        )}
        {mode === 'fix' && <FixEmail person={person} stacked onCancel={onClose} onDone={onDone} />}
        {mode === 'remove' && <RemoveRow person={person} stacked onCancel={onClose} onDone={onDone} />}
      </div>
    </dialog>
  )
}

// FixEmail corrects the row's email: field and buttons on one line on
// desktop, stacked in the phone sheet.
export function FixEmail({ person, stacked = false, onCancel, onDone }: { person: Pick<WaitingPerson, 'user_id' | 'full_name' | 'email'>; stacked?: boolean; onCancel: () => void; onDone: (outcome: Outcome) => void }) {
  const fix = useFixEmail()
  const id = useId()
  const [email, setEmail] = useState(person.email)
  const [error, setError] = useState('')
  const first = person.full_name.trim().split(/\s+/)[0]

  async function save() {
    const value = email.trim()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
      setError('Enter a full email, like name@gmail.com.')
      return
    }
    if (value.toLowerCase() === person.email.toLowerCase()) {
      setError(`That is still ${first}'s old email. Correct it, then save.`)
      return
    }
    try {
      await fix.mutateAsync({ id: person.user_id, email: value })
      onDone({ tone: 'ok', text: `Saved. ${person.full_name} now signs in with ${value}.` })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 409) {
        const taken = fixRefusal(err.details, value)
        if (taken) return setError(taken)
        return onDone({ tone: 'late', text: `${person.full_name} signed in a moment ago, so nothing was changed. They're no longer on this list.` })
      }
      if (err instanceof ApiRequestError && err.status === 400) return setError(`That is already ${first}'s email.`)
      setError("That didn't go through. Check your connection and try again.")
    }
  }

  return (
    <div role="group" aria-labelledby={`${id}-l`} className={cn('flex flex-col gap-2', !stacked && 'rounded-[10px] border-[1.5px] border-ink bg-surface px-[18px] py-3.5')} onKeyDown={(e) => e.key === 'Escape' && onCancel()}>
      <label id={`${id}-l`} htmlFor={`${id}-email`} className="text-sm font-semibold">
        {person.full_name}'s email
      </label>
      <div className={cn('flex gap-2', stacked ? 'flex-col' : 'flex-wrap items-center')}>
        <input
          id={`${id}-email`}
          type="email"
          autoFocus
          value={email}
          onChange={(e) => {
            setEmail(e.target.value)
            setError('')
          }}
          onKeyDown={(e) => e.key === 'Enter' && save()}
          aria-invalid={error ? true : undefined}
          aria-describedby={`${id}-note`}
          className={cn('min-h-11 rounded-lg border bg-surface px-3.5 text-[15px]', stacked ? 'w-full min-h-12' : 'min-w-0 flex-[1_1_240px] lg:max-w-[460px]', error ? 'border-[1.5px] border-danger' : 'border-line')}
        />
        <button type="button" disabled={fix.isPending} onClick={save} className={cn(buttonStyles.primary, stacked && 'min-h-12')}>
          {fix.isPending ? 'Saving…' : 'Save email'}
        </button>
        <button type="button" onClick={onCancel} className={cn(buttonStyles.secondary, stacked && 'min-h-12')}>
          Cancel
        </button>
      </div>
      {error ? (
        <span id={`${id}-note`} className="flex gap-1.5 text-[13px] leading-[1.4] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {error}
        </span>
      ) : (
        <span id={`${id}-note`} className="text-[13px] leading-[1.4] break-all text-ink-3">
          Was {person.email}. {first} signs in with the new email; the old one stops working.
        </span>
      )}
    </div>
  )
}

// RemoveRow confirms removing the row, which frees its USN and email.
export function RemoveRow({ person, stacked = false, onCancel, onDone }: { person: WaitingPerson; stacked?: boolean; onCancel: () => void; onDone: (outcome: Outcome) => void }) {
  const remove = useRemoveRow()
  const id = useId()
  const [error, setError] = useState('')
  const what = person.kind === 'student' ? `Their USN ${person.usn} and email` : 'Their email'

  async function confirm() {
    try {
      await remove.mutateAsync(person.user_id)
      onDone({ tone: 'ok', text: person.kind === 'student' ? `Removed ${person.full_name}. USN ${person.usn} can be added again.` : `Removed ${person.full_name}.` })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 409) {
        return onDone({ tone: 'late', text: `${person.full_name} signed in a moment ago, so nothing was changed. They're no longer on this list.` })
      }
      setError("That didn't go through. Check your connection and try again.")
    }
  }

  return (
    <div role="alertdialog" aria-labelledby={`${id}-h`} aria-describedby={`${id}-d`} className={cn('flex flex-col gap-2.5', !stacked && 'rounded-[10px] border-[1.5px] border-ink bg-surface px-[18px] py-3.5')} onKeyDown={(e) => e.key === 'Escape' && onCancel()}>
      <span id={`${id}-h`} className="text-[15px] font-semibold">
        Remove {person.full_name} from the list?
      </span>
      <p id={`${id}-d`} className="text-sm leading-[1.45]">
        They haven't signed in, so nothing of theirs is lost. {what} can be added again, for example from a corrected class list. Kept in the audit log with your name.
      </p>
      {error && (
        <p role="alert" className="flex gap-1.5 text-[13px] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {error}
        </p>
      )}
      <div className={cn('flex gap-2', stacked && 'flex-col')}>
        <button type="button" autoFocus disabled={remove.isPending} onClick={confirm} className={cn(buttonStyles.danger, stacked && 'min-h-12')}>
          {remove.isPending ? 'Removing…' : 'Remove'}
        </button>
        <button type="button" onClick={onCancel} className={cn(buttonStyles.secondary, stacked && 'min-h-12')}>
          Keep
        </button>
      </div>
    </div>
  )
}

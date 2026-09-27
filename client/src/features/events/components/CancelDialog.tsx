import { useId, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { useModal } from '../../../shared/ui/useModal'
import { buttonStyles } from '../../announcements/buttons'
import { errorRing, inputClass } from '../../announcements/components/compose/styles'
import { useCancelEvent } from '../api'

type Props = {
  open: boolean
  eventId: string
  title: string
  // What cancelling does, in a sentence or two.
  children: ReactNode
  reasonLabel: string
  keepLabel: string
  confirmLabel: string
  onClose: () => void
  onCancelled: () => void
}

const MAX_REASON = 500

// CancelDialog asks for a reason before an event or a proposal is cancelled.
// The reason is required, since the people it affects see it. Escape or
// "Keep" backs out.
export function CancelDialog({ open, eventId, title, children, reasonLabel, keepLabel, confirmLabel, onClose, onCancelled }: Props) {
  const ref = useModal(open)
  const id = useId()
  const cancel = useCancelEvent()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  async function confirm() {
    if (!reason.trim()) {
      setError('Say why, in a sentence.')
      return
    }
    setError('')
    try {
      await cancel.mutateAsync({ id: eventId, reason: reason.trim() })
      onCancelled()
    } catch (err) {
      setError(err instanceof ApiRequestError && err.status === 409 ? "It can't be cancelled any more. Reload to see where it stands." : "It couldn't be cancelled. Try again.")
    }
  }

  return (
    <dialog
      ref={ref}
      aria-labelledby={`${id}-h`}
      aria-describedby={`${id}-p`}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45 lg:m-auto lg:w-[460px] lg:rounded-[14px]"
    >
      <div className="flex flex-col gap-4 px-4 pt-6 pb-7 lg:p-[26px]">
        <div className="flex flex-col gap-1.5 px-1 lg:px-0">
          <h2 id={`${id}-h`} className="font-serif text-2xl font-medium">
            {title}
          </h2>
          <div id={`${id}-p`} className="text-[15px] leading-normal text-ink-2">
            {children}
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor={`${id}-reason`} className="text-[13px] font-semibold text-ink-2">
            {reasonLabel}
          </label>
          <textarea
            id={`${id}-reason`}
            rows={3}
            maxLength={MAX_REASON}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value)
              setError('')
            }}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? `${id}-error` : undefined}
            className={cn(inputClass, 'resize-none text-[15px] leading-normal lg:bg-surface', error && errorRing)}
          />
          {error && (
            <p id={`${id}-error`} role="alert" className="text-[13px] font-semibold text-danger">
              {error}
            </p>
          )}
        </div>
        <div className="flex flex-col-reverse gap-2 lg:flex-row lg:justify-end">
          <button type="button" onClick={onClose} className={buttonStyles.secondary + ' min-h-12 lg:min-h-11'}>
            {keepLabel}
          </button>
          <button type="button" onClick={confirm} disabled={cancel.isPending} className={buttonStyles.danger + ' min-h-12 lg:min-h-11'}>
            {cancel.isPending ? 'Cancelling…' : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  )
}

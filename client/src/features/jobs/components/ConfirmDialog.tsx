import { useId, type ReactNode } from 'react'
import { buttonStyles } from '../../announcements/buttons'
import { useModal } from '../../../shared/ui/useModal'

type Props = {
  open: boolean
  title: string
  children: ReactNode
  cancelLabel: string
  confirmLabel: string
  busyLabel: string
  busy: boolean
  danger?: boolean
  error?: string | null
  onCancel: () => void
  onConfirm: () => void
}

// ConfirmDialog asks once before a step that can't be undone. It is a sheet
// from the bottom on phones and a centred dialog on desktop; Escape cancels.
export function ConfirmDialog({ open, title, children, cancelLabel, confirmLabel, busyLabel, busy, danger, error, onCancel, onConfirm }: Props) {
  const ref = useModal(open)
  const id = useId()
  return (
    <dialog
      ref={ref}
      aria-labelledby={`${id}-h`}
      aria-describedby={`${id}-p`}
      onCancel={(e) => {
        e.preventDefault()
        onCancel()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45 lg:m-auto lg:w-[460px] lg:rounded-[14px]"
    >
      <div className="flex flex-col gap-4 px-4 pt-6 pb-7 lg:p-[26px]">
        <div className="flex flex-col gap-1.5 px-1 lg:px-0">
          <h2 id={`${id}-h`} className="font-serif text-2xl font-medium lg:text-[26px]">
            {title}
          </h2>
          <div id={`${id}-p`} className="flex flex-col gap-3 text-[15px] leading-normal text-ink-2">
            {children}
          </div>
        </div>
        {error && (
          <p role="alert" className="text-[13px] font-semibold text-danger">
            {error}
          </p>
        )}
        <div className="flex flex-col-reverse gap-2 lg:flex-row lg:justify-end">
          <button type="button" onClick={onCancel} className={buttonStyles.secondary + ' min-h-12 lg:min-h-11'}>
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={busy}
            className={(danger ? buttonStyles.danger : buttonStyles.primary) + ' min-h-12 lg:min-h-11'}
          >
            {busy ? busyLabel : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  )
}

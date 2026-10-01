import { useModal } from '../../../shared/ui/useModal'
import { buttonStyles } from '../buttons'

type Props = {
  open: boolean
  canSaveDraft: boolean
  saving: boolean
  onSaveDraft: () => void
  onKeepEditing: () => void
  onDiscard: () => void
  // What is being written, for the dialog's wording.
  what?: 'announcement' | 'event' | 'opportunity'
}

const words = {
  announcement: { started: "You've started writing.", lost: 'Your changes to this announcement will be lost.' },
  event: { started: "You've started proposing an event.", lost: 'Your changes to this event will be lost.' },
  opportunity: { started: "You've started an opportunity.", lost: 'Your changes to this opportunity will be lost.' },
}

// LeaveDialog asks before unsaved writing is lost. The native <dialog> keeps
// focus inside it and closes on Escape, which counts as "keep editing".
export function LeaveDialog({ open, canSaveDraft, saving, onSaveDraft, onKeepEditing, onDiscard, what = 'announcement' }: Props) {
  const ref = useModal(open)
  const text = words[what]

  return (
    <dialog
      ref={ref}
      aria-labelledby="leave-h"
      aria-describedby="leave-p"
      onCancel={(e) => {
        e.preventDefault()
        onKeepEditing()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45 lg:m-auto lg:w-[440px] lg:rounded-xl"
    >
      <div className="flex flex-col gap-4 px-4 pt-6 pb-7 lg:p-6">
        <div className="flex flex-col gap-1.5 px-1">
          <h2 id="leave-h" className="font-serif text-2xl font-medium">
            {canSaveDraft ? 'Keep this as a draft?' : 'Leave without saving?'}
          </h2>
          <p id="leave-p" className="text-[15px] leading-normal text-ink-2">
            {canSaveDraft
              ? `${text.started} Save it to finish later, or discard it.`
              : text.lost}
          </p>
        </div>
        <div className="flex flex-col gap-2">
          {canSaveDraft && (
            <button type="button" onClick={onSaveDraft} disabled={saving} className={buttonStyles.primary + ' min-h-12'}>
              {saving ? 'Saving…' : 'Save draft'}
            </button>
          )}
          <button type="button" onClick={onKeepEditing} className={buttonStyles.secondary + ' min-h-12'}>
            Keep editing
          </button>
          <button
            type="button"
            onClick={onDiscard}
            className="min-h-12 rounded-lg text-[15px] font-semibold text-danger hover:bg-danger-soft"
          >
            Discard
          </button>
        </div>
      </div>
    </dialog>
  )
}

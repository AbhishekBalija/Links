import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useModal } from '../../../../shared/ui/useModal'
import { buttonStyles } from '../../buttons'

// AudienceSheet holds the audience picker on phones, as a bottom sheet.
export function AudienceSheet({ open, onClose, children }: { open: boolean; onClose: () => void; children: ReactNode }) {
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

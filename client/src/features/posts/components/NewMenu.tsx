import { Plus } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useModal } from '../../../shared/ui/useModal'
import { buttonStyles } from '../../announcements/buttons'

const choices = [
  { to: '/mine/new', title: 'An announcement', text: 'A notice people read: timings, deadlines, news.' },
  { to: '/mine/events/new', title: 'An event', text: 'Something people come to, with a date, a place and answers.' },
]

// NewMenu is My posts' one "New" button. It asks what is being posted: a
// small menu under the button on desktop, a bottom sheet on phones.
export function NewMenu() {
  const isDesktop = useIsDesktop()
  const [open, setOpen] = useState(false)
  const menuId = useId()
  const wrapper = useRef<HTMLDivElement>(null)

  // A click outside or Escape closes the desktop menu.
  useEffect(() => {
    if (!open || !isDesktop) return
    const onClick = (e: MouseEvent) => {
      if (!wrapper.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, isDesktop])

  return (
    <div ref={wrapper} className="relative flex-none">
      <button
        type="button"
        aria-haspopup={isDesktop ? 'true' : 'dialog'}
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((o) => !o)}
        className={buttonStyles.primary}
      >
        <Plus aria-hidden="true" className="size-4" />
        New
      </button>
      {isDesktop ? (
        open && (
          <div
            id={menuId}
            className="absolute top-[calc(100%+8px)] right-0 z-30 flex w-[340px] flex-col gap-1.5 rounded-xl border border-line bg-surface p-2 shadow-[0_12px_32px_rgba(27,24,20,0.14)]"
          >
            {choices.map((choice) => (
              <Link key={choice.to} to={choice.to} className="flex flex-col gap-0.5 rounded-lg px-3.5 py-3 text-ink hover:bg-paper hover:text-ink focus-visible:bg-paper">
                <span className="text-[15px] font-semibold">{choice.title}</span>
                <span className="text-[13px] text-ink-2">{choice.text}</span>
              </Link>
            ))}
          </div>
        )
      ) : (
        <NewSheet id={menuId} open={open} onClose={() => setOpen(false)} />
      )}
    </div>
  )
}

function NewSheet({ id, open, onClose }: { id: string; open: boolean; onClose: () => void }) {
  const ref = useModal(open)
  return (
    <dialog
      id={id}
      ref={ref}
      aria-labelledby={`${id}-h`}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45"
    >
      <div className="flex flex-col gap-2.5 px-4 pt-[22px] pb-7">
        <h2 id={`${id}-h`} className="mx-1 mb-1.5 font-serif text-2xl font-medium">
          What are you posting?
        </h2>
        {choices.map((choice) => (
          <Link key={choice.to} to={choice.to} className="flex flex-col gap-0.5 rounded-xl border border-line bg-surface p-4 text-ink hover:text-ink">
            <span className="text-base font-semibold">{choice.title}</span>
            <span className="text-sm text-ink-2">{choice.text}</span>
          </Link>
        ))}
        <button type="button" onClick={onClose} className={buttonStyles.secondary + ' mt-1 min-h-12'}>
          Cancel
        </button>
      </div>
    </dialog>
  )
}

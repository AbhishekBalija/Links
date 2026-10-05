import { useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { buttonStyles } from '../../announcements/buttons'
import { useModal } from '../../../shared/ui/useModal'
import { useWelcomed } from '../api'
import { welcomeLine, type NewRole } from '../welcome'

// Welcome tells a student what changed when they became a student
// coordinator: shown once, the first time they open LINKS after. A centred
// dialog on desktop, a bottom sheet on phones. Any way of closing it counts.
export function Welcome({ role }: { role: NewRole }) {
  const [open, setOpen] = useState(true)
  const ref = useModal(open)
  const welcomed = useWelcomed()
  const id = useId()
  const close = () => {
    setOpen(false)
    welcomed.mutate(role.id)
  }
  const department = role.department.name
  const points: [string, string][] = [
    ['Post announcements', `for ${department} students. Your HOD approves each one before anyone sees it.`],
    ['Propose events', 'for the department. Your HOD, then the principal, approve them.'],
    ['Everything else stays the same:', 'you are still a student, with your classes, events and jobs.'],
  ]
  const start = 'flex min-h-11 items-center text-sm font-semibold text-rust hover:text-rust-deep'

  return (
    <dialog
      ref={ref}
      aria-labelledby={`${id}-h`}
      onCancel={(e) => {
        e.preventDefault()
        close()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45 lg:m-auto lg:w-[560px] lg:rounded-2xl"
    >
      <div className="flex flex-col gap-3.5 px-[22px] pt-[22px] pb-7 lg:gap-4 lg:px-9 lg:py-8">
        <p className="font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase">A new role</p>
        <h2 id={`${id}-h`} className="font-serif text-[26px] leading-[1.1] font-medium tracking-[-0.5px] lg:text-[30px]">
          You're a student coordinator now
        </h2>
        <p className="text-[15px] leading-normal text-ink-2 lg:text-base">{welcomeLine(role)}</p>
        <ul className="flex flex-col gap-2.5">
          {points.map(([bold, rest]) => (
            <li key={bold} className="flex items-start gap-3 text-sm leading-normal lg:text-[15px]">
              <span aria-hidden="true" className="mt-[9px] size-1.5 flex-none rounded-full bg-rust" />
              <span>
                <b className="font-semibold">{bold}</b> {rest}
              </span>
            </li>
          ))}
        </ul>
        {/* Phones: Got it first, full width, with the two starts under it.
            Desktop: the starts on the left, Got it on the right. */}
        <div className="mt-1 flex flex-col gap-1 lg:flex-row-reverse lg:items-center lg:justify-between lg:gap-3">
          <button type="button" onClick={close} className={buttonStyles.primary + ' min-h-12 w-full lg:min-h-11 lg:w-auto'}>
            Got it
          </button>
          <span className="flex justify-center gap-5 lg:justify-start lg:gap-[18px]">
            <Link to="/mine/new" onClick={close} className={start}>
              Write an announcement
            </Link>
            <Link to="/mine/events/new" onClick={close} className={start}>
              Propose an event
            </Link>
          </span>
        </div>
      </div>
    </dialog>
  )
}

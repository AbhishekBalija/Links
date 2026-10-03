import { CircleAlert } from 'lucide-react'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { roleLabel } from '../../../app/shell/nav'
import { ApiRequestError } from '../../../shared/api/types'
import { Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { useEndingPreview, useEndRole } from '../api'
import { consequences, defaultOrganiser } from '../roles'
import type { Handover, PersonRef, RoleAssignment } from '../types'

type Props = {
  userId: string
  firstName: string
  viewerId: string
  assignment: RoleAssignment
  onClose: () => void
  onEnded: (message: string) => void
  // On phones the confirmation sits in a bottom sheet, which is its frame.
  inSheet?: boolean
}

// EndConfirm asks before a role ends and says first what happens to the
// person's unfinished work (ADR 0028): the server previews it, so the list
// is exactly what ending will do. A scheduled role is cancelled instead,
// which touches nothing.
export function EndConfirm({ userId, firstName, viewerId, assignment, onClose, onEnded, inSheet = false }: Props) {
  const scheduled = assignment.state === 'scheduled'
  const label = roleLabel(assignment.role)
  const preview = useEndingPreview(userId, assignment.id, !scheduled)
  const end = useEndRole(userId)
  const [picked, setPicked] = useState<string | null>(null)
  const [changing, setChanging] = useState(false)
  const [pickError, setPickError] = useState('')
  const [failure, setFailure] = useState('')
  const headingRef = useRef<HTMLHeadingElement>(null)
  const headingId = useId()

  // Focus starts on the question, so a screen reader reads it first.
  useEffect(() => headingRef.current?.focus(), [])

  const handover = preview.data
  const fallback = handover ? defaultOrganiser(handover) : null
  const needsPick = Boolean(handover && handover.moved_events.length > 0 && !fallback)
  const lastAdmin = preview.error instanceof ApiRequestError && preview.error.status === 409

  async function confirm() {
    setFailure('')
    if (needsPick && !picked) {
      const several = (handover?.moved_events.length ?? 0) > 1
      setPickError(`Pick who runs ${several ? 'these events' : 'this event'} before ending the role.`)
      return
    }
    try {
      await end.mutateAsync({ assignmentId: assignment.id, organiserId: picked })
      onEnded(scheduled ? `${firstName}'s scheduled ${label} role was cancelled.` : `${firstName}'s ${label} role has ended.`)
    } catch (err) {
      const field = err instanceof ApiRequestError ? err.details?.organiser_id : undefined
      if (typeof field === 'string') {
        setChanging(true)
        setPickError(field)
      } else if (err instanceof ApiRequestError && err.status === 409) {
        setFailure(err.message.charAt(0).toUpperCase() + err.message.slice(1) + '.')
      } else {
        setFailure("Couldn't end the role. Nothing changed. Check your connection and try again.")
      }
    }
  }

  let body: ReactNode
  if (scheduled) {
    body = (
      <p className="text-sm leading-normal">
        It was due to start on {new Date(assignment.starts_at).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })}. It never takes
        effect, and {firstName} keeps any other roles.
      </p>
    )
  } else if (preview.isPending) {
    body = (
      <div role="status" className="flex flex-col gap-2.5">
        <span className="text-sm text-ink-2">Checking {firstName}'s posts and events…</span>
        <Skeleton className="h-3 w-[88%]" />
        <Skeleton className="h-3 w-[64%]" />
      </div>
    )
  } else if (lastAdmin) {
    body = <Alert>{"This is the college's last admin role, so it can't end. Make someone else an admin first, then end this one."}</Alert>
  } else if (preview.isError) {
    body = (
      <Alert>
        <b className="font-semibold">Couldn't check what ending this role changes.</b> Nothing has changed.{' '}
        <button type="button" onClick={() => preview.refetch()} className="font-semibold text-danger-ink underline">
          Try again
        </button>
      </Alert>
    )
  } else {
    body = (
      <Consequences
        handover={handover!}
        firstName={firstName}
        assignment={assignment}
        viewerId={viewerId}
        fallback={fallback}
        changing={changing || needsPick}
        onChange={() => setChanging(true)}
        picked={picked}
        onPick={(id) => {
          setPicked(id || null)
          setPickError('')
        }}
        pickError={pickError}
      />
    )
  }

  const ready = scheduled || preview.isSuccess
  return (
    <div
      role="alertdialog"
      aria-labelledby={headingId}
      onKeyDown={(e) => {
        if (e.key === 'Escape') onClose()
      }}
      className={cn('flex flex-col gap-3 bg-surface', inSheet ? 'px-2 pt-1' : 'rounded-[10px] border-[1.5px] border-ink px-[18px] py-4')}
    >
      <h3 id={headingId} ref={headingRef} tabIndex={-1} className="text-[15px] font-semibold outline-none">
        {scheduled ? `Cancel ${firstName}'s scheduled ${label} role?` : `End ${firstName}'s ${label} role today?`}
      </h3>
      {failure && <Alert>{failure}</Alert>}
      {body}
      <div className="flex justify-end gap-2">
        <button type="button" onClick={onClose} className={buttonStyles.secondary + ' flex-1 lg:flex-none'}>
          {scheduled ? 'Keep it' : 'Cancel'}
        </button>
        {!lastAdmin && (
          <button
            type="button"
            disabled={!ready || end.isPending}
            onClick={confirm}
            className={cn(buttonStyles.primary, 'disabled:opacity-40')}
          >
            {end.isPending ? 'Ending…' : failure ? 'Try again' : scheduled ? 'Cancel role' : 'End role'}
          </button>
        )}
      </div>
    </div>
  )
}

function Consequences(props: {
  handover: Handover
  firstName: string
  assignment: RoleAssignment
  viewerId: string
  fallback: PersonRef | null
  changing: boolean
  onChange: () => void
  picked: string | null
  onPick: (id: string) => void
  pickError: string
}) {
  const { handover, firstName, assignment, viewerId, fallback, changing, picked, pickError } = props
  const items = consequences(handover, firstName)
  const moving = handover.moved_events.length
  const selectId = useId()

  return (
    <>
      {items.length === 0 ? (
        <p className="text-sm leading-normal">Nothing of {firstName}'s is waiting for approval or coming up.</p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {items.map((item, i) => (
            <li key={i} className="flex gap-2.5 text-sm leading-[1.45]">
              <span aria-hidden="true" className="text-rust">
                ·
              </span>
              <span>
                <b className="font-semibold">{item.title}</b> {item.text}
              </span>
            </li>
          ))}
        </ul>
      )}

      {moving > 0 &&
        (!changing && fallback ? (
          <div className="flex items-center justify-between gap-3 rounded-lg bg-paper py-1 pr-2 pl-3.5">
            <span className="text-sm">
              {moving === 1 ? 'It moves' : 'They move'} to{' '}
              <b className="font-semibold">{fallback.user_id === viewerId ? `you, ${fallback.full_name}` : fallback.full_name}</b>
            </span>
            <button type="button" onClick={props.onChange} className="min-h-11 px-1 text-sm font-semibold text-rust">
              Change
            </button>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            <label htmlFor={selectId} className="text-sm font-semibold">
              Takes over {firstName}'s upcoming {moving === 1 ? 'event' : 'events'}
            </label>
            <select
              id={selectId}
              value={picked ?? ''}
              onChange={(e) => props.onPick(e.target.value)}
              aria-invalid={pickError ? true : undefined}
              aria-describedby={`${selectId}-hint`}
              className={cn(
                'min-h-12 w-full rounded-lg border bg-surface px-3 text-[15px]',
                pickError ? 'border-[1.5px] border-danger' : 'border-line',
              )}
            >
              <option value="">{fallback ? `${fallback.full_name} (default)` : `Choose who runs ${moving === 1 ? 'it' : 'them'}`}</option>
              {handover.organiser_options
                .filter((person) => person.user_id !== fallback?.user_id)
                .map((person) => (
                  <option key={person.user_id} value={person.user_id}>
                    {person.user_id === viewerId ? `${person.full_name} (you)` : person.full_name}
                  </option>
                ))}
            </select>
            {pickError && (
              <span className="flex gap-1.5 text-[13px] font-medium text-danger-ink">
                <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
                {pickError}
              </span>
            )}
            <span id={`${selectId}-hint`} className="text-[13px] text-ink-3">
              People who could propose {moving === 1 ? 'this event' : 'these events'}.
            </span>
          </div>
        ))}

      <p className="text-[13px] leading-[1.45] text-ink-3">
        Everything {firstName} already published stays, under their name.{' '}
        {assignment.role === 'student_coordinator'
          ? 'They stay a student and see their old posts read-only.'
          : assignment.role === 'hod' && assignment.department
            ? `${assignment.department.name} has no HOD until someone is appointed.`
            : 'Their account and other roles stay.'}
      </p>
    </>
  )
}

function Alert({ children }: { children: ReactNode }) {
  return (
    <div role="alert" className="flex items-start gap-2.5 rounded-lg bg-danger-soft px-3.5 py-3 text-sm leading-[1.45] text-danger-ink">
      <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <span>{children}</span>
    </div>
  )
}

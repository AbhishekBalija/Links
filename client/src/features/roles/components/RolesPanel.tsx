import { Ellipsis, Plus } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { roleLabel } from '../../../app/shell/nav'
import { Skeleton } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { useModal } from '../../../shared/ui/useModal'
import { useAuthStore } from '../../auth/store'
import { buttonStyles } from '../../announcements/buttons'
import type { PublicProfile } from '../../people/types'
import { useRoles } from '../api'
import { assignmentLine, firstName, grantableRoles, mayManage, sortAssignments } from '../roles'
import type { RoleAssignment } from '../types'
import { EndConfirm } from './EndConfirm'
import { GrantForm } from './GrantForm'

// RolesPanel shows a person's Role assignments to whoever manages roles:
// the principal and admins for anyone, an HOD for their own Department's
// students (ADR 0027). For anyone else the server answers 404 and the panel
// stays hidden.
export function RolesPanel({ profile }: { profile: PublicProfile }) {
  const isDesktop = useIsDesktop()
  const viewer = useAuthStore((s) => s.user)
  const viewerRoles = viewer?.roles ?? []
  const grantable = grantableRoles(viewerRoles)
  const roles = useRoles(profile.user_id, grantable.length > 0)
  const [granting, setGranting] = useState(false)
  const [ending, setEnding] = useState<string | null>(null)
  const [done, setDone] = useState('')
  const grantButton = useRef<HTMLButtonElement>(null)
  const first = firstName(profile.full_name)

  if (grantable.length === 0 || roles.isError) return null

  const list = sortAssignments(roles.data ?? [])
  const isHODOnly = !viewerRoles.includes('admin') && !viewerRoles.includes('principal')
  // An HOD appoints a coordinator only for a current student who isn't one.
  const holds = (role: string) => list.some((a) => a.role === role && a.state !== 'ended')
  const canAppoint = isHODOnly ? holds('student') && !holds('student_coordinator') : true

  function finished(message: string) {
    setGranting(false)
    setEnding(null)
    setDone(message)
  }

  return (
    <section aria-labelledby="roles-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-[18px] py-4 lg:px-6 lg:py-[22px]">
      <div className="flex items-center justify-between gap-3">
        <h2 id="roles-h" className="font-serif text-[22px] leading-[1.2] font-medium">
          Roles
        </h2>
        {canAppoint && !granting && (
          <button
            ref={grantButton}
            type="button"
            onClick={() => {
              setDone('')
              setEnding(null)
              setGranting(true)
            }}
            className={buttonStyles.primary + ' flex-none'}
          >
            <Plus aria-hidden="true" className="size-4" />
            {isHODOnly ? 'Make student coordinator' : 'Grant a role'}
          </button>
        )}
      </div>

      <p role="status" className={cn('text-sm font-medium text-success-ink', !done && 'sr-only')}>
        {done}
      </p>

      {roles.isPending ? (
        <div className="flex flex-col gap-2" aria-label="Loading roles">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : list.length === 0 ? (
        <p className="text-sm text-ink-3">{first} has no roles yet.</p>
      ) : (
        <ul className="-mx-3.5 flex flex-col gap-1">
          {list.map((assignment) => (
            <RoleRow
              key={assignment.id}
              assignment={assignment}
              line={assignmentLine(assignment, { department: profile.department?.name, batch: profile.batch_year })}
              manageable={assignment.state !== 'ended' && mayManage(viewerRoles, assignment.role)}
              ending={ending === assignment.id}
              onEnd={() => {
                setDone('')
                setGranting(false)
                setEnding(assignment.id)
              }}
              confirm={
                <EndConfirm
                  userId={profile.user_id}
                  firstName={first}
                  viewerId={viewer?.user_id ?? ''}
                  assignment={assignment}
                  onClose={() => setEnding(null)}
                  onEnded={finished}
                  inSheet={!isDesktop}
                />
              }
            />
          ))}
        </ul>
      )}

      {granting && (
        <GrantForm
          userId={profile.user_id}
          firstName={first}
          roles={isHODOnly ? ['student_coordinator'] : grantable}
          department={profile.department}
          viewerIsPrincipal={viewerRoles.includes('principal') && !viewerRoles.includes('admin')}
          onClose={() => {
            setGranting(false)
            grantButton.current?.focus()
          }}
          onGranted={finished}
        />
      )}
    </section>
  )
}

function RoleRow({ assignment, line, manageable, ending, onEnd, confirm }: {
  assignment: RoleAssignment
  line: string
  manageable: boolean
  ending: boolean
  onEnd: () => void
  confirm: React.ReactNode
}) {
  const isDesktop = useIsDesktop()
  const [menuOpen, setMenuOpen] = useState(false)
  const moreButton = useRef<HTMLButtonElement>(null)
  const wrapper = useRef<HTMLLIElement>(null)
  const label = roleLabel(assignment.role)
  const sheet = useModal(ending && !isDesktop)
  const wasEnding = useRef(false)

  // A click outside or Escape closes the menu.
  useEffect(() => {
    if (!menuOpen) return
    const onClick = (e: MouseEvent) => !wrapper.current?.contains(e.target as Node) && setMenuOpen(false)
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMenuOpen(false)
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [menuOpen])

  // When the confirmation closes, focus goes back to the row's menu button.
  useEffect(() => {
    if (wasEnding.current && !ending) moreButton.current?.focus()
    wasEnding.current = ending
  }, [ending])

  return (
    <>
      <li ref={wrapper} className={cn('relative flex items-center gap-3 rounded-lg px-3.5 py-3', ending && 'bg-paper')}>
        <span className="flex grow flex-col gap-[3px]">
          <span className="flex items-center gap-2">
            <span className={cn('text-[15px] font-semibold', assignment.state === 'ended' ? 'text-ink-3' : 'text-ink')}>{label}</span>
            <StateTag state={assignment.state} />
          </span>
          <span className="text-[13px] text-ink-3">{line}</span>
        </span>
        {manageable && (
          <>
            <button
              ref={moreButton}
              type="button"
              aria-label={`More for ${label}`}
              aria-haspopup="menu"
              aria-expanded={menuOpen}
              onClick={() => setMenuOpen((o) => !o)}
              className="inline-flex size-11 items-center justify-center rounded-lg text-ink-2 hover:bg-paper"
            >
              <Ellipsis aria-hidden="true" className="size-[18px]" />
            </button>
            {menuOpen && (
              <div role="menu" className="absolute top-[calc(100%-6px)] right-2 z-30 flex w-56 flex-col rounded-xl border border-line bg-surface p-1.5 shadow-[0_12px_32px_rgba(27,24,20,0.14)]">
                <button
                  type="button"
                  role="menuitem"
                  autoFocus
                  onClick={() => {
                    setMenuOpen(false)
                    onEnd()
                  }}
                  className="flex min-h-11 items-center rounded-lg px-3 text-left text-[15px] font-semibold text-ink hover:bg-paper focus-visible:bg-paper"
                >
                  {assignment.state === 'scheduled' ? 'Cancel role' : 'End role'}
                </button>
              </div>
            )}
          </>
        )}
      </li>
      {ending &&
        (isDesktop ? (
          <li>{confirm}</li>
        ) : (
          <li>
            <dialog ref={sheet} aria-label={`End ${label} role`} className="mt-auto mb-0 w-full max-w-none bg-transparent p-0 backdrop:bg-[rgba(27,24,20,0.38)]">
              <div className="rounded-t-2xl bg-surface px-3 pt-3 pb-7">{confirm}</div>
            </dialog>
          </li>
        ))}
    </>
  )
}

function StateTag({ state }: { state: RoleAssignment['state'] }) {
  const text = { active: 'Active', scheduled: 'Scheduled', ended: 'Ended' }[state]
  return (
    <span className={cn('rounded px-[7px] py-0.5 text-[11px] font-semibold whitespace-nowrap', state === 'active' ? 'bg-success-soft text-success-ink' : 'bg-well text-ink-2')}>
      {text}
    </span>
  )
}

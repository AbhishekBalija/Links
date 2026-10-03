import { CircleAlert } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { roleLabel } from '../../../app/shell/nav'
import { ApiRequestError } from '../../../shared/api/types'
import { buttonStyles } from '../../announcements/buttons'
import { useDepartments } from '../../announcements/api'
import { useGrantRole } from '../api'
import { grantPayload, grantProblems, needsDepartment, todayInIndia, type GrantDraft } from '../roles'

type Props = {
  userId: string
  firstName: string
  // The roles the viewer may grant, in order. An HOD only has
  // student_coordinator, and gets the shorter "Make coordinator" form.
  roles: string[]
  // For the coordinator form: the student's own Department.
  department?: { code: string; name: string }
  viewerIsPrincipal: boolean
  onClose: () => void
  onGranted: (message: string) => void
}

// GrantForm adds a Role assignment, starting today or later, with an
// optional end. Mistakes are shown next to their field.
export function GrantForm({ userId, firstName, roles, department, viewerIsPrincipal, onClose, onGranted }: Props) {
  const coordinatorOnly = roles.length === 1 && roles[0] === 'student_coordinator'
  const today = todayInIndia()
  const departments = useDepartments()
  const grant = useGrantRole(userId)
  const [draft, setDraft] = useState<GrantDraft>({ role: roles[0] ?? '', departmentId: '', starts: today, ends: '' })
  const [problems, setProblems] = useState<Record<string, string>>({})
  const [failure, setFailure] = useState('')
  const headingRef = useRef<HTMLSpanElement>(null)
  const id = useId()

  useEffect(() => headingRef.current?.focus(), [])

  // The coordinator form's Department is the student's own.
  const ownDepartmentId = departments.data?.find((d) => d.code === department?.code)?.id ?? ''
  const departmentId = coordinatorOnly ? ownDepartmentId : draft.departmentId

  function update(change: Partial<GrantDraft>) {
    setDraft((d) => ({ ...d, ...change }))
    setProblems({})
    setFailure('')
  }

  async function submit() {
    const ready = { ...draft, departmentId }
    const found = grantProblems(ready, today)
    if (Object.keys(found).length > 0) {
      setProblems(found)
      return
    }
    try {
      await grant.mutateAsync(grantPayload(ready, today))
      onGranted(
        coordinatorOnly
          ? `${firstName} is a student coordinator${ready.starts === today ? ' now' : ` from ${ready.starts}`}.`
          : `${firstName} has the ${roleLabel(ready.role)} role${ready.starts === today ? ' now' : ` from ${ready.starts}`}.`,
      )
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 400 && err.details) {
        const fields = err.details as Record<string, string>
        setProblems({
          department: fields.scope_id ?? '',
          starts: fields.starts_at ?? '',
          ends: fields.ends_at ?? '',
          role: fields.role ?? '',
        })
        if (!fields.scope_id && !fields.starts_at && !fields.ends_at && !fields.role) setFailure(err.message)
      } else if (err instanceof ApiRequestError && (err.status === 409 || err.status === 403)) {
        setFailure(err.message.charAt(0).toUpperCase() + err.message.slice(1) + '.')
      } else {
        setFailure("That didn't go through. Check your connection and try again.")
      }
    }
  }

  return (
    <div
      role="group"
      aria-labelledby={`${id}-h`}
      onKeyDown={(e) => e.key === 'Escape' && onClose()}
      className="flex flex-col gap-3.5 rounded-[10px] bg-paper p-[18px]"
    >
      <span id={`${id}-h`} ref={headingRef} tabIndex={-1} className="text-[15px] font-semibold outline-none">
        {coordinatorOnly ? `Make ${firstName} a student coordinator` : 'Grant a role'}
      </span>
      {coordinatorOnly && (
        <p className="text-sm leading-normal text-ink-2">
          They can post announcements and propose events for {department?.name ?? 'your department'}. You approve each one before anyone sees it.
        </p>
      )}
      {failure && (
        <p role="alert" className="flex gap-1.5 text-[13px] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {failure}
        </p>
      )}

      {!coordinatorOnly && (
        <>
          <Field id={`${id}-role`} label="Role" error={problems.role} hint={roleHint(viewerIsPrincipal)}>
            <select
              id={`${id}-role`}
              value={draft.role}
              onChange={(e) => update({ role: e.target.value })}
              className={selectStyle(problems.role)}
            >
              {roles.map((role) => (
                <option key={role} value={role}>
                  {roleLabel(role)}
                </option>
              ))}
            </select>
          </Field>
          {needsDepartment(draft.role) && (
            <Field id={`${id}-dept`} label="Department" error={problems.department}>
              <select
                id={`${id}-dept`}
                value={draft.departmentId}
                onChange={(e) => update({ departmentId: e.target.value })}
                className={selectStyle(problems.department)}
              >
                <option value="">Choose a department</option>
                {departments.data?.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </Field>
          )}
        </>
      )}

      <div className="grid grid-cols-2 gap-3">
        <Field id={`${id}-start`} label="Starts" error={problems.starts} hint={draft.starts === today ? 'Today' : undefined}>
          <input
            id={`${id}-start`}
            type="date"
            min={today}
            value={draft.starts}
            onChange={(e) => update({ starts: e.target.value || today })}
            className={selectStyle(problems.starts) + ' px-3.5'}
          />
        </Field>
        <Field id={`${id}-end`} label="Ends (optional)" error={problems.ends} hint="Leave empty for no end">
          <input
            id={`${id}-end`}
            type="date"
            min={draft.starts}
            value={draft.ends}
            onChange={(e) => update({ ends: e.target.value })}
            className={selectStyle(problems.ends) + ' px-3.5'}
          />
        </Field>
      </div>

      <div className="flex justify-end gap-2">
        <button type="button" onClick={onClose} className={buttonStyles.secondary + ' flex-1 lg:flex-none'}>
          Cancel
        </button>
        <button type="button" disabled={grant.isPending} onClick={submit} className={buttonStyles.primary}>
          {grant.isPending ? 'Saving…' : coordinatorOnly ? 'Make coordinator' : 'Grant role'}
        </button>
      </div>
    </div>
  )
}

function roleHint(viewerIsPrincipal: boolean) {
  return viewerIsPrincipal ? "Faculty, HOD or placement officer. A department's HOD appoints its student coordinators." : undefined
}

function selectStyle(error?: string) {
  return cn('min-h-12 w-full rounded-lg border bg-surface px-3 text-[15px]', error ? 'border-[1.5px] border-danger' : 'border-line')
}

function Field({ id, label, error, hint, children }: { id: string; label: string; error?: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-semibold">
        {label}
      </label>
      {children}
      {error ? (
        <span className="flex gap-1.5 text-[13px] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {error}
        </span>
      ) : (
        hint && <span className="text-[13px] leading-[1.4] text-ink-3">{hint}</span>
      )}
    </div>
  )
}

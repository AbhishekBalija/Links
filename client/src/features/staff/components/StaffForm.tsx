import { Check, CircleAlert } from 'lucide-react'
import { useId, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { roleLabel } from '../../../app/shell/nav'
import { ApiRequestError } from '../../../shared/api/types'
import { buttonStyles } from '../../announcements/buttons'
import { useAddStaff } from '../api'
import {
  addedMessage,
  isCollegeWide,
  isGoogleOnly,
  nextWithoutHOD,
  refusal,
  staffPayload,
  staffProblems,
  type Department,
  type Refusal,
  type StaffDraft,
} from '../logic'

type Props = {
  title: string
  roles: string[]
  departments: Department[]
  // Department codes that already have an HOD, so adding HODs one after
  // another moves on to the next department without one.
  withHOD: Set<string>
  // An HOD adds faculty to their own department only; the form shows it
  // instead of asking.
  fixedDepartment?: Department
  initialRole?: string
  initialDepartmentId?: string
  requestsPath: string
}

// StaffForm adds one staff member. After each one it keeps the role (and,
// for HODs, moves to the next department without one) and clears the name
// and email, since day 0 means adding several people in a row.
export function StaffForm({ title, roles, departments, withHOD, fixedDepartment, initialRole, initialDepartmentId, requestsPath }: Props) {
  const id = useId()
  const nameRef = useRef<HTMLInputElement>(null)
  const add = useAddStaff()
  const [draft, setDraft] = useState<StaffDraft>({
    fullName: '',
    email: '',
    role: initialRole && roles.includes(initialRole) ? initialRole : (roles[0] ?? ''),
    departmentId: fixedDepartment?.id ?? initialDepartmentId ?? '',
  })
  const [problems, setProblems] = useState<Record<string, string>>({})
  const [refused, setRefused] = useState<Refusal | null>(null)
  const [failure, setFailure] = useState('')
  const [added, setAdded] = useState('')
  const filled = useRef(new Set(withHOD))

  const collegeWide = isCollegeWide(draft.role)
  const department = fixedDepartment ?? departments.find((d) => d.id === draft.departmentId)

  function update(change: Partial<StaffDraft>) {
    setDraft((d) => ({ ...d, ...change }))
    setProblems({})
    setRefused(null)
    setFailure('')
  }

  async function submit() {
    const found = staffProblems(draft)
    if (Object.keys(found).length > 0) {
      setProblems(found)
      setAdded('')
      return
    }
    try {
      const result = await add.mutateAsync(staffPayload(draft))
      setAdded(
        addedMessage({
          fullName: draft.fullName.trim(),
          email: draft.email.trim(),
          role: draft.role,
          departmentName: collegeWide ? '' : (department?.name ?? ''),
          emailed: result.emailed,
        }),
      )
      let nextDepartment = draft.departmentId
      if (draft.role === 'hod' && department) {
        filled.current.add(department.code)
        nextDepartment = nextWithoutHOD(departments, filled.current, department.id)
      }
      setDraft((d) => ({ ...d, fullName: '', email: '', departmentId: fixedDepartment?.id ?? nextDepartment }))
      nameRef.current?.focus()
    } catch (err) {
      setAdded('')
      const why = err instanceof ApiRequestError && err.status === 409 ? refusal(err.details, draft.email.trim(), department?.name ?? '', requestsPath) : null
      if (why) setRefused(why)
      else if (err instanceof ApiRequestError && err.status === 400) setFailure('Check the name, email and role, then try again.')
      else if (err instanceof ApiRequestError && err.status === 403) setFailure("You can't add this role.")
      else setFailure("That didn't go through. Check your connection and try again.")
    }
  }

  const emailError = problems.email || (refused?.field === 'email' ? refused : undefined)
  const departmentError = problems.department || (refused?.field === 'department' ? refused : undefined)

  return (
    <section aria-labelledby={`${id}-h`} className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-5 lg:px-7 lg:py-6">
      <h2 id={`${id}-h`} className="font-serif text-[22px] leading-tight font-medium">
        {title}
      </h2>
      {added && (
        <p role="status" className="flex items-start gap-2.5 rounded-[10px] bg-success-soft px-4 py-3.5 text-sm leading-[1.45] text-success-ink">
          <Check aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
          <span>{added}</span>
        </p>
      )}
      {failure && (
        <p role="alert" className="flex gap-1.5 text-[13px] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {failure}
        </p>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Field id={`${id}-name`} label="Full name" hint="As it should appear in LINKS." error={problems.full_name}>
          {(described) => (
            <input
              id={`${id}-name`}
              ref={nameRef}
              value={draft.fullName}
              onChange={(e) => update({ fullName: e.target.value })}
              autoComplete="off"
              {...described}
              className={inputStyle(problems.full_name)}
            />
          )}
        </Field>
        <Field id={`${id}-email`} label="Email" hint="They sign in with this email." error={emailError}>
          {(described) => (
            <input
              id={`${id}-email`}
              type="email"
              value={draft.email}
              onChange={(e) => update({ email: e.target.value })}
              placeholder="name@college.edu"
              autoComplete="off"
              {...described}
              className={inputStyle(emailError)}
            />
          )}
        </Field>

        {fixedDepartment ? (
          <div className="flex flex-col gap-1 rounded-lg bg-paper px-3.5 py-3 lg:col-span-2">
            <span className="text-[13px] text-ink-3">Role</span>
            <span className="text-[15px] font-semibold">
              {roleLabel(draft.role)}, {fixedDepartment.name}
            </span>
          </div>
        ) : (
          <>
            <div className={collegeWide ? 'lg:col-span-2' : undefined}>
              <Field
                id={`${id}-role`}
                label="Role"
                hint={isGoogleOnly(draft.role) ? "The principal and admins sign in with Google, so use an email that is a Google account: Gmail, or the college's Google Workspace." : undefined}
              >
                {(described) => (
                  <select id={`${id}-role`} value={draft.role} onChange={(e) => update({ role: e.target.value })} {...described} className={inputStyle()}>
                    {roles.map((role) => (
                      <option key={role} value={role}>
                        {roleLabel(role)}
                      </option>
                    ))}
                  </select>
                )}
              </Field>
            </div>
            {!collegeWide && (
              <Field id={`${id}-dept`} label="Department" hint="Faculty and HODs belong to a department." error={departmentError}>
                {(described) => (
                  <select
                    id={`${id}-dept`}
                    value={draft.departmentId}
                    onChange={(e) => update({ departmentId: e.target.value })}
                    {...described}
                    className={inputStyle(departmentError)}
                  >
                    <option value="">Choose a department</option>
                    {departments.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.name}
                      </option>
                    ))}
                  </select>
                )}
              </Field>
            )}
          </>
        )}
      </div>

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <span className="text-[13px] leading-[1.4] text-ink-3">We email them that they were added and how to sign in.</span>
        <button type="button" disabled={add.isPending} onClick={submit} className={buttonStyles.primary}>
          {add.isPending ? 'Adding…' : fixedDepartment ? 'Add faculty member' : 'Add staff member'}
        </button>
      </div>
    </section>
  )
}

function inputStyle(error?: unknown) {
  return cn('min-h-12 w-full rounded-lg border bg-surface px-3.5 text-[15px]', error ? 'border-[1.5px] border-danger' : 'border-line')
}

type Described = { 'aria-describedby'?: string; 'aria-invalid'?: true }

// Field labels a control and ties its hint or error to it, so a screen reader
// reads the reason with the field.
function Field({ id, label, hint, error, children }: { id: string; label: string; hint?: string; error?: string | Refusal; children: (described: Described) => ReactNode }) {
  const note = error ? `${id}-err` : hint ? `${id}-hint` : undefined
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-semibold">
        {label}
      </label>
      {children({ 'aria-describedby': note, ...(error ? { 'aria-invalid': true } : {}) })}
      {error ? (
        <span id={`${id}-err`} className="flex gap-1.5 text-[13px] leading-[1.4] font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          <span>
            {typeof error === 'string' ? error : error.text}
            {typeof error !== 'string' && error.link && (
              <>
                {' '}
                <Link to={error.link.to} className="font-semibold underline underline-offset-2">
                  {error.link.label}
                </Link>
              </>
            )}
          </span>
        </span>
      ) : (
        hint && (
          <span id={`${id}-hint`} className="text-[13px] leading-[1.4] text-ink-3">
            {hint}
          </span>
        )
      )}
    </div>
  )
}

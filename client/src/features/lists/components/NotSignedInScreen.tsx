import { Check, CircleAlert, Ellipsis } from 'lucide-react'
import { Fragment, useId, useState } from 'react'
import { cn } from '@/lib/utils'
import { ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { buttonStyles } from '../../announcements/buttons'
import { notSignedInEmails, useNotSignedIn } from '../api'
import { addedOn, whoLine } from '../logic'
import type { ListFilter, WaitingPerson } from '../types'
import { FixEmail, RemoveRow, RowMenu, RowSheet, type Outcome } from './RowActions'

type Props = {
  // The admin picks a Department; an HOD's list is their own only.
  departments?: { code: string; name: string }[]
  // Admins and HODs fix or remove rows; the principal only reads.
  canFix: boolean
}

// NotSignedInScreen is who a class list or a staff invite let in and who
// hasn't signed in yet. LINKS doesn't email them, so it offers their emails
// to copy into the college's own mail, and a way to fix a wrong row (#174).
export function NotSignedInScreen({ departments, canFix }: Props) {
  const isDesktop = useIsDesktop()
  const id = useId()
  const [filter, setFilter] = useState<ListFilter>({ department: '', kind: '' })
  const list = useNotSignedIn(filter)
  const [editing, setEditing] = useState<{ id: string; mode: 'fix' | 'remove' } | null>(null)
  const [sheetFor, setSheetFor] = useState<WaitingPerson | null>(null)
  const [outcome, setOutcome] = useState<Outcome | null>(null)
  const [copy, setCopy] = useState<{ state: 'idle' | 'copying' | 'copied' | 'manual' | 'failed'; emails: string[] }>({ state: 'idle', emails: [] })

  const people = list.data?.pages.flatMap((page) => page.data) ?? []
  const total = list.data?.pages[0]?.meta?.total ?? 0

  function change(next: Partial<ListFilter>) {
    setFilter((f) => ({ ...f, ...next }))
    setEditing(null)
    setOutcome(null)
    setCopy({ state: 'idle', emails: [] })
  }

  function done(result: Outcome) {
    setEditing(null)
    setSheetFor(null)
    setOutcome(result)
  }

  async function copyEmails() {
    setCopy({ state: 'copying', emails: [] })
    let emails: string[]
    try {
      emails = await notSignedInEmails(filter)
    } catch {
      return setCopy({ state: 'failed', emails: [] })
    }
    try {
      await navigator.clipboard.writeText(emails.join(', '))
      setCopy({ state: 'copied', emails })
    } catch {
      // Some browsers refuse the clipboard; the emails are shown to copy by hand.
      setCopy({ state: 'manual', emails })
    }
  }

  return (
    <section aria-label="Not signed in yet" className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-4 lg:p-7">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="grid grid-cols-2 gap-3 lg:flex">
          {departments && (
            <label className="flex flex-col gap-1.5 text-sm font-semibold">
              Department
              <select value={filter.department} onChange={(e) => change({ department: e.target.value })} className="min-h-12 rounded-lg border border-line bg-surface px-3 text-[15px] font-normal lg:w-[280px]">
                <option value="">All departments</option>
                {departments.map((d) => (
                  <option key={d.code} value={d.code}>
                    {d.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label className={cn('flex flex-col gap-1.5 text-sm font-semibold', !departments && 'col-span-2')}>
            Who
            <select value={filter.kind} onChange={(e) => change({ kind: e.target.value as ListFilter['kind'] })} className="min-h-12 rounded-lg border border-line bg-surface px-3 text-[15px] font-normal lg:w-[220px]">
              <option value="">Students and staff</option>
              <option value="student">Students</option>
              <option value="staff">Staff</option>
            </select>
          </label>
        </div>
        <button type="button" disabled={total === 0 || copy.state === 'copying'} onClick={copyEmails} className={cn(buttonStyles.secondary, 'w-full lg:w-auto')}>
          {copy.state === 'copied' ? (
            <>
              <Check aria-hidden="true" className="size-4" />
              Copied {copy.emails.length} {copy.emails.length === 1 ? 'email' : 'emails'}
            </>
          ) : copy.state === 'copying' ? (
            'Copying…'
          ) : (
            'Copy their emails'
          )}
        </button>
      </div>

      {copy.state === 'copied' && (
        <p role="status" className="text-sm font-medium text-success-ink">
          Copied. Paste them into the Bcc line of a mail from your college account.
        </p>
      )}
      {copy.state === 'manual' && (
        <div className="flex flex-col gap-1.5">
          <label htmlFor={`${id}-emails`} className="text-sm font-medium text-ink-2">
            Your browser didn't allow copying. Select these and copy them into the Bcc line:
          </label>
          <textarea id={`${id}-emails`} readOnly rows={3} value={copy.emails.join(', ')} onFocus={(e) => e.currentTarget.select()} className="rounded-lg border border-line bg-paper px-3 py-2 font-mono text-[13px]" />
        </div>
      )}
      {copy.state === 'failed' && (
        <p role="alert" className="flex gap-1.5 text-sm font-medium text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
          The emails could not be loaded. Try again.
        </p>
      )}
      {outcome?.tone === 'ok' && (
        <p role="status" className="text-sm font-medium text-success-ink">
          {outcome.text}
        </p>
      )}
      {outcome?.tone === 'late' && (
        <p role="alert" className="flex gap-2.5 rounded-[10px] bg-danger-soft px-4 py-3.5 text-sm leading-[1.45] text-danger-ink">
          <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
          {outcome.text}
        </p>
      )}

      <p className="text-sm text-ink-2">
        LINKS doesn't email them, so send a reminder yourself: copy the emails into your college mail. They sign in with Google or an email code, using the email shown here.
      </p>

      {list.isPending ? (
        <LoadingStatus label="Loading who hasn't signed in" />
      ) : list.isError ? (
        <ErrorState message="The list could not be loaded." onRetry={() => list.refetch()} />
      ) : people.length === 0 ? (
        <div className="flex flex-col gap-1 rounded-[10px] bg-paper px-4 py-5">
          <span className="font-serif text-xl font-medium">Nobody is waiting</span>
          <span className="text-sm text-ink-2">{filter.department || filter.kind ? 'Everyone here has signed in. Try other filters.' : 'Everyone added from class lists and staff invites has signed in.'}</span>
        </div>
      ) : isDesktop ? (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="text-left text-xs font-semibold text-ink-3">
              <th scope="col" className="px-3 pb-2">Name</th>
              <th scope="col" className="px-3 pb-2">Email</th>
              <th scope="col" className="px-3 pb-2">USN or role</th>
              <th scope="col" className="px-3 pb-2">Department</th>
              <th scope="col" className="px-3 pb-2">Added</th>
              <th scope="col" className="px-3 pb-2">By</th>
              {canFix && (
                <th scope="col" className="w-[52px]">
                  <span className="sr-only">Actions</span>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {people.map((person, i) => (
              <Fragment key={person.user_id}>
                <tr className={cn(i % 2 === 0 && 'bg-paper')}>
                  <td className="rounded-l-lg px-3 py-2.5 font-semibold">{person.full_name}</td>
                  <td className="px-3 py-2.5 break-all">{person.email}</td>
                  <td className="px-3 py-2.5 font-mono text-[13px]">{whoLine(person)}</td>
                  <td className="px-3 py-2.5">{person.department_code ? `${person.department_code}${person.batch_year ? ` · ${person.batch_year}` : ''}` : 'Whole college'}</td>
                  <td className="px-3 py-2.5 font-mono text-[13px] text-ink-2">{addedOn(person.added_at)}</td>
                  <td className={cn('px-3 py-2.5 text-ink-2', !canFix && 'rounded-r-lg')}>{person.added_by.full_name}</td>
                  {canFix && (
                    <td className="rounded-r-lg px-2 py-1 text-right">
                      <RowMenu person={person} onChoose={(mode) => setEditing({ id: person.user_id, mode })} />
                    </td>
                  )}
                </tr>
                {editing?.id === person.user_id && (
                  <tr>
                    <td colSpan={7} className="pt-1 pb-3">
                      {editing.mode === 'fix' ? <FixEmail person={person} onCancel={() => setEditing(null)} onDone={done} /> : <RemoveRow person={person} onCancel={() => setEditing(null)} onDone={done} />}
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      ) : (
        <ul className="flex flex-col">
          {people.map((person) => (
            <li key={person.user_id} className="flex items-center gap-2 border-b border-line py-3 last:border-b-0">
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="flex justify-between gap-2">
                  <span className="text-[15px] font-semibold">{person.full_name}</span>
                  <span className="font-mono text-xs text-ink-3">{addedOn(person.added_at)}</span>
                </span>
                <span className="text-[13px] break-all text-ink-2">{person.email}</span>
                <span className="font-mono text-xs text-ink-2">
                  {whoLine(person)}
                  {person.department_code && ` · ${person.department_code}`}
                </span>
              </span>
              {canFix && (
                <button type="button" aria-label={`More for ${person.full_name}`} onClick={() => setSheetFor(person)} className="-mr-2 inline-flex size-11 flex-none items-center justify-center rounded-lg text-ink-2">
                  <Ellipsis aria-hidden="true" className="size-[18px]" />
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      {people.length > 0 && (
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <span className="text-[13px] text-ink-3">
            Showing {people.length} of {total}. Oldest first.
          </span>
          {list.hasNextPage && (
            <button type="button" disabled={list.isFetchingNextPage} onClick={() => list.fetchNextPage()} className={buttonStyles.secondary}>
              {list.isFetchingNextPage ? 'Loading…' : 'Show more'}
            </button>
          )}
        </div>
      )}

      {sheetFor && <RowSheet person={sheetFor} onClose={() => setSheetFor(null)} onDone={done} />}
    </section>
  )
}

import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { buttonStyles } from '../../announcements/buttons'
import { ConfirmDialog } from '../../jobs/components/ConfirmDialog'
import { reportNotMe } from '../api'
import { useAuthStore } from '../store'
import { SignInHeading, SignInLayout } from '../components/SignInLayout'
import { roleLabel } from '../../../app/shell/nav'
import { greetingName } from '../signIn'

// FirstSignIn shows who a first sign-in signed in as, so a wrong class list
// row is reported ("Not you?") instead of used (spec #129).
export default function FirstSignIn() {
  const navigate = useNavigate()
  const first = useAuthStore((s) => s.firstSignIn)
  const confirm = useAuthStore((s) => s.confirmFirstSignIn)
  const signedOutAfterReport = useAuthStore((s) => s.signedOutAfterReport)
  const [asking, setAsking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!first) return <Navigate to="/" replace />

  const firstName = greetingName(first.full_name)
  const staffRole = first.roles.find((role) => role !== 'student')
  const rows: Array<[string, string, boolean]> = first.usn
    ? [
        ['Name', first.full_name, false],
        ['USN', first.usn, true],
        ['Department', first.department_name ?? '', false],
        ['Batch', String(first.batch_year ?? ''), true],
        ['Email', first.email, false],
      ]
    : [
        ['Name', first.full_name, false],
        ['Role', [staffRole ? roleLabel(staffRole) : '', first.department_name].filter(Boolean).join(', '), false],
        ['Email', first.email, false],
      ]

  async function report() {
    setBusy(true)
    setError(null)
    try {
      await reportNotMe()
      signedOutAfterReport()
      navigate('/login', { replace: true })
    } catch {
      setBusy(false)
      setError("Couldn't report it. Check your connection and try again.")
    }
  }

  return (
    <SignInLayout>
      <SignInHeading label="First sign-in">Welcome to Links, {firstName}</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        {first.usn ? 'Your college added you from its class list.' : 'Your college added you to LINKS as staff.'} Check these are yours before you go on.
      </p>
      <div className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-[22px] py-5">
        <p className="text-[13px] text-ink-3">You're signed in as</p>
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-[18px] gap-y-2 text-[15px]">
          {rows.map(([label, value, mono]) => (
            <div key={label} className="contents">
              <dt className="text-ink-3">{label}</dt>
              <dd className={mono ? 'font-mono break-all' : 'font-semibold break-words'}>{value}</dd>
            </div>
          ))}
        </dl>
      </div>
      <button
        type="button"
        onClick={() => {
          confirm()
          navigate('/', { replace: true })
        }}
        className={buttonStyles.primary + ' w-full'}
      >
        Yes, that's me
      </button>
      <button type="button" onClick={() => setAsking(true)} className="min-h-11 text-sm font-semibold text-rust">
        Not you? Report it
      </button>
      <ConfirmDialog
        open={asking}
        title="Report this account as not you?"
        cancelLabel="Cancel"
        confirmLabel="Report and sign out"
        busyLabel="Reporting…"
        busy={busy}
        error={error}
        onCancel={() => setAsking(false)}
        onConfirm={report}
      >
        <p>
          {first.usn
            ? "You'll be signed out. Nobody can sign into this account until your HOD or an admin checks the class list."
            : "You'll be signed out. Nobody can sign into this account until an admin checks who it was added for."}
        </p>
      </ConfirmDialog>
    </SignInLayout>
  )
}

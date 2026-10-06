import { useCallback, useEffect, useId, useState, type FormEvent } from 'react'
import { Check, Mail } from 'lucide-react'
import { buttonStyles } from '../../announcements/buttons'
import { ApiRequestError } from '../../../shared/api/types'
import { requestCode, sendAccessRequest, usePublicDepartments, verifyCode, googleSignIn } from '../api'
import { useAuthStore } from '../store'
import { codeState, isEmail, limitText, mailLinks, outcomeOf, readUSN, resendIn, type Outcome } from '../signIn'
import { SignInHeading, SignInLayout } from '../components/SignInLayout'
import { Notice } from '../components/Notice'
import { Steps } from '../components/Steps'
import { CodeBoxes } from '../components/CodeBoxes'
import { GoogleButton } from '../components/GoogleButton'
import { googleClientId } from '../google'

// Proven is an email on no list that has proved it is theirs, carried
// through choosing student or staff and the Access request.
type Proven = { email: string; fullName: string; requestToken: string; proof: 'google' | 'code' }

type Screen =
  | { name: 'start'; notice?: StartNotice; email?: string }
  | { name: 'code'; email: string; challengeId: string; sentAt: number; wrongTries: number; resent: boolean }
  | ({ name: 'fork' } & Proven)
  | ({ name: 'staff' } & Proven)
  | ({ name: 'request' } & Proven)
  | { name: 'sent'; sentAt: number; department: string; email: string }
  | { name: 'waiting' | 'declined' | 'suspended' }

type StartNotice = { tone: 'danger' | 'ok' | 'info' | 'warn'; text: string }

const field = 'block min-h-12 w-full rounded-lg border border-line bg-surface px-3.5 text-[15px] text-ink outline-none focus:border-rust focus:ring-2 focus:ring-rust/20'
const smallPrint = 'text-[13px] leading-normal text-ink-3'
const tryAgain = "Couldn't reach LINKS. Check your connection and try again."

// SignIn is every step of getting in (spec #129): Google or an email code,
// then, for an email on no class list, an Access request to the HOD.
export default function SignIn() {
  const leftBecause = useAuthStore((s) => s.leftBecause)
  const forgetLeftBecause = useAuthStore((s) => s.forgetLeftBecause)
  const [screen, setScreen] = useState<Screen>(() =>
    leftBecause === 'signed-out' ? { name: 'start', notice: { tone: 'ok', text: "You're signed out on this device." } } : { name: 'start' },
  )
  const [showReported, setShowReported] = useState(leftBecause === 'reported')

  useEffect(() => {
    if (leftBecause) forgetLeftBecause()
  }, [leftBecause, forgetLeftBecause])

  const restart = (notice?: StartNotice) => setScreen({ name: 'start', notice })

  // A failed Google or code sign-in decides the next screen the same way.
  const handleOutcome = useCallback((outcome: Outcome, proof: 'google' | 'code', email: string) => {
    switch (outcome.kind) {
      case 'not-on-list':
        // Students and staff get in differently, so they say which first (#200).
        setScreen({ name: 'fork', email: outcome.email || email, fullName: outcome.fullName, requestToken: outcome.requestToken, proof })
        return true
      case 'waiting':
      case 'declined':
      case 'suspended':
        setScreen({ name: outcome.kind })
        return true
      default:
        return false
    }
  }, [])

  if (showReported) {
    return (
      <SignInLayout>
        <SignInHeading label="Reported">Thanks. You're signed out.</SignInHeading>
        <p className="text-[15px] leading-normal text-ink-2">The account goes back to your department to check. Nobody can use it until they do.</p>
        <Notice tone="info">If you're a student here, tell your HOD which details were wrong. Once they fix your row, sign in again the same way.</Notice>
        <button type="button" onClick={() => setShowReported(false)} className="min-h-11 self-center text-sm font-semibold text-rust">
          Back to sign in
        </button>
      </SignInLayout>
    )
  }

  switch (screen.name) {
    case 'start':
      return <StartScreen key={screen.notice?.text ?? 'start'} initial={screen} onCode={(email, challengeId) => setScreen({ name: 'code', email, challengeId, sentAt: Date.now(), wrongTries: 0, resent: false })} onOutcome={handleOutcome} />
    case 'code':
      return <CodeScreen screen={screen} onChange={setScreen} onBack={() => restart()} onOutcome={handleOutcome} />
    case 'fork':
      return <ForkScreen email={screen.email} onStudent={() => setScreen({ ...screen, name: 'request' })} onStaff={() => setScreen({ ...screen, name: 'staff' })} onOtherEmail={() => restart()} />
    case 'staff':
      return <StaffScreen email={screen.email} onBack={() => setScreen({ ...screen, name: 'fork' })} onOtherEmail={() => restart()} />
    case 'request':
      return <RequestScreen screen={screen} onSent={(department) => setScreen({ name: 'sent', sentAt: Date.now(), department, email: screen.email })} onBack={() => setScreen({ ...screen, name: 'fork' })} onExpired={() => restart({ tone: 'danger', text: 'The proof of your email has expired. Sign in again to send the request.' })} />
    case 'sent':
      return <SentScreen sentAt={screen.sentAt} department={screen.department} email={screen.email} onBack={() => restart()} />
    default:
      return <StatusScreen name={screen.name} onBack={() => restart()} />
  }
}

function StartScreen({ initial, onCode, onOutcome }: {
  initial: { notice?: StartNotice; email?: string }
  onCode: (email: string, challengeId: string) => void
  onOutcome: (outcome: Outcome, proof: 'google' | 'code', email: string) => boolean
}) {
  const signedIn = useAuthStore((s) => s.signedIn)
  const [email, setEmail] = useState(initial.email ?? '')
  const [emailError, setEmailError] = useState<string | null>(null)
  const [notice, setNotice] = useState<StartNotice | undefined>(initial.notice)
  const [sending, setSending] = useState(false)
  const [googleKey, setGoogleKey] = useState(0)
  const [googleBroken, setGoogleBroken] = useState(false)
  const [limited, setLimited] = useState(false)
  const errorId = useId()

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!isEmail(email)) {
      setEmailError("That doesn't look like an email. Check for a missing .com or .in.")
      return
    }
    setEmailError(null)
    setNotice(undefined)
    setSending(true)
    try {
      const challenge = await requestCode(email)
      onCode(email.trim(), challenge.challenge_id)
    } catch (error) {
      setSending(false)
      const outcome = outcomeOf(error)
      if (outcome.kind === 'limit') {
        setLimited(true)
        // A whole campus on one network isn't the person's doing: a warning,
        // not an error (#202).
        setNotice({ tone: outcome.by === 'network' ? 'warn' : 'danger', text: limitText(outcome, { google: true }) })
      } else if (error instanceof ApiRequestError && error.status === 400) {
        setEmailError("That doesn't look like an email. Check for a missing .com or .in.")
      } else {
        setNotice({ tone: 'danger', text: tryAgain })
      }
    }
  }

  const onCredential = useCallback(async (credential: string) => {
    setNotice(undefined)
    try {
      await signedIn(await googleSignIn(credential))
    } catch (error) {
      const outcome = outcomeOf(error)
      if (!onOutcome(outcome, 'google', '')) {
        setNotice({ tone: 'danger', text: outcome.kind === 'error' ? tryAgain : "Google sign-in didn't finish. Try again, or get a code by email below." })
        setGoogleKey((k) => k + 1)
      }
    }
  }, [signedIn, onOutcome])

  const onGoogleError = useCallback(() => setGoogleBroken(true), [])
  const google = googleClientId !== '' && !googleBroken

  return (
    <SignInLayout>
      {notice && <Notice tone={notice.tone}>{notice.text}</Notice>}
      <SignInHeading>Sign in</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">Use the email your college has for you: a college address or a personal one.</p>
      {google && <GoogleButton key={googleKey} onCredential={onCredential} onError={onGoogleError} />}
      <form onSubmit={submit} noValidate className="flex flex-col gap-4">
        {google && <p className="text-center text-[13px] text-ink-3">or get a code by email</p>}
        <div className="flex flex-col gap-1.5">
          <label htmlFor="email" className="text-sm font-semibold">
            Email
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="you@example.com"
            value={email}
            aria-invalid={emailError ? true : undefined}
            aria-describedby={emailError ? errorId : undefined}
            onChange={(e) => {
              setEmail(e.target.value)
              setEmailError(null)
              setLimited(false)
            }}
            className={field + (emailError ? ' border-[1.5px] border-danger' : '')}
          />
          {emailError && (
            <p id={errorId} className="text-[13px] font-medium text-danger-ink">
              {emailError}
            </p>
          )}
        </div>
        <button type="submit" disabled={sending} className={(email.trim() && !limited ? buttonStyles.primary : buttonStyles.secondary) + ' w-full'}>
          {sending ? 'Sending…' : 'Email me a code'}
        </button>
      </form>
      <p className={smallPrint}>Not on a class list, or a new staff member? Sign in anyway and LINKS shows you what to do.</p>
      <p className={smallPrint}>The principal and admins sign in with Google.</p>
    </SignInLayout>
  )
}

function useNow(active: boolean) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!active) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [active])
  return now
}

function CodeScreen({ screen, onChange, onBack, onOutcome }: {
  screen: Extract<Screen, { name: 'code' }>
  onChange: (screen: Screen) => void
  onBack: () => void
  onOutcome: (outcome: Outcome, proof: 'google' | 'code', email: string) => boolean
}) {
  const signedIn = useAuthStore((s) => s.signedIn)
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const now = useNow(true)
  const dead = codeState(screen, now) === 'dead'
  const wait = resendIn(screen.sentAt, now)
  const errorId = useId()

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (code.length < 6) {
      setError('Type all 6 digits from the email.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await signedIn(await verifyCode({ challengeId: screen.challengeId, email: screen.email, code }))
    } catch (err) {
      setBusy(false)
      const outcome = outcomeOf(err)
      if (onOutcome(outcome, 'code', screen.email)) return
      if (outcome.kind === 'refused') {
        onChange({ ...screen, wrongTries: screen.wrongTries + 1 })
        setError("That code didn't work. Check it's from the newest email.")
      } else {
        setError(tryAgain)
      }
    }
  }

  async function resend() {
    setBusy(true)
    setError(null)
    try {
      const challenge = await requestCode(screen.email)
      setCode('')
      onChange({ ...screen, challengeId: challenge.challenge_id, sentAt: Date.now(), wrongTries: 0, resent: true })
    } catch (err) {
      const outcome = outcomeOf(err)
      setError(outcome.kind === 'limit' ? limitText(outcome, { google: false }) : tryAgain)
    } finally {
      setBusy(false)
    }
  }

  return (
    <SignInLayout back={{ label: 'Sign in', onClick: onBack }}>
      <SignInHeading>Check your email</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        If this email can use LINKS, a code is on its way to <b className="font-semibold text-ink">{screen.email}</b>. It works for 10 minutes.{' '}
        <button type="button" onClick={onBack} className="font-semibold text-rust">
          Wrong email?
        </button>
      </p>
      {dead ? (
        <>
          <Notice tone="warn">
            <b className="font-semibold">This code has stopped working.</b> Codes last 10 minutes and allow 5 tries. Send a new one.
          </Notice>
          {error && <Notice tone="danger">{error}</Notice>}
          <button type="button" onClick={resend} disabled={busy} className={buttonStyles.primary + ' w-full'}>
            {busy ? 'Sending…' : 'Send a new code'}
          </button>
        </>
      ) : (
        <form onSubmit={submit} noValidate className="flex flex-col gap-4">
          {screen.resent && !error && <Notice tone="ok">New code sent. Use the newest email; older codes won't work here.</Notice>}
          <CodeBoxes value={code} onChange={(next) => { setCode(next); setError(null) }} invalid={error !== null} describedBy={error ? errorId : undefined} />
          {error && (
            <p id={errorId} role="alert" className="-mt-2 text-sm font-medium text-danger-ink">
              {error}
            </p>
          )}
          <button type="submit" disabled={busy} className={buttonStyles.primary + ' w-full'}>
            {busy ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      )}
      <MailLinks email={screen.email} />
      {!dead &&
        (wait ? (
          <p className={smallPrint}>
            Nothing yet? Check spam · new code in <span className="font-mono text-ink-2">{wait}</span>
          </p>
        ) : (
          <>
            <NoCodeYet email={screen.email} />
            <button type="button" onClick={resend} disabled={busy} className={buttonStyles.secondary + ' w-full'}>
              {busy ? 'Sending…' : 'Send a new code'}
            </button>
          </>
        ))}
    </SignInLayout>
  )
}

const mailHref = { gmail: 'https://mail.google.com/', outlook: 'https://outlook.live.com/mail/' }
const mailName = { gmail: 'Open Gmail', outlook: 'Open Outlook' }

function MailLinks({ email }: { email: string }) {
  return (
    <div className="flex flex-wrap gap-x-3.5">
      {mailLinks(email).map((link) => (
        <a key={link} href={mailHref[link]} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-10 items-center gap-1.5 text-sm font-semibold text-rust">
          <Mail aria-hidden="true" className="size-4" />
          {mailName[link]}
        </a>
      ))}
    </div>
  )
}

const choice = 'flex w-full flex-col gap-1 rounded-xl border border-line bg-surface px-5 py-[18px] text-left hover:border-ink-3'

// ForkScreen asks an email on no list who they are: a student sends their
// USN to their HOD; staff are added by the college and send nothing here.
function ForkScreen({ email, onStudent, onStaff, onOtherEmail }: { email: string; onStudent: () => void; onStaff: () => void; onOtherEmail: () => void }) {
  return (
    <SignInLayout>
      <SignInHeading>This email isn't on LINKS yet</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        <b className="font-semibold text-ink">{email}</b> isn't on a class list or the staff list. Which are you?
      </p>
      <button type="button" onClick={onStudent} className={choice}>
        <span className="text-base font-semibold">I'm a student here</span>
        <span className="text-sm text-ink-2">Send your USN to your HOD. They let you in.</span>
      </button>
      <button type="button" onClick={onStaff} className={choice}>
        <span className="text-base font-semibold">I work here</span>
        <span className="text-sm text-ink-2">Faculty, HODs and office staff are added by the college.</span>
      </button>
      <p className={smallPrint}>
        Signed in with the wrong account?{' '}
        <button type="button" onClick={onOtherEmail} className="font-semibold text-rust">
          Use a different email
        </button>
      </p>
    </SignInLayout>
  )
}

// StaffScreen tells staff on no list who adds them, with the email to send.
function StaffScreen({ email, onBack, onOtherEmail }: { email: string; onBack: () => void; onOtherEmail: () => void }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(email)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  return (
    <SignInLayout back={{ label: 'Student or staff', onClick: onBack }}>
      <SignInHeading>Ask to be added</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        Staff don't send a request here. Your HOD adds faculty; the college office adds HODs, the principal and office staff.
      </p>
      <div className="flex flex-col gap-2.5 rounded-xl border border-line bg-surface px-[18px] py-4">
        <span className="text-sm text-ink-2">Send them the email you'll sign in with:</span>
        <div className="flex items-center justify-between gap-2.5">
          <span className="min-w-0 text-[15px] font-semibold break-all">{email}</span>
          <button type="button" onClick={copy} className={buttonStyles.secondary}>
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
        <span role="status" className="sr-only">
          {copied ? 'Email copied' : ''}
        </span>
      </div>
      <p className="text-[15px] leading-normal text-ink-2">Once they've added you, sign in again the same way. LINKS also emails you when it's done.</p>
      <p className={smallPrint}>
        Rather use your college email?{' '}
        <button type="button" onClick={onOtherEmail} className="font-semibold text-rust">
          Use a different email
        </button>
      </p>
    </SignInLayout>
  )
}

function RequestScreen({ screen, onSent, onBack, onExpired }: {
  screen: Extract<Screen, { name: 'request' }>
  onSent: (department: string) => void
  onBack: () => void
  onExpired: () => void
}) {
  const departments = usePublicDepartments()
  const [fullName, setFullName] = useState(screen.fullName)
  const [usn, setUSN] = useState('')
  const [touched, setTouched] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)
  const [nameError, setNameError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const usnErrorId = useId()

  const reading = readUSN(usn, departments.data ?? [])
  let usnError: string | null = serverError
  if (!usnError && touched && !reading.ok) {
    if (reading.reason === 'unknown-department') usnError = `No department has the code ${reading.code}. Check the letters in the middle of your USN.`
    if (reading.reason === 'format') usnError = 'A USN looks like 4MN23CS042: 10 letters and digits.'
    if (reading.reason === 'empty') usnError = 'Type your USN.'
  }
  const department = reading.ok ? reading.parts.department : null

  async function submit(e: FormEvent) {
    e.preventDefault()
    setTouched(true)
    if (!fullName.trim()) {
      setNameError('Type your name as your college has it.')
      return
    }
    if (!reading.ok) return
    setBusy(true)
    try {
      await sendAccessRequest({ requestToken: screen.requestToken, usn: reading.usn, fullName })
      onSent(reading.parts.department)
    } catch (err) {
      setBusy(false)
      if (err instanceof ApiRequestError && err.status === 401) return onExpired()
      if (err instanceof ApiRequestError && err.status === 409) {
        setServerError(`${reading.usn} already has an account. Sign in with the email your college has for you, or ask your department office.`)
      } else if (err instanceof ApiRequestError && err.status === 400) {
        setServerError(capitalise(err.message) + '.')
      } else {
        setServerError(tryAgain)
      }
    }
  }

  return (
    <SignInLayout back={{ label: 'Student or staff', onClick: onBack }}>
      <SignInHeading>You're not on a class list yet</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        <b className="font-semibold text-ink">{screen.email}</b> isn't on any class list yet. Send a request and your HOD can let you in.
      </p>
      <form onSubmit={submit} noValidate className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <label htmlFor="full-name" className="text-sm font-semibold">
            Your name
          </label>
          <input
            id="full-name"
            autoComplete="name"
            value={fullName}
            aria-invalid={nameError ? true : undefined}
            onChange={(e) => { setFullName(e.target.value); setNameError(null) }}
            className={field + (nameError ? ' border-[1.5px] border-danger' : '')}
          />
          <p className={'text-[13px] ' + (nameError ? 'font-medium text-danger-ink' : 'text-ink-3')}>{nameError ?? 'As your college has it.'}</p>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-semibold">Email</span>
          <p className="flex min-h-11 items-center rounded-lg bg-well px-3.5 text-[15px]">{screen.email}</p>
          <p className="flex items-center gap-1.5 text-[13px] font-medium text-success-ink">
            <Check aria-hidden="true" className="size-4" />
            {screen.proof === 'google' ? 'Proved by Google' : 'Proved with an email code'}
          </p>
        </div>
        <div className="flex flex-col gap-1.5">
          <label htmlFor="usn" className="text-sm font-semibold">
            USN
          </label>
          <input
            id="usn"
            autoCapitalize="characters"
            placeholder="4MN23CS042"
            value={usn}
            aria-invalid={usnError ? true : undefined}
            aria-describedby={usnError ? usnErrorId : undefined}
            onChange={(e) => { setUSN(e.target.value); setServerError(null) }}
            onBlur={() => usn && setTouched(true)}
            className={field + ' font-mono tracking-[0.5px]' + (usnError ? ' border-[1.5px] border-danger' : '')}
          />
          {reading.ok && !serverError && (
            <div aria-live="polite" className="flex items-center justify-between gap-2 rounded-lg bg-success-soft px-3.5 py-2.5">
              <span className="flex gap-3.5">
                {([['college', reading.parts.college], ['batch', reading.parts.batch], ['dept', reading.parts.department], ['roll', reading.parts.roll]] as const).map(([label, part]) => (
                  <span key={label} className="flex flex-col items-center gap-0.5">
                    <span className="font-mono text-[15px] font-medium">{part}</span>
                    <span className="text-[11px] text-ink-3">{label}</span>
                  </span>
                ))}
              </span>
              <span className="flex items-center gap-1.5 text-right text-[13px] font-semibold text-success-ink">
                <Check aria-hidden="true" className="size-4 shrink-0" />
                {reading.departmentName}, {reading.batchYear}
              </span>
            </div>
          )}
          {usnError ? (
            <p id={usnErrorId} className="text-[13px] font-medium text-danger-ink">
              {usnError}
            </p>
          ) : (
            <p className="text-[13px] text-ink-3">Your Department and Batch come from it.</p>
          )}
        </div>
        <button type="submit" disabled={busy} className={buttonStyles.primary + ' w-full'}>
          {busy ? 'Sending…' : department ? `Send request to the ${department} HOD` : 'Send request to your HOD'}
        </button>
      </form>
      <p className={smallPrint}>
        Your HOD sees your name, email and USN.{' '}
        <button type="button" onClick={onBack} className="font-semibold text-rust">
          Use a different email
        </button>
      </p>
    </SignInLayout>
  )
}

const timeFormat = new Intl.DateTimeFormat('en-GB', { hour: 'numeric', minute: '2-digit', hour12: true })

function SentScreen({ sentAt, department, email, onBack }: { sentAt: number; department: string; email: string; onBack: () => void }) {
  return (
    <SignInLayout>
      <SignInHeading label="Request sent">Your request is with the {department} HOD</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">
        We'll email you at <b className="font-semibold text-ink break-all">{email}</b> when they decide.
      </p>
      <Steps
        steps={[
          { label: 'Request sent', detail: `Today, ${timeFormat.format(sentAt)}`, state: 'done' },
          { label: 'Your HOD decides', detail: "They'll check your USN against the college records.", state: 'now' },
          { label: 'You sign in', detail: 'The same way as today. Once approved, you go straight in.', state: 'later' },
        ]}
      />
      <p className="text-sm text-ink-3">There's nothing else to do now. You can close this page; the email will tell you.</p>
      <button type="button" onClick={onBack} className="min-h-11 self-center text-sm font-semibold text-rust">
        Back to sign in
      </button>
    </SignInLayout>
  )
}

function StatusScreen({ name, onBack }: { name: 'waiting' | 'declined' | 'suspended'; onBack: () => void }) {
  return (
    <SignInLayout>
      {name === 'waiting' && (
        <>
          <SignInHeading label="Waiting for approval">Your request is still waiting</SignInHeading>
          <p className="text-[15px] leading-normal text-ink-2">Your HOD hasn't decided yet.</p>
          <Steps
            steps={[
              { label: 'Request sent', detail: 'Your name, email and USN reached your HOD.', state: 'done' },
              { label: 'Your HOD decides', detail: "If it's taking long, ask your department office.", state: 'now' },
              { label: 'You sign in', detail: 'The same way as today.', state: 'later' },
            ]}
          />
        </>
      )}
      {name === 'declined' && (
        <>
          <SignInHeading label="Request not approved">Your request wasn't approved</SignInHeading>
          <p className="text-[15px] leading-normal text-ink-2">The request from this email was turned down, so it can't use LINKS.</p>
          <Notice tone="info">If you study or work here, ask your department office to add you to the class list. Then sign in again.</Notice>
        </>
      )}
      {name === 'suspended' && (
        <>
          <SignInHeading label="Account paused">This account can't sign in right now</SignInHeading>
          <Notice tone="info">Your college has paused this account. Ask your department office or an admin if you think it's a mistake.</Notice>
        </>
      )}
      <button type="button" onClick={onBack} className="min-h-11 self-center text-sm font-semibold text-rust">
        Back to sign in
      </button>
    </SignInLayout>
  )
}

// NoCodeYet lists every reason a code might not come, once the resend wait
// is over (#202). It reads the same for every email, so it never says
// whether this one has an account.
function NoCodeYet({ email }: { email: string }) {
  const id = useId()
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2 rounded-[10px] border border-line bg-surface px-4 py-3.5">
      <h2 id={id} className="text-sm font-semibold">
        No code yet?
      </h2>
      <ul className="flex list-disc flex-col gap-1.5 pl-[18px] text-[13px] leading-normal text-ink-2">
        <li>
          Check spam, and that <b className="font-semibold text-ink">{email}</b> is spelled right.
        </li>
        <li>The principal and admins sign in with Google, so they get no code.</li>
        <li>An email whose request was declined, or an account that is paused, gets no code. Ask your department office.</li>
        <li>New to LINKS and not on a class list? On busy days codes for new emails can run out. Try again tomorrow, or ask your department office to add you.</li>
      </ul>
    </section>
  )
}

function capitalise(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

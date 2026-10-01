import { ExternalLink } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { buttonStyles } from '../../announcements/buttons'
import { dayAndDate, dayMonth } from '../../events/standing'
import { Tag } from '../../events/components/Tags'
import { useApplication } from '../api'
import { daysLeft } from '../format'
import { jobAction, studentStatus } from '../standing'
import type { Opportunity } from '../types'
import { ConfirmDialog } from './ConfirmDialog'

type Ask = 'apply' | 'withdraw' | 'marked' | null

// The server's refusals, in words a student can act on.
function refusal(error: unknown, action: 'apply' | 'withdraw'): string {
  if (error instanceof ApiRequestError && error.status === 409) {
    return action === 'apply'
      ? "It can't take your application any more: it may have just closed, or you applied before. The page now shows where you stand."
      : 'It can no longer be withdrawn: the placement office has already moved it. The page now shows where you stand.'
  }
  return action === 'apply' ? 'Your application could not be sent. Try again.' : 'It could not be withdrawn. Try again.'
}

// JobAction is the one place to act on a job: a panel beside it on desktop,
// a bar at the bottom of the screen on phones. What it offers follows
// jobAction: apply, mark an outside application, or where it stands now.
export function JobAction({ job }: { job: Opportunity }) {
  const application = useApplication(job.id)
  const [ask, setAsk] = useState<Ask>(null)
  // After "Open company site", coming back to LINKS asks whether they applied.
  const [askOnReturn, setAskOnReturn] = useState(false)
  const action = jobAction(job)

  useEffect(() => {
    if (!askOnReturn) return
    function back() {
      if (document.visibilityState === 'visible') {
        setAskOnReturn(false)
        setAsk('marked')
      }
    }
    window.addEventListener('focus', back)
    document.addEventListener('visibilitychange', back)
    return () => {
      window.removeEventListener('focus', back)
      document.removeEventListener('visibilitychange', back)
    }
  }, [askOnReturn])

  async function run(kind: 'apply' | 'withdraw') {
    try {
      await application.mutateAsync(kind)
      setAsk(null)
    } catch {
      // The error shows in the dialog; the job reloads behind it.
    }
  }

  function close() {
    setAsk(null)
    application.reset()
  }

  const left = daysLeft(job.apply_by)
  const mine = job.my_application
  const companySite = job.external_url && (
    <a
      href={job.external_url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => setAskOnReturn(true)}
      className={cn(action === 'apply-external' ? buttonStyles.primary : buttonStyles.secondary, 'min-h-12 w-full lg:min-h-11')}
    >
      <ExternalLink aria-hidden="true" className="size-4" />
      Open company site
    </a>
  )

  let body: ReactNode
  switch (action) {
    case 'apply':
      body = (
        <>
          <Heading>Apply in LINKS</Heading>
          <Note>
            <b className="font-semibold text-ink">Apply by {dayAndDate(job.apply_by)}</b> · {left}
          </Note>
          <Note desktopOnly>The placement office sees your name, email, USN, department and batch. Never your phone number.</Note>
          <button type="button" onClick={() => setAsk('apply')} className={buttonStyles.primary + ' min-h-12 w-full'}>
            Apply
          </button>
          <Note desktopOnly small>
            You can apply once. If you withdraw, you can't apply again.
          </Note>
        </>
      )
      break
    case 'apply-external':
      body = (
        <>
          <Heading>Apply on {job.company}'s site</Heading>
          <Note>
            <b className="font-semibold text-ink">Apply by {dayAndDate(job.apply_by)}</b> · {left}. Then come back and mark it, so the placement office counts it.
          </Note>
          {companySite}
          <button type="button" onClick={() => setAsk('marked')} className={buttonStyles.secondary + ' w-full'}>
            I applied there
          </button>
        </>
      )
      break
    case 'applied':
      body = (
        <>
          <StatusLine job={job} />
          <Heading>You applied</Heading>
          <Note>The placement office reviews applications after it closes on {dayMonth(job.apply_by)}. Your status changes here and in Jobs, under Applied.</Note>
          <button type="button" onClick={() => setAsk('withdraw')} className={buttonStyles.secondary + ' w-full lg:w-auto lg:self-start'}>
            Withdraw application
          </button>
          <Note desktopOnly small>
            Withdrawing is final.
          </Note>
        </>
      )
      break
    case 'applied-external':
      body = (
        <>
          <StatusLine job={job} />
          <Note>You marked that you applied on {job.company}'s site. The placement office counts it.</Note>
          {companySite}
          <button type="button" onClick={() => setAsk('withdraw')} className="min-h-9 self-center text-sm font-semibold text-danger hover:text-danger-ink">
            I didn't apply after all
          </button>
        </>
      )
      break
    case 'shortlisted':
      body = (
        <>
          <StatusLine job={job} />
          <Note>You're shortlisted. The placement office will contact you about the next round. You can't withdraw now.</Note>
        </>
      )
      break
    case 'selected':
      body = (
        <>
          <StatusLine job={job} />
          <Note>Congratulations. The placement office will contact you about the offer.</Note>
        </>
      )
      break
    case 'rejected':
      body = (
        <>
          <StatusLine job={job} />
          <Note>Not this time. Keep an eye on Jobs for the next drive.</Note>
        </>
      )
      break
    case 'withdrawn':
      body = (
        <>
          <StatusLine job={job} />
          <Note>
            You withdrew{mine?.withdrawn_at ? ` on ${dayMonth(mine.withdrawn_at)}` : ''}, so you can't apply to this one again.
          </Note>
        </>
      )
      break
    case 'closed':
      body = (
        <>
          <span className="flex">
            <Tag>Closed</Tag>
          </span>
          <Note>Closed on {dayMonth(job.closed_at ?? job.apply_by)}. Applications are no longer taken.</Note>
        </>
      )
      break
  }

  const error = application.error ? refusal(application.error, ask === 'withdraw' ? 'withdraw' : 'apply') : null

  return (
    <>
      <section
        aria-label="Your application"
        className="fixed inset-x-0 bottom-0 z-40 flex flex-col gap-2.5 bg-surface px-4 pt-3 pb-[max(env(safe-area-inset-bottom),20px)] shadow-[0_-1px_0_var(--color-line)] lg:static lg:gap-3 lg:rounded-xl lg:border lg:border-line lg:p-5 lg:shadow-none"
      >
        {body}
      </section>

      <ConfirmDialog
        open={ask === 'apply'}
        title={`Apply to ${job.company}?`}
        cancelLabel="Cancel"
        confirmLabel="Apply"
        busyLabel="Applying…"
        busy={application.isPending}
        error={error}
        onCancel={close}
        onConfirm={() => run('apply')}
      >
        <p>
          For <b className="font-semibold text-ink">{job.title}</b>. The placement office will see your name, email, USN, department and batch.
        </p>
        <p className="rounded-lg bg-warning-soft px-3 py-2.5 text-sm text-warning-ink">
          You can withdraw while it's still Applied, but you can't apply to this one again after that.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        open={ask === 'withdraw'}
        title="Withdraw your application?"
        cancelLabel="Keep it"
        confirmLabel="Withdraw"
        busyLabel="Withdrawing…"
        busy={application.isPending}
        danger
        error={error}
        onCancel={close}
        onConfirm={() => run('withdraw')}
      >
        <p>
          You can't apply to {job.company}'s {job.title} again.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        open={ask === 'marked'}
        title="Did you apply on their site?"
        cancelLabel="Not yet"
        confirmLabel="Yes, I applied"
        busyLabel="Saving…"
        busy={application.isPending}
        error={error}
        onCancel={close}
        onConfirm={() => run('apply')}
      >
        <p>Marking it lets the placement office count your application. It doesn't send anything to {job.company}.</p>
      </ConfirmDialog>
    </>
  )
}

function Heading({ children }: { children: ReactNode }) {
  return <h2 className="hidden font-serif text-[21px] font-medium lg:block">{children}</h2>
}

function Note({ children, desktopOnly, small }: { children: ReactNode; desktopOnly?: boolean; small?: boolean }) {
  return (
    <p className={cn('leading-[1.45] text-ink-2', small ? 'text-[13px] text-ink-3' : 'text-[13px] lg:text-sm', desktopOnly && 'hidden lg:block')}>
      {children}
    </p>
  )
}

function StatusLine({ job }: { job: Opportunity }) {
  const mine = job.my_application
  if (!mine) return null
  const status = studentStatus(mine)
  return (
    <span className="flex items-center gap-2">
      <Tag tone={status.tone}>{status.label}</Tag>
      <span className="text-[13px] text-ink-3">{mine.mode === 'external' ? 'marked' : 'since'} {dayMonth(mine.applied_at)}</span>
    </span>
  )
}

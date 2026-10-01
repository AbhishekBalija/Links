import type { MyApplication, Opportunity } from './types'

// JobAction is what the student can do or see at the bottom of a job: apply,
// mark an outside application, or where their application stands.
export type JobAction =
  | 'apply'
  | 'apply-external'
  | 'applied'
  | 'applied-external'
  | 'shortlisted'
  | 'selected'
  | 'rejected'
  | 'withdrawn'
  | 'closed'

// jobAction follows the student's own Application once there is one, open or
// closed. Without one, it is apply (in LINKS or on the company's site) while
// open, and closed after.
export function jobAction(job: Pick<Opportunity, 'open' | 'application_mode' | 'my_application'>): JobAction {
  const mine = job.my_application
  if (mine) {
    if (mine.status === 'applied') return mine.mode === 'external' ? 'applied-external' : 'applied'
    return mine.status
  }
  if (!job.open) return 'closed'
  return job.application_mode === 'external' ? 'apply-external' : 'apply'
}

export type StatusTone = 'plain' | 'warning' | 'going' | 'cancelled' | 'outline'

// studentStatus words an Application for the student. A rejection reads as
// "Not selected", which is kinder and just as clear.
export function studentStatus(application: MyApplication): { label: string; tone: StatusTone } {
  switch (application.status) {
    case 'applied':
      return { label: application.mode === 'external' ? 'Applied on their site' : 'Applied', tone: 'plain' }
    case 'shortlisted':
      return { label: 'Shortlisted', tone: 'warning' }
    case 'selected':
      return { label: 'Selected', tone: 'going' }
    case 'rejected':
      return { label: 'Not selected', tone: 'cancelled' }
    case 'withdrawn':
      return { label: 'Withdrawn', tone: 'outline' }
  }
}

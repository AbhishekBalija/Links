import type { ApplicantCounts, ApplicationStatus } from '../jobs/types'

// An Application as placement staff see it in the applicant list. Never a
// phone number (ADR 0024).
export type Applicant = {
  id: string
  opportunity_id: string
  student: {
    user_id: string
    full_name: string
    username: string
    email: string | null
    usn: string | null
    department_code: string | null
    batch_year: number | null
  }
  mode: 'internal' | 'external'
  status: ApplicationStatus
  applied_at: string
  withdrawn_at: string | null
  status_changed_at: string | null
}

export type ApplicantFilters = {
  status: ApplicationStatus | null
  department: string
  batch: string
  q: string
}

// The statuses staff move an Application between. Withdrawn is the
// student's own decision, so it is never offered.
export const staffStatuses: ApplicationStatus[] = ['applied', 'shortlisted', 'selected', 'rejected']

const labels: Record<ApplicationStatus, string> = {
  applied: 'To review',
  shortlisted: 'Shortlisted',
  selected: 'Selected',
  rejected: 'Rejected',
  withdrawn: 'Withdrawn',
}

// staffStatusLabel words a status for the office: an applied one is waiting
// for them, so it reads "To review".
export function staffStatusLabel(status: ApplicationStatus): string {
  return labels[status]
}

export type ApplicantTab = { value: ApplicationStatus | null; label: string; count?: number }

// applicantTabs are the status tabs. "All" has no number: it includes the
// withdrawn ones, which the total leaves out, so a number there would
// disagree with the total shown elsewhere.
export function applicantTabs(counts: ApplicantCounts | undefined): ApplicantTab[] {
  const statuses: ApplicationStatus[] = ['applied', 'shortlisted', 'selected', 'rejected', 'withdrawn']
  return [
    { value: null, label: 'All' },
    ...statuses.map((status) => ({ value: status, label: labels[status], ...(counts ? { count: counts[status] } : {}) })),
  ]
}

// applicantParams is the query string for one page of the list.
export function applicantParams(filters: ApplicantFilters, cursor: string): URLSearchParams {
  const params = new URLSearchParams({ limit: '50' })
  if (filters.status) params.set('status', filters.status)
  if (filters.department) params.set('department', filters.department)
  if (filters.batch) params.set('batch', filters.batch)
  const q = filters.q.trim()
  if (q) params.set('q', q)
  if (cursor) params.set('cursor', cursor)
  return params
}

const slug = (text: string) => text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

// exportName names the CSV after the company, role and status.
export function exportName(title: string, company: string, status: ApplicationStatus | null): string {
  return `${slug(company)}-${slug(title)}`.slice(0, 80) + `-${status ?? 'applicants'}.csv`
}

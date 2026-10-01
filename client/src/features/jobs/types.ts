// Placement Opportunities as members read them (ADR 0024). Field names
// follow the API.
export type OpportunityType = 'job' | 'internship' | 'training'
export type ApplicationMode = 'internal' | 'external'
export type ApplicationStatus = 'applied' | 'shortlisted' | 'rejected' | 'selected' | 'withdrawn'

// The three views of Jobs: open ones, what the student applied to, and closed ones.
export type JobState = 'open' | 'applied' | 'closed'

// The server leaves out a field that is empty, so each one may be missing.
export type EligibilityRule = {
  department_id?: string | null
  department_code?: string | null
  batch_year?: number | null
  role?: string | null
}

// MyApplication is the student's own Application. Nobody sees anyone else's here.
export type MyApplication = {
  id: string
  opportunity_id: string
  mode: ApplicationMode
  status: ApplicationStatus
  applied_at: string
  withdrawn_at: string | null
}

// ApplicantCounts is how many Applications an Opportunity has in each
// status. Only placement staff get it; withdrawn ones are not in total.
export type ApplicantCounts = {
  total: number
  applied: number
  shortlisted: number
  rejected: number
  selected: number
  withdrawn: number
}

export type Opportunity = {
  id: string
  opportunity_type: OpportunityType
  title: string
  company: string
  description: string
  location: string | null
  compensation: string | null
  apply_by: string
  application_mode: ApplicationMode
  external_url: string | null
  eligibility: EligibilityRule[]
  status: 'draft' | 'published' | 'closed'
  // True while it is published and apply_by is ahead.
  open: boolean
  my_application: MyApplication | null
  posted_by: { user_id: string; full_name: string }
  published_at: string | null
  closed_at: string | null
  created_at: string
  updated_at: string
  applicant_counts?: ApplicantCounts
}

export const opportunityTypes: { value: OpportunityType; label: string }[] = [
  { value: 'job', label: 'Job' },
  { value: 'internship', label: 'Internship' },
  { value: 'training', label: 'Training' },
]

export function isOpportunityType(value: string | null): value is OpportunityType {
  return opportunityTypes.some((type) => type.value === value)
}

export function typeLabel(type: OpportunityType): string {
  return opportunityTypes.find((t) => t.value === type)?.label ?? type
}

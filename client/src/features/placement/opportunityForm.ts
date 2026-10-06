import type { ApplicationMode, Opportunity, OpportunityType } from '../jobs/types'
import { fromRules, MAX_RULES, ruleCount, toRules, type EligibilityChoice } from './eligibility'
import { collegeDate, collegeTimeInput, fromCollegeTime } from '../../shared/time/college'

// OpportunityForm is the form as placement staff fill it in. The deadline
// is a date and a time in the officer's time zone, as the inputs give them.
export type OpportunityForm = {
  opportunity_type: OpportunityType | null
  title: string
  company: string
  location: string
  compensation: string
  applyDate: string
  applyTime: string
  description: string
  application_mode: ApplicationMode
  external_url: string
  eligibility: EligibilityChoice
}

export type OpportunityErrors = Partial<
  Record<'opportunity_type' | 'title' | 'company' | 'location' | 'compensation' | 'apply_by' | 'description' | 'external_url' | 'eligibility', string>
>

// What the server receives on create; an update sends the same fields.
export type OpportunityInput = {
  opportunity_type: OpportunityType
  title: string
  company: string
  description: string
  location: string | null
  compensation: string | null
  apply_by: string
  application_mode: ApplicationMode
  external_url: string | null
  eligibility: ReturnType<typeof toRules>
}

export function emptyOpportunity(): OpportunityForm {
  return {
    opportunity_type: null,
    title: '',
    company: '',
    location: '',
    compensation: '',
    applyDate: '',
    applyTime: '23:59',
    description: '',
    application_mode: 'internal',
    external_url: '',
    eligibility: { everyone: true, departments: [], batches: [] },
  }
}


export function fromOpportunity(item: Opportunity): OpportunityForm {
  return {
    opportunity_type: item.opportunity_type,
    title: item.title,
    company: item.company,
    location: item.location ?? '',
    compensation: item.compensation ?? '',
    applyDate: collegeDate(item.apply_by, 'input'),
    applyTime: collegeTimeInput(item.apply_by),
    description: item.description,
    application_mode: item.application_mode,
    external_url: item.external_url ?? '',
    // Rules the chips can't show start over as every student; the form says so.
    eligibility: fromRules(item.eligibility) ?? { everyone: true, departments: [], batches: [] },
  }
}

// applyBy joins the date and time inputs into the instant they mean in the
// college (#211).
export function applyBy(form: OpportunityForm): Date | null {
  if (!form.applyDate || !form.applyTime) return null
  return fromCollegeTime(form.applyDate, form.applyTime)
}

// checkOpportunity finds what must be fixed before saving, with the server's
// limits. A draft may keep a past deadline; publishing needs one ahead.
export function checkOpportunity(form: OpportunityForm, now: Date, { publishing }: { publishing: boolean }): OpportunityErrors {
  const errors: OpportunityErrors = {}
  if (!form.opportunity_type) errors.opportunity_type = 'Choose a job, internship or training.'
  const title = form.title.trim()
  if (!title) errors.title = 'Write the role.'
  else if (title.length < 3) errors.title = 'Use at least 3 characters.'
  else if (title.length > 200) errors.title = 'Keep it under 200 characters.'
  const company = form.company.trim()
  if (!company) errors.company = "Add the company's name."
  else if (company.length > 200) errors.company = 'Keep it under 200 characters.'
  if (form.location.trim().length > 200) errors.location = 'Keep it under 200 characters.'
  if (form.compensation.trim().length > 200) errors.compensation = 'Keep it under 200 characters.'
  if (form.description.length > 10000) errors.description = 'Keep it under 10,000 characters.'

  const due = applyBy(form)
  if (!due) errors.apply_by = 'Choose the last day to apply.'
  else if (publishing && due <= now) errors.apply_by = 'Pick a time after now.'

  if (form.application_mode === 'external') {
    const url = form.external_url.trim()
    if (!url || url === 'https://') errors.external_url = 'Add the full link students apply at.'
    else if (!/^https?:\/\/\S+\.\S+/.test(url)) errors.external_url = 'Start the link with https://.'
  }

  const count = ruleCount(form.eligibility)
  if (count > MAX_RULES) errors.eligibility = `That makes ${count} groups; pick at most ${MAX_RULES}, or choose All departments.`
  return errors
}

// toInput is what the server receives. Call it once checkOpportunity passes.
export function toInput(form: OpportunityForm): OpportunityInput {
  const due = applyBy(form)
  if (!form.opportunity_type || !due) throw new Error('check the opportunity before sending it')
  const optional = (value: string) => value.trim() || null
  return {
    opportunity_type: form.opportunity_type,
    title: form.title.trim(),
    company: form.company.trim(),
    description: form.description.trim(),
    location: optional(form.location),
    compensation: optional(form.compensation),
    apply_by: due.toISOString(),
    application_mode: form.application_mode,
    external_url: form.application_mode === 'external' ? form.external_url.trim() : null,
    eligibility: toRules(form.eligibility),
  }
}

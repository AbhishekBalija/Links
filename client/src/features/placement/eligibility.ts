import type { RuleInput } from '../announcements/types'
import { audienceLabel } from '../notices/format'
import type { EligibilityRule } from '../jobs/types'

// The server takes at most this many Eligibility rules.
export const MAX_RULES = 20

// EligibilityChoice is what the "Who can apply" chips hold. Opportunities are
// for students, so every rule carries the student role. No department chosen
// means every department; no batch chosen means every batch.
export type EligibilityChoice = {
  everyone: boolean
  departments: string[]
  batches: number[]
}

// toRules turns the chips into Eligibility rules: one per department and
// batch pair, so "CS and IS, batch 2023" is two rules.
export function toRules(choice: EligibilityChoice): RuleInput[] {
  if (choice.everyone) return [{ role: 'student' }]
  const departments: (string | undefined)[] = choice.departments.length ? choice.departments : [undefined]
  const batches: (number | undefined)[] = choice.batches.length ? choice.batches : [undefined]
  const rules: RuleInput[] = []
  for (const batch of batches) {
    for (const department of departments) {
      rules.push({
        ...(department ? { department_id: department } : {}),
        ...(batch ? { batch_year: batch } : {}),
        role: 'student',
      })
    }
  }
  return rules
}

// ruleCount is how many rules a choice makes, to check against MAX_RULES.
export function ruleCount(choice: EligibilityChoice): number {
  if (choice.everyone) return 1
  return Math.max(choice.departments.length, 1) * Math.max(choice.batches.length, 1)
}

// fromRules reads saved rules back into chips, or returns null when the
// rules are something the chips could not have made (set through the API).
export function fromRules(rules: (EligibilityRule | RuleInput)[]): EligibilityChoice | null {
  if (rules.length === 0) return null
  if (rules.some((rule) => rule.role !== 'student')) return null
  if (rules.length === 1 && !rules[0].department_id && !rules[0].batch_year) return { everyone: true, departments: [], batches: [] }
  const departments = [...new Set(rules.map((rule) => rule.department_id).filter((id): id is string => Boolean(id)))]
  const batches = [...new Set(rules.map((rule) => rule.batch_year).filter((year): year is number => Boolean(year)))]
  const choice = { everyone: false, departments, batches }
  // Only a full department-by-batch grid comes from the chips.
  if (ruleCount(choice) !== rules.length) return null
  const made = toRules(choice)
  const same = rules.every((rule) => made.some((m) => m.department_id === (rule.department_id ?? undefined) && m.batch_year === (rule.batch_year ?? undefined)))
  return same ? choice : null
}

function list(items: string[]): string {
  if (items.length <= 1) return items.join('')
  return `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`
}

// eligibilitySummary says who can apply: "CS and IS students, batch 2023".
export function eligibilitySummary(choice: EligibilityChoice, departments: { id: string; code: string }[]): string {
  if (choice.everyone) return 'every student'
  const codes = choice.departments.map((id) => departments.find((d) => d.id === id)?.code ?? '…').sort()
  const batches = [...choice.batches].sort((a, b) => a - b).map(String)
  const who = codes.length ? `${list(codes)} students` : 'students of every department'
  if (batches.length === 0) return `${who}, every batch`
  return `${who}, ${batches.length === 1 ? 'batch' : 'batches'} ${list(batches)}`
}

// describeEligibility puts saved rules into one sentence for lists and pages:
// "CS and AD students, batch 2023". Rules the chips could not have made are
// listed one by one instead.
export function describeEligibility(rules: EligibilityRule[]): string {
  // No rules at all means everyone, staff included.
  if (rules.length === 0) return 'Everyone'
  const choice = fromRules(rules)
  const sentence = choice
    ? eligibilitySummary(
        choice,
        rules.flatMap((rule) => (rule.department_id && rule.department_code ? [{ id: rule.department_id, code: rule.department_code }] : [])),
      )
    : audienceLabel(
        rules.map((rule) => ({
          department_id: rule.department_id ?? null,
          department_code: rule.department_code ?? null,
          batch_year: rule.batch_year ?? null,
          role: rule.role ?? null,
        })),
      )
  return sentence.charAt(0).toUpperCase() + sentence.slice(1)
}

import { audienceLabel } from '../notices/format'
import type { AudienceRule } from '../notices/types'
import type { RuleInput } from './types'

export type Preset = { key: string; label: string; audience: RuleInput[] }

type Dept = { id: string; code: string } | null

// Quick picks cover what most authors send: their own Department, its
// students or its faculty, or the whole college. Anything else is a custom
// group.
export function presetsFor(department: Dept): Preset[] {
  const presets: Preset[] = [{ key: 'college', label: 'Whole college', audience: [] }]
  if (department) {
    presets.push(
      { key: 'dept', label: `Everyone in ${department.code}`, audience: [{ department_id: department.id }] },
      { key: 'dept-students', label: `${department.code} students`, audience: [{ department_id: department.id, role: 'student' }] },
      { key: 'dept-faculty', label: `${department.code} faculty`, audience: [{ department_id: department.id, role: 'faculty' }] },
    )
  } else {
    presets.push(
      { key: 'students', label: 'All students', audience: [{ role: 'student' }] },
      { key: 'faculty', label: 'All faculty', audience: [{ role: 'faculty' }] },
    )
  }
  return presets
}

function sameRule(a: RuleInput, b: RuleInput) {
  return a.department_id === b.department_id && a.batch_year === b.batch_year && a.role === b.role
}

// matchPreset finds the quick pick an audience equals, so reopening a notice
// shows "CS students" rather than a custom group.
export function matchPreset(audience: RuleInput[], presets: Preset[]): string | null {
  for (const preset of presets) {
    if (preset.audience.length === audience.length && preset.audience.every((rule, i) => sameRule(rule, audience[i]))) {
      return preset.key
    }
  }
  return null
}

// Only students have a batch, so a batch on any other role matches no one.
const batchRoles = ['student', 'student_coordinator']
export function batchApplies(role: string | undefined) {
  return role === undefined || batchRoles.includes(role)
}

export const roleOptions = [
  { value: '', label: 'Any role' },
  { value: 'student', label: 'Students' },
  { value: 'student_coordinator', label: 'Student coordinators' },
  { value: 'faculty', label: 'Faculty' },
  { value: 'hod', label: 'HODs' },
]

export function batchOptions(now = new Date()) {
  const year = now.getFullYear()
  return Array.from({ length: 6 }, (_, i) => year - i)
}

// A group needs at least one field; an empty one would match everyone,
// which is what "Whole college" is for.
export function isEmptyRule(rule: RuleInput) {
  return !rule.department_id && !rule.batch_year && !rule.role
}

// describe puts an audience into words, e.g. "CS students, batch 2023".
export function describe(audience: RuleInput[], codes: Map<string, string>) {
  return audienceLabel(
    audience.map((rule) => ({
      department_id: rule.department_id ?? null,
      department_code: rule.department_id ? (codes.get(rule.department_id) ?? '…') : null,
      batch_year: rule.batch_year ?? null,
      role: rule.role ?? null,
    })),
  )
}

// toRules turns rules as the server returns them (nulls for "anyone") into
// rules as the composer sends them (fields left out).
export function toRules(audience: (AudienceRule | RuleInput)[]): RuleInput[] {
  return audience.map((r) => ({
    ...(r.department_id ? { department_id: r.department_id } : {}),
    ...(r.batch_year ? { batch_year: r.batch_year } : {}),
    ...(r.role ? { role: r.role } : {}),
  }))
}

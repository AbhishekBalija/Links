import { wallClock } from '../../shared/time/college'

// The role filter offers the roles people look for; the labels read as a
// group ("Students", not "student").
export const roleOptions = [
  { value: 'student', label: 'Students' },
  { value: 'faculty', label: 'Faculty' },
  { value: 'hod', label: 'HODs' },
  { value: 'student_coordinator', label: 'Coordinators' },
  { value: 'placement_officer', label: 'Placement office' },
]

export function roleOptionLabel(role: string | undefined) {
  return roleOptions.find((option) => option.value === role)?.label
}

// Batches still studying: the four years before this one and this year's
// new students.
export function batchOptions(now = new Date()) {
  const year = wallClock(now).year
  return [year - 4, year - 3, year - 2, year - 1, year].map(String)
}

// Only students have a Batch, so the Batch filter shows for everyone and
// for students, not for staff roles.
export function allowsBatch(role: string | undefined) {
  return !role || role === 'student'
}

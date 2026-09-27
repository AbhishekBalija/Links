import { describe, expect, it } from 'vitest'
import { routeFor } from './route'

const cs = { id: 'd1', code: 'CS' }

describe('routeFor', () => {
  it('sends a faculty or coordinator proposal to the HOD, then the principal', () => {
    expect(routeFor({ roles: ['faculty'], eventType: 'talk', department: cs, hasHOD: true })).toEqual({
      sentence: "Goes to the CS HOD, then the principal. It's published once both approve.",
      button: 'Submit for approval',
    })
    expect(routeFor({ roles: ['student', 'student_coordinator'], eventType: 'talk', department: cs, hasHOD: true }).sentence).toBe(
      "Goes to the CS HOD, then the principal. It's published once both approve.",
    )
  })

  it('lets the principal do both reviews when the Department has no HOD', () => {
    expect(routeFor({ roles: ['faculty'], eventType: 'talk', department: cs, hasHOD: false }).sentence).toBe(
      'No CS HOD is listed, so the principal does both reviews.',
    )
  })

  it("counts an HOD's own approval", () => {
    expect(routeFor({ roles: ['hod'], eventType: 'talk', department: cs, hasHOD: true }).sentence).toBe(
      'Goes to the principal for final approval. Your approval as HOD is already counted.',
    )
  })

  it('sends placement training straight to final approval', () => {
    expect(routeFor({ roles: ['placement_officer'], eventType: 'training', department: null, hasHOD: false }).sentence).toBe(
      'Goes to the principal for final approval.',
    )
  })

  it('publishes straight away for the principal and admins', () => {
    for (const role of ['principal', 'admin']) {
      expect(routeFor({ roles: [role], eventType: 'cultural', department: null, hasHOD: false })).toEqual({
        sentence: 'Publishes now. Everyone invited sees it straight away.',
        button: 'Publish',
      })
    }
  })

  it('sends a proposal back to whoever asked for changes', () => {
    expect(routeFor({ roles: ['faculty'], eventType: 'talk', department: cs, hasHOD: true, resubmitting: 'hod_changes_requested' }).sentence).toBe(
      'Goes back to the CS HOD, then the principal.',
    )
    expect(routeFor({ roles: ['faculty'], eventType: 'talk', department: cs, hasHOD: true, resubmitting: 'final_changes_requested' })).toEqual({
      sentence: 'Goes back to the principal for final approval.',
      button: 'Resubmit',
    })
  })
})

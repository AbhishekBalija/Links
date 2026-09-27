import { audienceLabel } from '../notices/format'
import { dayMonth } from './standing'
import type { EventQueueItem } from './types'

export type ApprovalCopy = {
  // What approving does, beside the buttons.
  sentence: string
  button: string
  // Final approval publishes, so it is confirmed first.
  publishes: boolean
  // At final approval: what happened at the HOD stage.
  earlier: string | null
}

// approvalCopy says what approving an event does at its stage (ADR 0023).
export function approvalCopy(item: EventQueueItem, roles: string[]): ApprovalCopy {
  const code = item.department?.code
  if (item.stage === 'final') {
    const hodApproval = [...(item.reviews ?? [])].reverse().find((r) => r.stage === 'hod' && r.decision === 'approve')
    const audience = audienceLabel(
      (item.audience ?? []).map((r) => ({ department_id: r.department_id ?? null, department_code: r.department_code ?? null, batch_year: r.batch_year ?? null, role: r.role ?? null })),
    )
    return {
      sentence: `Approving publishes it to ${audience === 'Whole college' ? 'the whole college' : audience} now.`,
      button: 'Approve and publish',
      publishes: true,
      earlier: hodApproval
        ? `${hodApproval.reviewer_name}${code ? `, ${code} HOD` : ''} approved it on ${dayMonth(hodApproval.decided_at)}. Yours is the final approval.`
        : 'It skipped the HOD review, so yours is the only approval.',
    }
  }
  // The principal and admins review the HOD stage only where there is no HOD.
  const standsIn = !roles.includes('hod') && (roles.includes('principal') || roles.includes('admin'))
  return {
    sentence: standsIn
      ? `No ${code ? `${code} ` : ''}HOD is listed, so you review it here and again at final approval.`
      : 'Approving sends it to the principal for final approval.',
    button: 'Approve',
    publishes: false,
    earlier: null,
  }
}

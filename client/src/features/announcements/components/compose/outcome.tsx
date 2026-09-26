import { approverPhrase } from '../../status'
import type { Preview } from '../../types'

export type Mode = 'new' | 'draft' | 'published'

// outcomeFor turns the publishing preview into the sentence beside the
// buttons and the main button's label.
export function outcomeFor(preview: Preview | undefined, mode: Mode) {
  if (!preview) return { sentence: 'Checking who approves this…', button: mode === 'published' ? 'Save changes' : 'Send' }
  const people = `${preview.reach.toLocaleString('en-IN')} ${preview.reach === 1 ? 'person sees' : 'people see'} it`
  const approver = approverPhrase(preview.approver)
  if (mode === 'published') {
    return preview.publishes_directly
      ? { sentence: <><b className="font-semibold text-ink">Your changes go live now.</b> {people} straight away.</>, button: 'Publish changes' }
      : { sentence: <>Your changes go to <b className="font-semibold text-ink">{approver}</b>. Readers keep the current version until then.</>, button: 'Submit changes' }
  }
  return preview.publishes_directly
    ? { sentence: <><b className="font-semibold text-ink">Publishes now.</b> {people} straight away.</>, button: 'Publish' }
    : { sentence: <>Goes to <b className="font-semibold text-ink">{approver}</b> for approval. Readers see it once it's approved.</>, button: 'Submit for approval' }
}

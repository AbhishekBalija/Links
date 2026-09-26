import { ChevronRight } from 'lucide-react'
import { useId } from 'react'
import { friendlyDate, toDateInput, toEndOfDay } from './expiryDate'

// PhoneRows are the composer's "who sees it" and expiry rows on phones.
export function PhoneRows({ audience, reach, expiry, audienceError, expiryError, onOpenAudience, onExpiry }: {
  audience: string
  reach: number | undefined
  expiry: string | null
  audienceError?: string
  expiryError?: string
  onOpenAudience: () => void
  onExpiry: (v: string | null) => void
}) {
  const id = useId()
  return (
    <div className="flex flex-col rounded-[10px] border border-line bg-surface">
      <button type="button" onClick={onOpenAudience} className="flex min-h-14 items-center justify-between gap-3 px-4 py-2 text-left">
        <span className="flex flex-col">
          <span className="text-xs text-ink-3">Who sees it</span>
          <span className="text-[15px] font-semibold">{audience}</span>
          {audienceError && <span className="text-[13px] font-semibold text-danger">{audienceError}</span>}
        </span>
        <span className="flex items-center gap-1 font-mono text-xs text-ink-3">
          {reach !== undefined && reach.toLocaleString('en-IN')}
          <ChevronRight aria-hidden="true" className="size-4" />
        </span>
      </button>
      <div className="flex min-h-14 items-center justify-between gap-3 px-4 py-2 shadow-[0_-1px_0_var(--color-line)]">
        <label htmlFor={id} className="flex flex-col">
          <span className="text-xs text-ink-3">Expires (optional)</span>
          <span className="text-[15px]">{expiry ? <>After <b className="font-semibold">{friendlyDate(expiry)}</b></> : 'No end date'}</span>
          {expiryError && <span className="text-[13px] font-semibold text-danger">{expiryError}</span>}
        </label>
        <input
          id={id}
          type="date"
          value={toDateInput(expiry)}
          min={toDateInput(new Date().toISOString())}
          onChange={(e) => onExpiry(e.target.value ? toEndOfDay(e.target.value) : null)}
          className="min-h-10 w-[136px] rounded-[7px] border border-input bg-paper px-2 font-mono text-[13px]"
        />
      </div>
    </div>
  )
}

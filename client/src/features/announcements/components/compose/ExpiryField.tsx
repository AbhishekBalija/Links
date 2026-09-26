import { X } from 'lucide-react'
import { useId } from 'react'
import { friendlyDate, toDateInput, toEndOfDay } from './expiryDate'

// ExpiryField is the optional end date on desktop.
export function ExpiryField({ value, onChange, error }: { value: string | null; onChange: (v: string | null) => void; error?: string }) {
  const id = useId()
  return (
    <section aria-label="Expiry" className="flex flex-col gap-2 rounded-xl border border-line bg-surface py-3.5 pr-3 pl-5">
      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <label htmlFor={id} className="text-sm font-semibold">
            Expires <span className="font-normal text-ink-3">(optional)</span>
          </label>
          <span className="text-xs text-ink-2">
            {value ? (
              <>
                Leaves the feed after <b className="font-semibold">{friendlyDate(value)}</b>
              </>
            ) : (
              'Stays in the feed until you withdraw it.'
            )}
          </span>
        </div>
        <span className="flex items-center gap-0.5">
          <input
            id={id}
            type="date"
            value={toDateInput(value)}
            min={toDateInput(new Date().toISOString())}
            onChange={(e) => onChange(e.target.value ? toEndOfDay(e.target.value) : null)}
            className="min-h-10 rounded-[7px] border border-input bg-paper px-2.5 font-mono text-sm"
          />
          {value && (
            <button
              type="button"
              aria-label="Remove expiry"
              onClick={() => onChange(null)}
              className="flex size-10 items-center justify-center rounded-lg text-ink-3 hover:bg-well hover:text-ink"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          )}
        </span>
      </div>
      {error && <p className="text-[13px] font-semibold text-danger">{error}</p>}
    </section>
  )
}

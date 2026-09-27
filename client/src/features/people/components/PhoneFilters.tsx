import { ChevronDown, X } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { useModal } from '../../../shared/ui/useModal'
import { useDepartments } from '../../announcements/api'
import { buttonStyles } from '../../announcements/buttons'
import { useDirectoryCount } from '../api'
import { peopleWord } from '../labels'
import { allowsBatch, batchOptions, roleOptionLabel, roleOptions } from '../options'
import type { Filters } from '../types'

type Props = {
  filters: Filters
  onChange: (next: Partial<Filters>) => void
}

// PhoneFilters are chips under the search. A set filter shows as a dark chip
// with its own clear button; tapping any chip opens one sheet with every
// filter, so all the choices are in one place.
export function PhoneFilters({ filters, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const roleLabel = roleOptionLabel(filters.role)
  const roleChip = [roleLabel, filters.batch].filter(Boolean).join(' · ')

  return (
    <>
      <div className="-mx-4 flex gap-2 overflow-x-auto px-4 pb-0.5">
        {filters.department ? (
          <SetChip label={filters.department} onOpen={() => setOpen(true)} onClear={() => onChange({ department: undefined })} />
        ) : (
          <Chip label="Department" onOpen={() => setOpen(true)} />
        )}
        {roleChip ? (
          <SetChip label={roleChip} onOpen={() => setOpen(true)} onClear={() => onChange({ role: undefined, batch: undefined })} />
        ) : (
          <Chip label="Role" onOpen={() => setOpen(true)} />
        )}
      </div>
      {open && (
        <FilterSheet
          filters={filters}
          onClose={() => setOpen(false)}
          onApply={(next) => {
            setOpen(false)
            onChange(next)
          }}
        />
      )}
    </>
  )
}

function Chip({ label, onOpen }: { label: string; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="inline-flex min-h-10 shrink-0 items-center gap-1.5 rounded-full border border-[#d6ccbb] bg-surface px-3.5 text-sm font-medium text-ink"
    >
      {label}
      <ChevronDown aria-hidden="true" className="size-4 text-ink-3" />
    </button>
  )
}

function SetChip({ label, onOpen, onClear }: { label: string; onOpen: () => void; onClear: () => void }) {
  return (
    <span className="inline-flex min-h-10 shrink-0 items-center rounded-full border border-ink bg-ink text-sm font-semibold text-paper">
      <button type="button" onClick={onOpen} className="min-h-10 pr-1 pl-3.5">
        {label}
      </button>
      <button type="button" onClick={onClear} aria-label={`Clear ${label}`} className="flex size-10 items-center justify-center">
        <X aria-hidden="true" className="size-4" />
      </button>
    </span>
  )
}

type SheetProps = {
  filters: Filters
  onClose: () => void
  onApply: (next: Partial<Filters>) => void
}

// FilterSheet collects the choices and applies them together. The button
// says how many people they would show, so nobody applies a dead end.
function FilterSheet({ filters, onClose, onApply }: SheetProps) {
  const ref = useModal(true)
  const departments = useDepartments()
  const [draft, setDraft] = useState<Filters>({ department: filters.department, role: filters.role, batch: filters.batch })
  const count = useDirectoryCount({ ...draft, q: filters.q }, true)

  function set(next: Partial<Filters>) {
    setDraft((current) => {
      const merged = { ...current, ...next }
      if (!allowsBatch(merged.role)) merged.batch = undefined
      return merged
    })
  }

  const showLabel = count.data === undefined ? 'Show people' : `Show ${peopleWord(count.data, draft.role)}`

  return (
    <dialog
      ref={ref}
      aria-labelledby="filter-h"
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45"
    >
      <header className="flex items-center justify-between pt-4 pr-3 pb-1.5 pl-5">
        <h2 id="filter-h" className="font-serif text-2xl font-medium">
          Filter people
        </h2>
        <button type="button" onClick={onClose} aria-label="Close" className="flex size-11 items-center justify-center text-ink-3">
          <X aria-hidden="true" className="size-5" />
        </button>
      </header>
      <div className="flex flex-col gap-[18px] px-5 pt-1.5 pb-[18px]">
        <Options
          legend="Department"
          value={draft.department}
          onPick={(department) => set({ department })}
          options={[{ value: undefined, label: 'All' }, ...(departments.data ?? []).map((d) => ({ value: d.code, label: d.code }))]}
        />
        <Options
          legend="Role"
          value={draft.role}
          onPick={(role) => set({ role })}
          options={[{ value: undefined, label: 'Everyone' }, ...roleOptions]}
        />
        {allowsBatch(draft.role) && (
          <Options
            legend="Batch"
            hint="students only"
            value={draft.batch}
            onPick={(batch) => set({ batch })}
            options={[{ value: undefined, label: 'Any' }, ...batchOptions().map((year) => ({ value: year, label: year }))]}
          />
        )}
      </div>
      <footer className="flex gap-2.5 bg-surface px-4 pt-3 pb-[max(env(safe-area-inset-bottom),26px)] shadow-[0_-1px_0_var(--color-line)]">
        <button type="button" onClick={() => setDraft({})} className={buttonStyles.secondary}>
          Clear
        </button>
        <button
          type="button"
          onClick={() => onApply({ department: draft.department, role: draft.role, batch: draft.batch })}
          className={buttonStyles.primary}
        >
          {showLabel}
        </button>
      </footer>
    </dialog>
  )
}

type OptionsProps = {
  legend: string
  hint?: string
  value: string | undefined
  onPick: (value: string | undefined) => void
  options: { value: string | undefined; label: string }[]
}

function Options({ legend, hint, value, onPick, options }: OptionsProps) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 text-[13px] font-semibold text-ink-2">
        {legend} {hint && <span className="font-normal text-ink-3">({hint})</span>}
      </legend>
      <div className="flex flex-wrap gap-2">
        {options.map((option) => {
          const selected = option.value === value
          return (
            <button
              key={option.label}
              type="button"
              aria-pressed={selected}
              onClick={() => onPick(option.value)}
              className={cn(
                'min-h-10 rounded-full border px-3.5 text-sm',
                selected ? 'border-ink bg-ink font-semibold text-paper' : 'border-[#d6ccbb] bg-surface text-ink',
              )}
            >
              {option.label}
            </button>
          )
        })}
      </div>
    </fieldset>
  )
}

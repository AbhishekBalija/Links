import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useDepartments } from '../../announcements/api'
import { allowsBatch, batchOptions, roleOptions } from '../options'
import type { Filters } from '../types'

type Props = {
  filters: Filters
  onChange: (next: Partial<Filters>) => void
}

// DesktopFilters are plain selects next to each other. A select that
// narrows the list is filled dark, so what's filtered reads at a glance.
export function DesktopFilters({ filters, onChange }: Props) {
  const departments = useDepartments()
  return (
    <div className="flex items-end gap-3">
      <Select
        label="Department"
        value={filters.department ?? ''}
        onChange={(value) => onChange({ department: value || undefined })}
        width="w-60"
        options={[
          { value: '', label: 'All departments' },
          ...(departments.data ?? []).map((d) => ({ value: d.code, label: `${d.code} · ${d.name}` })),
        ]}
      />
      <Select
        label="Role"
        value={filters.role ?? ''}
        onChange={(value) => onChange({ role: value || undefined, batch: allowsBatch(value || undefined) ? filters.batch : undefined })}
        width="w-52"
        options={[{ value: '', label: 'Everyone' }, ...roleOptions]}
      />
      {allowsBatch(filters.role) && (
        <Select
          label="Batch"
          value={filters.batch ?? ''}
          onChange={(value) => onChange({ batch: value || undefined })}
          width="w-36"
          options={[{ value: '', label: 'Any' }, ...batchOptions().map((year) => ({ value: year, label: year }))]}
        />
      )}
    </div>
  )
}

type SelectProps = {
  label: string
  value: string
  onChange: (value: string) => void
  options: { value: string; label: string }[]
  width: string
}

function Select({ label, value, onChange, options, width }: SelectProps) {
  const active = value !== ''
  return (
    <label className="flex flex-col gap-1">
      <span className="text-xs text-ink-3">{label}</span>
      <span className={cn('relative flex', width)}>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            'min-h-[42px] w-full cursor-pointer appearance-none truncate rounded-lg border pr-9 pl-3 text-sm font-medium',
            active ? 'border-ink bg-ink text-paper' : 'border-[#d6ccbb] bg-surface text-ink',
          )}
        >
          {options.map((option) => (
            <option key={option.value} value={option.value} className="bg-surface text-ink">
              {option.label}
            </option>
          ))}
        </select>
        <ChevronDown aria-hidden="true" className={cn('pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2', active ? 'text-paper' : 'text-ink-3')} />
      </span>
    </label>
  )
}

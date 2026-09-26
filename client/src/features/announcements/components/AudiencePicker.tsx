import { Plus, X } from 'lucide-react'
import { useId, useState } from 'react'
import { cn } from '@/lib/utils'
import type { Category } from '../../notices/types'
import { usePreview } from '../api'
import { batchApplies, batchOptions, describe, isEmptyRule, matchPreset, presetsFor, roleOptions, type Preset } from '../audience'
import type { Department, RuleInput } from '../types'

type Props = {
  value: RuleInput[]
  onChange: (audience: RuleInput[]) => void
  department: { id: string; code: string } | null
  departments: Department[]
  category: Category
  error?: string
}

const CUSTOM = 'custom'

// AudiencePicker answers "who sees it". Quick picks cover the usual cases in
// one tap; "Choose groups" opens the full builder. The custom groups are kept
// while a quick pick is chosen, so switching back doesn't lose them.
export function AudiencePicker({ value, onChange, department, departments, category, error }: Props) {
  const presets = presetsFor(department)
  const matched = matchPreset(value, presets)
  const [custom, setCustom] = useState<RuleInput[]>(() => (matched ? [{}] : value))
  const [mode, setMode] = useState(matched ?? CUSTOM)
  const name = useId()
  const codes = new Map(departments.map((d) => [d.id, d.code]))

  function choose(key: string) {
    setMode(key)
    if (key === CUSTOM) {
      onChange(custom.filter((rule) => !isEmptyRule(rule)))
    } else {
      onChange(presets.find((p) => p.key === key)?.audience ?? [])
    }
  }

  function updateCustom(next: RuleInput[]) {
    setCustom(next)
    onChange(next.filter((rule) => !isEmptyRule(rule)))
  }

  return (
    <div className="flex flex-col gap-2.5">
      <div role="radiogroup" aria-label="Who sees it" className="flex flex-col gap-0.5">
        {presets.map((preset) => (
          <PresetOption key={preset.key} preset={preset} name={name} category={category} checked={mode === preset.key} onChoose={choose} />
        ))}
        <label className={cn('flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm', mode === CUSTOM && 'bg-well font-semibold')}>
          <input type="radio" name={name} checked={mode === CUSTOM} onChange={() => choose(CUSTOM)} className="size-4 accent-ink" />
          Choose groups…
        </label>
      </div>

      {mode === CUSTOM && (
        <div className="flex flex-col gap-2 rounded-[10px] bg-paper p-2.5">
          {custom.map((rule, index) => (
            <GroupRow
              key={index}
              index={index}
              rule={rule}
              departments={departments}
              canRemove={custom.length > 1}
              onChange={(next) => updateCustom(custom.map((r, i) => (i === index ? next : r)))}
              onRemove={() => updateCustom(custom.filter((_, i) => i !== index))}
            />
          ))}
          <button
            type="button"
            onClick={() => updateCustom([...custom, {}])}
            className="inline-flex min-h-10 items-center gap-1.5 self-start px-1 text-sm font-semibold text-rust hover:text-rust-deep"
          >
            <Plus aria-hidden="true" className="size-4" />
            Add a group
          </button>
        </div>
      )}

      <Summary value={value} category={category} codes={codes} />
      {error && <p className="text-[13px] font-semibold text-danger">{error}</p>}
    </div>
  )
}

function PresetOption({ preset, name, category, checked, onChoose }: {
  preset: Preset
  name: string
  category: Category
  checked: boolean
  onChoose: (key: string) => void
}) {
  const preview = usePreview(category, preset.audience)
  return (
    <label className={cn('flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm', checked && 'bg-well font-semibold')}>
      <input type="radio" name={name} checked={checked} onChange={() => onChoose(preset.key)} className="size-4 accent-ink" />
      <span className="flex-1">{preset.label}</span>
      {preview.data && (
        <span className="font-mono text-xs font-normal text-ink-3" aria-label={`${preview.data.reach} people`}>
          {preview.data.reach.toLocaleString('en-IN')}
        </span>
      )}
    </label>
  )
}

function Summary({ value, category, codes }: { value: RuleInput[]; category: Category; codes: Map<string, string> }) {
  const preview = usePreview(category, value)
  if (value.length === 0 && preview.data === undefined) return null
  return (
    <p className="flex items-baseline justify-between gap-3 text-[13px] text-ink-2" aria-live="polite">
      <span>
        Sent to <b className="font-semibold text-ink">{describe(value, codes)}</b>
      </span>
      {preview.data && <span className="shrink-0 font-mono text-xs text-ink-3">about {preview.data.reach.toLocaleString('en-IN')} people</span>}
    </p>
  )
}

const selectClass =
  'min-h-10 w-full rounded-[7px] border border-input bg-surface px-2 text-sm font-medium text-ink disabled:border-dashed disabled:bg-paper disabled:text-ink-3'

function GroupRow({ index, rule, departments, canRemove, onChange, onRemove }: {
  index: number
  rule: RuleInput
  departments: Department[]
  canRemove: boolean
  onChange: (rule: RuleInput) => void
  onRemove: () => void
}) {
  const id = useId()
  const batchAllowed = batchApplies(rule.role)
  return (
    <fieldset className="flex flex-col gap-1">
      <legend className="sr-only">Group {index + 1}</legend>
      <div className="flex items-end gap-2">
        <label className="flex min-w-0 flex-[1.1] flex-col gap-1">
          <span className="text-[11px] text-ink-3">Department</span>
          <select
            className={selectClass}
            value={rule.department_id ?? ''}
            onChange={(e) => onChange({ ...rule, department_id: e.target.value || undefined })}
          >
            <option value="">Any</option>
            {departments.map((d) => (
              <option key={d.id} value={d.id}>
                {d.code}
              </option>
            ))}
          </select>
        </label>
        <label className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-[11px] text-ink-3">Batch</span>
          <select
            className={selectClass}
            disabled={!batchAllowed}
            aria-describedby={batchAllowed ? undefined : `${id}-batch`}
            value={batchAllowed ? (rule.batch_year ?? '') : ''}
            onChange={(e) => onChange({ ...rule, batch_year: e.target.value ? Number(e.target.value) : undefined })}
          >
            <option value="">Any</option>
            {batchOptions().map((year) => (
              <option key={year} value={year}>
                {year}
              </option>
            ))}
          </select>
        </label>
        <label className="flex min-w-0 flex-[1.3] flex-col gap-1">
          <span className="text-[11px] text-ink-3">Role</span>
          <select
            className={selectClass}
            value={rule.role ?? ''}
            onChange={(e) => {
              const role = e.target.value || undefined
              // A batch only means something for students, so it goes with them.
              onChange({ ...rule, role, batch_year: batchApplies(role) ? rule.batch_year : undefined })
            }}
          >
            {roleOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
        {canRemove && (
          <button
            type="button"
            onClick={onRemove}
            aria-label={`Remove group ${index + 1}`}
            className="flex size-10 shrink-0 items-center justify-center rounded-lg text-ink-3 hover:bg-well hover:text-ink"
          >
            <X aria-hidden="true" className="size-4" />
          </button>
        )}
      </div>
      {!batchAllowed && (
        <span id={`${id}-batch`} className="text-xs text-ink-3">
          Batch applies to students only.
        </span>
      )}
    </fieldset>
  )
}

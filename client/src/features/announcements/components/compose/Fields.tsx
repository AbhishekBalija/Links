import { useId, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

// The composer's labelled fields and its segmented choices.
export function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold text-ink-2">{label}</span>
      {children}
      {error && <p className="text-[13px] font-semibold text-danger">{error}</p>}
    </div>
  )
}

type InputProps = { id: string; 'aria-invalid'?: boolean; 'aria-describedby'?: string }

export function TextField({ label, error, hint, input }: { label: string; error?: string; hint?: string; input: (props: InputProps) => ReactNode }) {
  const id = useId()
  const describedBy = [error && `${id}-error`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-[13px] font-semibold text-ink-2">
        {label}
      </label>
      {input({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy })}
      {error && (
        <p id={`${id}-error`} className="text-[13px] font-semibold text-danger">
          {error}
        </p>
      )}
      {hint && (
        <p id={`${id}-hint`} className="hidden text-xs text-ink-3 lg:block">
          {hint}
        </p>
      )}
    </div>
  )
}

export function Segmented({ label, options, value, onChange, className, full }: {
  label: string
  options: { value: string; label: string; short?: string }[]
  value: string
  onChange: (value: string) => void
  className?: string
  full?: boolean
}) {
  return (
    <div role="group" aria-label={label} className={cn('flex gap-1 self-start rounded-[10px] bg-well p-1', full && 'self-stretch', className)}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={option.value === value}
          onClick={() => onChange(option.value)}
          className={cn(
            'min-h-9 rounded-[7px] px-4 text-sm',
            full && 'flex-1 px-2',
            option.value === value ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2 hover:text-ink',
          )}
        >
          {full && option.short ? option.short : option.label}
        </button>
      ))}
    </div>
  )
}

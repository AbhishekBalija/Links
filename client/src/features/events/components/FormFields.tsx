import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { errorRing } from '../../announcements/components/compose/styles'

// The event forms' labelled fields: shared by proposing and by editing a
// published event's details.

export function FieldError({ message, id }: { message?: string; id?: string }) {
  if (!message) return null
  return (
    <p id={id} className="text-[13px] font-semibold text-danger">
      {message}
    </p>
  )
}

type InputProps = { id: string; 'aria-invalid'?: boolean; 'aria-describedby'?: string }

export function Labelled({ label, optional, error, children }: { label: string; optional?: boolean; error?: string; children: (props: InputProps) => ReactNode }) {
  const id = `f-${label.toLowerCase().replace(/\W+/g, '-')}`
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-[13px] font-semibold text-ink-2">
        {label}
        {optional && <span className="font-normal text-ink-3"> (optional)</span>}
      </label>
      {children({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': error ? `${id}-error` : undefined })}
      <FieldError id={`${id}-error`} message={error} />
    </div>
  )
}

const pickerClass = 'min-h-[46px] rounded-[10px] border border-line bg-surface px-3 font-mono text-[15px] text-ink outline-none focus-visible:border-rust lg:bg-paper'

export function DateTime({ label, date, time, onDate, onTime, error, after }: {
  label: string
  date: string
  time: string
  onDate: (v: string) => void
  onTime: (v: string) => void
  error?: string
  after?: ReactNode
}) {
  const id = `f-${label.toLowerCase()}`
  return (
    <fieldset className="flex flex-col gap-2" aria-describedby={error ? `${id}-error` : undefined}>
      <legend className="mb-2 text-[13px] font-semibold text-ink-2">{label}</legend>
      <div className="flex flex-wrap items-center gap-2.5 lg:gap-3">
        <input
          type="date"
          aria-label={`${label} date`}
          aria-invalid={error ? true : undefined}
          value={date}
          onChange={(e) => onDate(e.target.value)}
          className={cn(pickerClass, 'min-w-0 flex-1 lg:w-[220px] lg:flex-none', error && errorRing)}
        />
        <input
          type="time"
          aria-label={`${label} time`}
          aria-invalid={error ? true : undefined}
          value={time}
          onChange={(e) => onTime(e.target.value)}
          className={cn(pickerClass, 'w-[148px] lg:w-[150px]', error && errorRing)}
        />
        {after && <span className="basis-full px-1 lg:basis-auto lg:px-0">{after}</span>}
      </div>
      <FieldError id={`${id}-error`} message={error} />
    </fieldset>
  )
}

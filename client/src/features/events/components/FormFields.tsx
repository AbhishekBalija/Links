import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Segmented } from '../../announcements/components/compose/Fields'
import { errorRing, inputClass } from '../../announcements/components/compose/styles'
import { durationLabel, type ProposalErrors, type ProposalForm } from '../proposal'

type FieldsProps = {
  form: ProposalForm
  errors: ProposalErrors
  set: <K extends keyof ProposalForm>(key: K, value: ProposalForm[K]) => void
}

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

// WhenAndWhere is the part of an event that can still change after it is
// published: start, end, place and description.
export function WhenAndWhere({ form, errors, set, between }: FieldsProps & { between?: ReactNode }) {
  const duration = durationLabel(form)
  return (
    <>
      <DateTime label="Starts" date={form.startDate} time={form.startTime} onDate={(v) => set('startDate', v)} onTime={(v) => set('startTime', v)} error={errors.starts} />
      <DateTime
        label="Ends"
        date={form.endDate}
        time={form.endTime}
        onDate={(v) => set('endDate', v)}
        onTime={(v) => set('endTime', v)}
        error={errors.ends}
        after={duration && <span className="font-mono text-[13px] text-ink-3">{duration}</span>}
      />
      <Labelled label="Where" error={errors.location}>
        {(props) => (
          <input
            {...props}
            type="text"
            maxLength={200}
            value={form.location}
            onChange={(e) => set('location', e.target.value)}
            className={cn(inputClass, 'text-[15px]', errors.location && errorRing)}
          />
        )}
      </Labelled>
      {between}
      <Labelled label="About the event" optional error={errors.description}>
        {(props) => (
          <textarea
            {...props}
            rows={5}
            maxLength={5000}
            value={form.description}
            onChange={(e) => set('description', e.target.value)}
            className={cn(inputClass, 'resize-y font-serif text-[17px] leading-relaxed text-prose lg:text-lg', errors.description && errorRing)}
          />
        )}
      </Labelled>
    </>
  )
}

// SeatLimit switches between no limit and a number of seats for Going.
export function SeatLimit({ form, errors, set, heading, hint }: FieldsProps & { heading: ReactNode; hint: ReactNode }) {
  return (
    <>
      <div className="flex items-center justify-between gap-3">
        {heading}
        <Segmented
          label="Seats"
          options={[
            { value: 'none', label: 'No limit' },
            { value: 'limit', label: 'Limit' },
          ]}
          value={form.limitSeats ? 'limit' : 'none'}
          onChange={(v) => set('limitSeats', v === 'limit')}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center gap-2.5">
          {form.limitSeats && (
            <input
              type="text"
              inputMode="numeric"
              aria-label="Seat limit"
              aria-invalid={errors.capacity ? true : undefined}
              value={form.capacity}
              onChange={(e) => set('capacity', e.target.value)}
              className={cn('min-h-11 w-[110px] shrink-0 rounded-lg border border-input bg-paper px-3 font-mono text-[15px] outline-none focus-visible:border-rust', errors.capacity && errorRing)}
            />
          )}
          {hint}
        </div>
        <FieldError message={errors.capacity} />
      </div>
    </>
  )
}

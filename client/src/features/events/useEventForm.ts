import { useEffect, useRef, useState } from 'react'
import { useBlocker } from 'react-router-dom'
import type { ProposalErrors, ProposalForm } from './proposal'

// Which error a field's change clears.
function errorFor(key: keyof ProposalForm): keyof ProposalErrors {
  if (key === 'startDate' || key === 'startTime') return 'starts'
  if (key === 'endDate' || key === 'endTime') return 'ends'
  if (key === 'limitSeats') return 'capacity'
  return key as keyof ProposalErrors
}

// useEventForm holds an event form's values and errors, and guards against
// leaving with unsaved changes: the browser asks before closing the tab, and
// `blocker` lets the page ask before moving to another screen. Call
// `allowLeaving` just before navigating away after a save.
export function useEventForm(initialForm: () => ProposalForm) {
  const [initial] = useState(initialForm)
  const [form, setForm] = useState(initial)
  const [errors, setErrors] = useState<ProposalErrors>({})
  const leaving = useRef(false)

  const dirty = JSON.stringify(form) !== JSON.stringify(initial)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  function set<K extends keyof ProposalForm>(key: K, value: ProposalForm[K]) {
    setForm((f) => {
      const next = { ...f, [key]: value }
      // A new start date carries the end date along while they matched.
      if (key === 'startDate' && (f.endDate === '' || f.endDate === f.startDate)) next.endDate = value as string
      return next
    })
    setErrors((e) => ({ ...e, [errorFor(key)]: undefined }))
  }

  return {
    form,
    set,
    errors,
    setErrors,
    blocker,
    allowLeaving: () => {
      leaving.current = true
    },
  }
}

// serverErrors maps a 400's field details onto the form's errors. The
// server's messages are lower-case fragments ("must be in the future").
export function serverErrors(details: Record<string, unknown>): ProposalErrors {
  const fields: Record<string, keyof ProposalErrors> = {
    title: 'title',
    description: 'description',
    event_type: 'event_type',
    location: 'location',
    starts_at: 'starts',
    ends_at: 'ends',
    capacity: 'capacity',
    audience: 'audience',
  }
  const errors: ProposalErrors = {}
  for (const [field, message] of Object.entries(details)) {
    const key = fields[field]
    if (key && typeof message === 'string') errors[key] = message.charAt(0).toUpperCase() + message.slice(1) + (message.endsWith('.') ? '' : '.')
  }
  return errors
}

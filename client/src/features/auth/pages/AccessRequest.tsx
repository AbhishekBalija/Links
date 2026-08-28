import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '../../../components/ui/button'
import { requestAccess } from '../api'
import { ApiRequestError } from '../../../shared/api/types'

const DEPARTMENTS = [
  { code: 'CS', name: 'Computer Science' },
  { code: 'AD', name: 'AI & Data Science' },
  { code: 'CI', name: 'Computer Engineering (CI)' },
  { code: 'CV', name: 'Civil Engineering' },
  { code: 'ME', name: 'Mechanical Engineering' },
  { code: 'EC', name: 'Electronics & Communication' },
]

type FormData = {
  email: string
  password: string
  full_name: string
  usn: string
  department_code: string
  phone: string
}

type FormErrors = Partial<Record<keyof FormData, string>>

export default function AccessRequest() {
  const [form, setForm] = useState<FormData>({
    email: '',
    password: '',
    full_name: '',
    usn: '',
    department_code: '',
    phone: '',
  })
  const [errors, setErrors] = useState<FormErrors>({})
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle')
  const [serverError, setServerError] = useState<string | null>(null)

  function validate(): FormErrors {
    const errs: FormErrors = {}
    if (!form.full_name.trim()) errs.full_name = 'Full name is required'
    if (!form.email.trim()) errs.email = 'Email is required'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) errs.email = 'Enter a valid email'
    if (!form.password) errs.password = 'Password is required'
    else if (form.password.length < 8) errs.password = 'Password must be at least 8 characters'
    else if (!/[A-Z]/.test(form.password)) errs.password = 'Password must contain an uppercase letter'
    else if (!/[a-z]/.test(form.password)) errs.password = 'Password must contain a lowercase letter'
    else if (!/[0-9]/.test(form.password)) errs.password = 'Password must contain a digit'
    if (form.usn && !/^4MN\d{2}(CS|AD|CI|CV|ME|EC)\d{3}$/i.test(form.usn)) {
      errs.usn = 'USN format: 4MN<year><dept><roll> (e.g., 4MN22CS001)'
    }
    if (!form.department_code) errs.department_code = 'Select your department'
    return errs
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const errs = validate()
    setErrors(errs)
    if (Object.keys(errs).length > 0) return

    setStatus('loading')
    setServerError(null)

    try {
      await requestAccess(form)
      setStatus('success')
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setServerError(err.message)
      } else {
        setServerError('Something went wrong. Please try again.')
      }
      setStatus('error')
    }
  }

  function update(field: keyof FormData, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) setErrors((prev) => ({ ...prev, [field]: undefined }))
  }

  if (status === 'success') {
    return (
      <div className="flex min-h-dvh items-center justify-center p-4">
        <div className="w-full max-w-sm bg-white border border-border rounded-lg p-8 shadow-sm text-center">
          <h1 className="text-xl font-semibold text-foreground mb-2">Access requested</h1>
          <p className="text-sm text-muted-foreground mb-2">
            Your request has been submitted. An admin or HOD will review your details and verify your account.
          </p>
          <p className="text-sm text-muted-foreground mb-6">
			When your request is approved, you will receive an email link to set your password and activate your account.
          </p>
          <Link to="/login">
            <Button>Go to login</Button>
          </Link>
        </div>
      </div>
    )
  }

  const inputClass = "block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <div className="w-full max-w-sm bg-white border border-border rounded-lg p-8 shadow-sm">
        <h1 className="text-xl font-semibold text-foreground mb-1">Request access</h1>
        <p className="text-sm text-muted-foreground mb-6">Submit your details for verification by an admin or HOD.</p>

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div>
            <label htmlFor="full_name" className="block text-sm font-medium text-foreground mb-1.5">Full name</label>
            <input id="full_name" type="text" value={form.full_name} onChange={(e) => update('full_name', e.target.value)} className={inputClass} autoComplete="name" />
            {errors.full_name && <p className="mt-1 text-xs text-destructive">{errors.full_name}</p>}
          </div>

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-foreground mb-1.5">Email (Gmail)</label>
            <input id="email" type="email" value={form.email} onChange={(e) => update('email', e.target.value)} className={inputClass} autoComplete="email" />
            {errors.email && <p className="mt-1 text-xs text-destructive">{errors.email}</p>}
          </div>

          <div>
            <label htmlFor="phone" className="block text-sm font-medium text-foreground mb-1.5">Phone (optional)</label>
            <input id="phone" type="tel" value={form.phone} onChange={(e) => update('phone', e.target.value)} className={inputClass} autoComplete="tel" />
          </div>

          <div>
            <label htmlFor="usn" className="block text-sm font-medium text-foreground mb-1.5">USN</label>
            <input id="usn" type="text" value={form.usn} onChange={(e) => update('usn', e.target.value)} placeholder="4MN22CS001" className={inputClass} autoComplete="off" />
            {errors.usn && <p className="mt-1 text-xs text-destructive">{errors.usn}</p>}
          </div>

          <div>
            <label htmlFor="department_code" className="block text-sm font-medium text-foreground mb-1.5">Department</label>
            <select id="department_code" value={form.department_code} onChange={(e) => update('department_code', e.target.value)} className={inputClass}>
              <option value="">Select department</option>
              {DEPARTMENTS.map((d) => (
                <option key={d.code} value={d.code}>{d.name} ({d.code})</option>
              ))}
            </select>
            {errors.department_code && <p className="mt-1 text-xs text-destructive">{errors.department_code}</p>}
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-foreground mb-1.5">Password</label>
            <input id="password" type="password" value={form.password} onChange={(e) => update('password', e.target.value)} className={inputClass} autoComplete="new-password" />
            {errors.password && <p className="mt-1 text-xs text-destructive">{errors.password}</p>}
          </div>

          {serverError && (
            <div className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2 text-sm text-destructive">{serverError}</div>
          )}

          <Button type="submit" disabled={status === 'loading'} className="w-full">
            {status === 'loading' ? 'Submitting…' : 'Submit request'}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Already have an account?{' '}
          <Link to="/login" className="text-primary hover:underline">Log in</Link>
        </p>
      </div>
    </div>
  )
}

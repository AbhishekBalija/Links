import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Button, buttonVariants } from '../../../components/ui/button'
import { ApiRequestError } from '../../../shared/api/types'
import { activateAccount } from '../api'

type FormErrors = {
  password?: string
  confirmation?: string
}

export default function ActivateAccount() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [errors, setErrors] = useState<FormErrors>({})
  const [serverError, setServerError] = useState('')
  const [status, setStatus] = useState<'idle' | 'loading' | 'success'>('idle')

  function validate(): FormErrors {
    const nextErrors: FormErrors = {}
    if (!password) nextErrors.password = 'Password is required'
    else if (password.length < 8) nextErrors.password = 'Password must be at least 8 characters'
    else if (!/[A-Z]/.test(password)) nextErrors.password = 'Password must contain an uppercase letter'
    else if (!/[a-z]/.test(password)) nextErrors.password = 'Password must contain a lowercase letter'
    else if (!/[0-9]/.test(password)) nextErrors.password = 'Password must contain a digit'
    if (confirmation !== password) nextErrors.confirmation = 'Passwords do not match'
    return nextErrors
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const nextErrors = validate()
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    setStatus('loading')
    setServerError('')
    try {
      await activateAccount({ token, password })
      setStatus('success')
    } catch (error) {
      setServerError(error instanceof ApiRequestError ? error.message : 'Something went wrong. Please try again.')
      setStatus('idle')
    }
  }

  if (!token) {
    return (
      <div className="flex min-h-dvh items-center justify-center p-4">
        <div className="w-full max-w-sm rounded-lg border border-border bg-white p-8 text-center shadow-sm">
          <h1 className="mb-2 text-xl font-semibold text-foreground">Invalid activation link</h1>
          <p className="mb-6 text-sm text-muted-foreground">This link is missing its activation token. Open the complete link from your email.</p>
          <Link to="/login" className={buttonVariants()}>Go to login</Link>
        </div>
      </div>
    )
  }

  if (status === 'success') {
    return (
      <div className="flex min-h-dvh items-center justify-center p-4">
        <div className="w-full max-w-sm rounded-lg border border-border bg-white p-8 text-center shadow-sm">
          <h1 className="mb-2 text-xl font-semibold text-foreground">Account activated</h1>
          <p className="mb-6 text-sm text-muted-foreground">Your password is set and your account is ready.</p>
          <Link to="/login" className={buttonVariants()}>Log in</Link>
        </div>
      </div>
    )
  }

  const inputClass = 'block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20'

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <div className="w-full max-w-sm rounded-lg border border-border bg-white p-8 shadow-sm">
        <h1 className="mb-1 text-xl font-semibold text-foreground">Activate your account</h1>
        <p className="mb-6 text-sm text-muted-foreground">Choose the password you will use to log in to LINKS.</p>

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div>
            <label htmlFor="password" className="mb-1.5 block text-sm font-medium text-foreground">New password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(event) => { setPassword(event.target.value); setErrors((current) => ({ ...current, password: undefined })) }}
              className={inputClass}
              autoComplete="new-password"
            />
            {errors.password && <p className="mt-1 text-xs text-destructive">{errors.password}</p>}
          </div>

          <div>
            <label htmlFor="password-confirmation" className="mb-1.5 block text-sm font-medium text-foreground">Confirm password</label>
            <input
              id="password-confirmation"
              type="password"
              value={confirmation}
              onChange={(event) => { setConfirmation(event.target.value); setErrors((current) => ({ ...current, confirmation: undefined })) }}
              className={inputClass}
              autoComplete="new-password"
            />
            {errors.confirmation && <p className="mt-1 text-xs text-destructive">{errors.confirmation}</p>}
          </div>

          {serverError && (
            <div role="alert" className="rounded-md border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {serverError}
            </div>
          )}

          <Button type="submit" disabled={status === 'loading'} className="w-full">
            {status === 'loading' ? 'Activating…' : 'Activate account'}
          </Button>
        </form>
      </div>
    </div>
  )
}

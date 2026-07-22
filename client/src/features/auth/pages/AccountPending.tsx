import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store'

export default function AccountPending() {
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <div className="w-full max-w-sm bg-card border border-border rounded-lg p-8 shadow-sm text-center">
        <h1 className="text-xl font-semibold text-foreground mb-2">Account setup incomplete</h1>
        <p className="text-sm text-muted-foreground mb-1">
          Your account has been created, but no role has been assigned yet.
        </p>
        <p className="text-sm text-muted-foreground mb-6">
          Please contact an administrator or HOD to complete your account setup.
        </p>
        <button
          type="button"
          onClick={handleLogout}
          className="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
        >
          Log out
        </button>
      </div>
    </div>
  )
}

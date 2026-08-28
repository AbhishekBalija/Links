import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../store'

export function ProtectedRoute() {
  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) {
    return <div className="flex min-h-dvh items-center justify-center text-muted-foreground text-sm">Loading…</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (user && user.roles.length === 0) {
    return <Navigate to="/account-pending" replace />
  }

  return <Outlet />
}

export function GuestRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) {
    return <div className="flex min-h-dvh items-center justify-center text-muted-foreground text-sm">Loading…</div>
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}

export function PendingRoute() {
  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) {
    return <div className="flex min-h-dvh items-center justify-center text-muted-foreground text-sm">Loading…</div>
  }

  if (isAuthenticated && user && user.roles.length > 0) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}

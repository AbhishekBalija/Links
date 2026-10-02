import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../store'
import { PageLoading } from '../../../shared/ui/states'

export function ProtectedRoute() {
  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) return <PageLoading />
  if (!isAuthenticated) return <Navigate to="/login" replace />
  if (user && user.roles.length === 0) return <Navigate to="/account-pending" replace />
  return <Outlet />
}

// GuestRoute is for the sign-in screens. Signing in moves on from them: to
// "You're signed in as" on a first sign-in, to Home otherwise.
export function GuestRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)
  const firstSignIn = useAuthStore((s) => s.firstSignIn)

  if (isLoading) return <PageLoading />
  if (isAuthenticated) return <Navigate to={firstSignIn ? '/welcome' : '/'} replace />
  return <Outlet />
}

// PendingRoute is for someone signed in whose account has no role.
export function PendingRoute() {
  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isLoading = useAuthStore((s) => s.isLoading)

  if (isLoading) return <PageLoading />
  if (!isAuthenticated) return <Navigate to="/login" replace />
  if (user && user.roles.length > 0) return <Navigate to="/" replace />
  return <Outlet />
}

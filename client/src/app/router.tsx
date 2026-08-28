import { Routes, Route, Navigate } from 'react-router-dom'
import { ProtectedRoute, GuestRoute, PendingRoute } from '../features/auth/components/ProtectedRoute'
import Login from '../features/auth/pages/Login'
import AccessRequest from '../features/auth/pages/AccessRequest'
import AccountPending from '../features/auth/pages/AccountPending'
import ActivateAccount from '../features/auth/pages/ActivateAccount'
import Dashboard from '../features/dashboard/pages/Dashboard'
import EditProfile from '../features/profiles/pages/EditProfile'

export function AppRouter() {
  return (
    <Routes>
      <Route element={<GuestRoute />}>
        <Route path="/login" element={<Login />} />
        <Route path="/access-request" element={<AccessRequest />} />
        <Route path="/activate" element={<ActivateAccount />} />
      </Route>
      <Route element={<PendingRoute />}>
        <Route path="/account-pending" element={<AccountPending />} />
      </Route>
      <Route element={<ProtectedRoute />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/profile/edit" element={<EditProfile />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { ProtectedRoute, GuestRoute, PendingRoute } from '../features/auth/components/ProtectedRoute'
import Login from '../features/auth/pages/Login'
import AccessRequest from '../features/auth/pages/AccessRequest'
import AccountPending from '../features/auth/pages/AccountPending'
import ActivateAccount from '../features/auth/pages/ActivateAccount'
import Home from '../features/home/pages/Home'
import Notices from '../features/notices/pages/Notices'
import NoticeDetail from '../features/notices/pages/NoticeDetail'
import ApprovalQueue from '../features/announcements/pages/ApprovalQueue'
import Compose from '../features/announcements/pages/Compose'
import MyAnnouncement from '../features/announcements/pages/MyAnnouncement'
import MyAnnouncements from '../features/announcements/pages/MyAnnouncements'
import { useAuthStore } from '../features/auth/store'
import { AppShell } from './shell/AppShell'
import { canApprove, canPost } from './shell/nav'
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
        <Route element={<AppShell />}>
          <Route path="/" element={<Home />} />
          <Route path="/notices" element={<Notices />} />
          <Route path="/notices/:id" element={<NoticeDetail />} />
          <Route path="/mine" element={<MyAnnouncements />} />
          <Route path="/mine/:id" element={<MyAnnouncement />} />
          <Route element={<ApproverRoute />}>
            <Route path="/approvals" element={<ApprovalQueue />} />
            <Route path="/approvals/:id" element={<ApprovalQueue />} />
          </Route>
          <Route element={<PosterRoute />}>
            <Route path="/mine/new" element={<Compose />} />
            <Route path="/mine/:id/edit" element={<Compose />} />
          </Route>
          <Route path="/profile/edit" element={<EditProfile />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

// PosterRoute keeps the composer to roles that can post. The server checks
// again; this only saves others a screen that would refuse them.
function PosterRoute() {
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  return canPost(roles) ? <Outlet /> : <Navigate to="/mine" replace />
}

// ApproverRoute keeps the queue to HODs, the principal and admins. The server
// decides what each of them may approve.
function ApproverRoute() {
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  return canApprove(roles) ? <Outlet /> : <Navigate to="/" replace />
}

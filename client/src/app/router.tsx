import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { ProtectedRoute, GuestRoute, PendingRoute } from '../features/auth/components/ProtectedRoute'
import { useAuthStore } from '../features/auth/store'
import { AppShell } from './shell/AppShell'
import { canApprove, canPost } from './shell/nav'
import { PageLoading } from '../shared/ui/states'

// Each page is its own chunk, so the first visit downloads only the screen
// it opens, which matters most for students on phones.
const Login = lazy(() => import('../features/auth/pages/Login'))
const AccessRequest = lazy(() => import('../features/auth/pages/AccessRequest'))
const AccountPending = lazy(() => import('../features/auth/pages/AccountPending'))
const ActivateAccount = lazy(() => import('../features/auth/pages/ActivateAccount'))
const Home = lazy(() => import('../features/home/pages/Home'))
const Notices = lazy(() => import('../features/notices/pages/Notices'))
const NoticeDetail = lazy(() => import('../features/notices/pages/NoticeDetail'))
const ApprovalQueue = lazy(() => import('../features/announcements/pages/ApprovalQueue'))
const Compose = lazy(() => import('../features/announcements/pages/Compose'))
const MyAnnouncement = lazy(() => import('../features/announcements/pages/MyAnnouncement'))
const MyAnnouncements = lazy(() => import('../features/announcements/pages/MyAnnouncements'))
const EditProfile = lazy(() => import('../features/profiles/pages/EditProfile'))
const People = lazy(() => import('../features/people/pages/People'))
const Profile = lazy(() => import('../features/people/pages/Profile'))
const MyProfile = lazy(() => import('../features/people/pages/MyProfile'))
const Department = lazy(() => import('../features/people/pages/Department'))

export function AppRouter() {
  return (
    <Suspense fallback={<PageLoading />}>
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
          <Route path="/people" element={<People />} />
          <Route path="/people/:username" element={<Profile />} />
          <Route path="/departments/:code" element={<Department />} />
          <Route path="/profile" element={<MyProfile />} />
          <Route path="/profile/edit" element={<EditProfile />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
    </Suspense>
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

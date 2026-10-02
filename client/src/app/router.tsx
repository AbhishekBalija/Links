import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { ProtectedRoute, GuestRoute, PendingRoute } from '../features/auth/components/ProtectedRoute'
import { useAuthStore } from '../features/auth/store'
import { AppShell } from './shell/AppShell'
import { canApprove, canPost, isPlacementStaff } from './shell/nav'
import { PageLoading } from '../shared/ui/states'

// Each page is its own chunk, so the first visit downloads only the screen
// it opens, which matters most for students on phones.
const SignIn = lazy(() => import('../features/auth/pages/SignIn'))
const FirstSignIn = lazy(() => import('../features/auth/pages/FirstSignIn'))
const AccountPending = lazy(() => import('../features/auth/pages/AccountPending'))
const Home = lazy(() => import('../features/home/pages/Home'))
const Notices = lazy(() => import('../features/notices/pages/Notices'))
const NoticeDetail = lazy(() => import('../features/notices/pages/NoticeDetail'))
const ApprovalQueue = lazy(() => import('../features/announcements/pages/ApprovalQueue'))
const Compose = lazy(() => import('../features/announcements/pages/Compose'))
const MyAnnouncement = lazy(() => import('../features/announcements/pages/MyAnnouncement'))
const MyPosts = lazy(() => import('../features/posts/pages/MyPosts'))
const EditProfile = lazy(() => import('../features/profiles/pages/EditProfile'))
const Events = lazy(() => import('../features/events/pages/Events'))
const EventDetail = lazy(() => import('../features/events/pages/EventDetail'))
const EventEdit = lazy(() => import('../features/events/pages/EventEdit'))
const EventPeople = lazy(() => import('../features/events/pages/EventPeople'))
const MyEvent = lazy(() => import('../features/events/pages/MyEvent'))
const Propose = lazy(() => import('../features/events/pages/Propose'))
const Jobs = lazy(() => import('../features/jobs/pages/Jobs'))
const JobDetail = lazy(() => import('../features/jobs/pages/JobDetail'))
const Placement = lazy(() => import('../features/placement/pages/Placement'))
const PlacementOpportunity = lazy(() => import('../features/placement/pages/PlacementOpportunity'))
const Applicants = lazy(() => import('../features/placement/pages/Applicants'))
const OpportunityForm = lazy(() => import('../features/placement/pages/OpportunityForm'))
const People = lazy(() => import('../features/people/pages/People'))
const Profile = lazy(() => import('../features/people/pages/Profile'))
const MyProfile = lazy(() => import('../features/people/pages/MyProfile'))
const Department = lazy(() => import('../features/people/pages/Department'))

export function AppRouter() {
  return (
    <Suspense fallback={<PageLoading />}>
    <Routes>
      <Route element={<GuestRoute />}>
        <Route path="/login" element={<SignIn />} />
      </Route>
      {/* Old links from before passwordless sign-in (spec #129). */}
      <Route path="/access-request" element={<Navigate to="/login" replace />} />
      <Route path="/activate" element={<Navigate to="/login" replace />} />
      <Route element={<PendingRoute />}>
        <Route path="/account-pending" element={<AccountPending />} />
      </Route>
      <Route element={<ProtectedRoute />}>
        <Route path="/welcome" element={<FirstSignIn />} />
        <Route element={<AppShell />}>
          <Route path="/" element={<Home />} />
          <Route path="/notices" element={<Notices />} />
          <Route path="/notices/:id" element={<NoticeDetail />} />
          <Route path="/mine" element={<MyPosts />} />
          <Route path="/mine/:id" element={<MyAnnouncement />} />
          <Route path="/mine/events/:id" element={<MyEvent />} />
          <Route element={<ApproverRoute />}>
            <Route path="/approvals" element={<ApprovalQueue />} />
            <Route path="/approvals/:id" element={<ApprovalQueue />} />
          </Route>
          <Route element={<PosterRoute />}>
            <Route path="/mine/new" element={<Compose />} />
            <Route path="/mine/:id/edit" element={<Compose />} />
            <Route path="/mine/events/new" element={<Propose />} />
            <Route path="/mine/events/:id/edit" element={<Propose />} />
          </Route>
          <Route path="/events" element={<Events />} />
          <Route path="/events/:id" element={<EventDetail />} />
          <Route path="/events/:id/people" element={<EventPeople />} />
          <Route path="/events/:id/edit" element={<EventEdit />} />
          <Route path="/jobs" element={<Jobs />} />
          <Route path="/jobs/:id" element={<JobDetail />} />
          <Route element={<PlacementRoute />}>
            <Route path="/placement" element={<Placement />} />
            <Route path="/placement/new" element={<OpportunityForm />} />
            <Route path="/placement/:id" element={<PlacementOpportunity />} />
            <Route path="/placement/:id/edit" element={<OpportunityForm />} />
            <Route path="/placement/:id/applicants" element={<Applicants />} />
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

// PosterRoute keeps the composer and the proposal form to roles that can
// post (the same roles propose events). The server checks
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

// PlacementRoute keeps Placement to the placement officer, the principal and
// admins. The server checks again on every request.
function PlacementRoute() {
  const roles = useAuthStore((s) => s.user?.roles) ?? []
  return isPlacementStaff(roles) ? <Outlet /> : <Navigate to="/" replace />
}

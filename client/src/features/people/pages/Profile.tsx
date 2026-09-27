import { Link, Navigate, useParams } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useAuthStore } from '../../auth/store'
import { useProfile } from '../api'
import { BackLink } from '../components/BackLink'
import { ProfileSkeleton, ProfileView } from '../components/ProfileView'

// Profile is another member's page, opened from People or a Department.
export default function Profile() {
  const { username = '' } = useParams()
  const me = useAuthStore((s) => s.user?.profile.username)
  const profile = useProfile(username)

  if (username === me) return <Navigate to="/profile" replace />

  return (
    <div className="flex flex-col gap-3 lg:gap-[22px]">
      <BackLink />
      {profile.isPending ? (
        <>
          <LoadingStatus label="Loading profile" />
          <ProfileSkeleton />
        </>
      ) : profile.isError ? (
        profile.error instanceof ApiRequestError && profile.error.status === 404 ? (
          <div className="max-w-[640px]">
            <EmptyState title="This profile isn't available">
              <p>They may have made it private, or the link is out of date.</p>
              <Link to="/people" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
                Search People
              </Link>
            </EmptyState>
          </div>
        ) : (
          <ErrorState message="This profile could not be loaded." onRetry={() => profile.refetch()} />
        )
      ) : (
        <ProfileView profile={profile.data} />
      )}
    </div>
  )
}

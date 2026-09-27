import { Link } from 'react-router-dom'
import { ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { useAuthStore } from '../../auth/store'
import { useProfile } from '../api'
import { ProfileSkeleton, ProfileView } from '../components/ProfileView'

// MyProfile shows the member their own page as others see it, with the one
// action they came for: Edit profile.
export default function MyProfile() {
  const username = useAuthStore((s) => s.user?.profile.username)
  const profile = useProfile(username)
  const edit = (
    <Link to="/profile/edit" className={buttonStyles.primary + ' flex-none'}>
      Edit profile
    </Link>
  )

  return (
    <div className="flex flex-col gap-3.5 lg:gap-[22px]">
      <header className="flex items-center justify-between gap-4 px-1 lg:hidden">
        <h1 className="font-serif text-[28px] font-medium">Profile</h1>
        {edit}
      </header>

      {profile.isPending ? (
        <>
          <LoadingStatus label="Loading your profile" />
          <ProfileSkeleton />
        </>
      ) : profile.isError ? (
        <ErrorState message="Your profile could not be loaded." onRetry={() => profile.refetch()} />
      ) : (
        <>
          <p className="mx-1 text-[13px] text-ink-3 lg:hidden">
            {profile.data.public_profile_enabled ? 'This is how others see you in People.' : <PrivateNote />}
          </p>
          <ProfileView
            profile={profile.data}
            own
            action={
              <>
                <span className="text-[13px] text-ink-3">
                  {profile.data.public_profile_enabled ? 'Your profile, as others see it in People' : <PrivateNote />}
                </span>
                {edit}
              </>
            }
          />
        </>
      )}
    </div>
  )
}

function PrivateNote() {
  return <>Your profile is private: only you can see it, and you aren't listed in People.</>
}

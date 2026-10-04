import { ChevronLeft } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useDashboard } from '../../home/api'
import { StaffForm } from '../components/StaffForm'

// HODStaff is an HOD's Add faculty page, reached from their Home: faculty
// for their own department only (ADR 0029).
export default function HODStaff() {
  const dashboard = useDashboard()
  const department = dashboard.data?.user.department

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <Link to="/" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        Home
      </Link>
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Add faculty</h1>
        {department && <p className="text-[15px] text-ink-2">{department.name} staff sign in once you add them.</p>}
      </header>
      {dashboard.isPending ? (
        <LoadingStatus label="Loading" />
      ) : dashboard.isError || !department ? (
        <ErrorState message="Your department could not be loaded." onRetry={() => dashboard.refetch()} />
      ) : (
        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:gap-6">
          <StaffForm
            title="Add a faculty member"
            roles={['faculty']}
            departments={[department]}
            withHOD={new Set()}
            fixedDepartment={department}
            requestsPath="/approvals/access"
          />
          <aside className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-5 lg:px-6 lg:py-[22px]">
            <h2 className="font-serif text-xl leading-tight font-medium">Your department's staff</h2>
            <p className="text-sm leading-[1.55] text-ink-2">
              You add faculty for {department.name}. The office adds HODs, the principal and placement officers.
            </p>
            <p className="text-sm leading-[1.55] text-ink-2">To make a student a coordinator, open their profile in People.</p>
          </aside>
        </div>
      )}
    </div>
  )
}

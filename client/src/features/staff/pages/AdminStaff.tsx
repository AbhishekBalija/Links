import { useSearchParams } from 'react-router-dom'
import { ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { WorkspaceTabs } from '../../access/components/WorkspaceTabs'
import { adminTabs } from '../../access/adminTabs'
import { useDepartments } from '../../announcements/api'
import { useDashboard } from '../../home/api'
import { StaffForm } from '../components/StaffForm'

// AdminStaff is the Add staff tab of the Admin workspace. Home's "No HOD" row
// opens it with ?role=hod&department=CODE already chosen.
export default function AdminStaff() {
  const [params] = useSearchParams()
  const departments = useDepartments()
  const dashboard = useDashboard()
  const withHOD = new Set((dashboard.data?.college?.departments ?? []).filter((d) => d.hod).map((d) => d.code))
  const initialDepartmentId = departments.data?.find((d) => d.code === params.get('department'))?.id

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Admin</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">People, class lists and staff for the whole college.</p>
      </header>
      <WorkspaceTabs label="Admin" active="Add staff" tabs={adminTabs} />
      {departments.isPending || dashboard.isPending ? (
        <LoadingStatus label="Loading" />
      ) : departments.isError ? (
        <ErrorState message="The departments could not be loaded." onRetry={() => departments.refetch()} />
      ) : (
        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] lg:gap-6">
          <StaffForm
            title="Add a staff member"
            roles={['hod', 'faculty', 'placement_officer', 'principal', 'admin']}
            departments={departments.data}
            withHOD={withHOD}
            initialRole={params.get('role') ?? undefined}
            initialDepartmentId={initialDepartmentId}
            requestsPath="/admin/requests"
          />
          <aside className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-5 lg:px-6 lg:py-[22px]">
            <h2 className="font-serif text-xl leading-tight font-medium">How staff get in</h2>
            <p className="text-sm leading-[1.55] text-ink-2">
              Add them here. LINKS emails them, and the first time they sign in with that email they're in with the right role.
            </p>
            <p className="text-sm leading-[1.55] text-ink-2">Add the principal and each HOD first. HODs then add their own faculty.</p>
            <p className="text-sm leading-[1.55] text-ink-2">Someone already on LINKS? Give them the role from their profile in People.</p>
          </aside>
        </div>
      )}
    </div>
  )
}

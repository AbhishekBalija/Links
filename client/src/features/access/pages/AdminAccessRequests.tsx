import { AccessRequestsView } from '../components/AccessRequestsView'
import { adminTabs } from '../adminTabs'
import { WorkspaceTabs } from '../components/WorkspaceTabs'

// AdminAccessRequests is the Access requests tab of the Admin workspace:
// every Department's requests.
export default function AdminAccessRequests() {
  const header = (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Admin</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">People, class lists and staff for the whole college.</p>
      </header>
      <WorkspaceTabs label="Admin" active="Access requests" tabs={adminTabs} />
    </div>
  )
  return (
    <AccessRequestsView
      header={header}
      basePath="/admin/requests"
      backLabel="Admin"
      empty='Nobody is waiting to get in. Students on class lists sign in without asking; requests come from people on no list, or rows reported with "Not you?".'
    />
  )
}

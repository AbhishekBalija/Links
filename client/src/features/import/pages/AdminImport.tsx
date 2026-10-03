import { WorkspaceTabs } from '../../access/components/WorkspaceTabs'
import { adminTabs } from '../../access/adminTabs'
import { ImportScreen } from '../components/ImportScreen'

// AdminImport is the Import students tab of the Admin workspace: a class
// list for any Department, each USN deciding the Department and Batch.
export default function AdminImport() {
  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Admin</h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">People, class lists and staff for the whole college.</p>
      </header>
      <WorkspaceTabs label="Admin" active="Import students" tabs={adminTabs} />
      <ImportScreen />
    </div>
  )
}

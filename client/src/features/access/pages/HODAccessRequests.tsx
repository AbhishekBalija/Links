import { Link } from 'react-router-dom'
import { AccessRequestsView } from '../components/AccessRequestsView'
import { QueueTabs } from '../components/QueueTabs'

// HODAccessRequests is the Access requests tab of an HOD's Approval queue:
// only their own Department's requests (ADR 0025).
export default function HODAccessRequests() {
  const header = (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">
          <span className="lg:hidden">Approvals</span>
          <span className="hidden lg:inline">Approval queue</span>
        </h1>
        <p className="hidden text-[15px] text-ink-2 lg:block">Oldest first. Only your department's requests.</p>
      </header>
      <QueueTabs active="Access requests" />
    </div>
  )
  return (
    <AccessRequestsView
      header={header}
      basePath="/approvals/access"
      backLabel="Approvals"
      empty={
        <>
          Students on your class lists sign in without asking. Requests only come from people who aren't on a list, or who report a wrong row.{' '}
          <Link to="/" className="font-semibold">
            Back to Home
          </Link>
        </>
      }
    />
  )
}

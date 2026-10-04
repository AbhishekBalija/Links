import { ChevronLeft } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useDashboard } from '../../home/api'
import { NotSignedInScreen } from '../components/NotSignedInScreen'

// HODNotSignedIn is an HOD's own department's list, reached from their Home.
export default function HODNotSignedIn() {
  const dashboard = useDashboard()
  const department = dashboard.data?.user.department
  const waiting = dashboard.data?.lists?.waiting_count

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <Link to="/" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        Home
      </Link>
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">
          Not signed in yet {waiting !== undefined && <span className="font-mono text-xl text-rust">{waiting}</span>}
        </h1>
        {department && <p className="text-[15px] text-ink-2">{department.name} students and staff on your class lists and invites.</p>}
      </header>
      <NotSignedInScreen canFix />
    </div>
  )
}

import { ChevronLeft } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ErrorState, LoadingStatus } from '../../../shared/ui/states'
import { useDashboard } from '../../home/api'
import { ImportScreen } from '../components/ImportScreen'

// HODImport is an HOD's Import students page, reached from their Home. The
// import is held to their own Department, so the screen says so up front.
export default function HODImport() {
  const dashboard = useDashboard()
  const department = dashboard.data?.user.department

  return (
    <div className="flex flex-col gap-4 lg:gap-5">
      <Link to="/" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        Home
      </Link>
      <header className="flex flex-col gap-1 px-1 lg:px-0">
        <h1 className="font-serif text-[28px] font-medium tracking-[-0.4px] lg:text-[38px] lg:leading-tight lg:tracking-[-0.6px]">Import students</h1>
        <p className="text-[15px] text-ink-2">Add a class list so students can sign in.</p>
      </header>
      {dashboard.isPending ? (
        <LoadingStatus label="Loading" />
      ) : dashboard.isError || !department ? (
        <ErrorState message="Your department could not be loaded." onRetry={() => dashboard.refetch()} />
      ) : (
        <ImportScreen department={{ code: department.code, name: department.name }} />
      )}
    </div>
  )
}

import type { ReactNode } from 'react'
import { LoadingStatus, Skeleton } from '../../../../shared/ui/states'

export function QueueSkeleton({ header }: { header: ReactNode }) {
  return (
    <div className="flex flex-col gap-5">
      {header}
      <LoadingStatus label="Loading the approval queue" />
      <div className="grid gap-5 lg:grid-cols-[360px_minmax(0,1fr)]">
        <div className="flex flex-col gap-2.5">
          {Array.from({ length: 3 }, (_, i) => (
            <div key={i} className="flex flex-col gap-2 rounded-[10px] border border-line bg-surface p-4">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-4 w-11/12" />
              <Skeleton className="h-3 w-3/5" />
            </div>
          ))}
        </div>
        <div className="hidden flex-col gap-4 rounded-xl border border-line bg-surface p-9 lg:flex">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-9 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
        </div>
      </div>
    </div>
  )
}

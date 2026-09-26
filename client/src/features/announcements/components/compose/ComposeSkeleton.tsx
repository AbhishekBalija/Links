import { LoadingStatus, Skeleton } from '../../../../shared/ui/states'

export function ComposeSkeleton() {
  return (
    <div className="flex flex-col gap-5">
      <LoadingStatus label="Loading the composer" />
      <Skeleton className="h-9 w-64" />
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_400px]">
        <div className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-6">
          <Skeleton className="h-9 w-72" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
        <div className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-5">
          <Skeleton className="h-6 w-32" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    </div>
  )
}

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { JobState, MyApplication, Opportunity, OpportunityType } from './types'

const PAGE_SIZE = 20

type FeedMeta = { next_cursor?: string }

// useJobFeed loads the student's Opportunities a page at a time: open ones
// soonest deadline first, or what they applied to and closed ones, latest
// deadline first.
export function useJobFeed(state: JobState, type: OpportunityType | null) {
  return useInfiniteQuery({
    queryKey: ['jobs', 'feed', state, type ?? 'all'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE) })
      if (state !== 'open') params.set('state', state)
      if (type) params.set('type', type)
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<Opportunity[], FeedMeta>(`/api/v1/opportunities?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

export function useJob(id: string) {
  return useQuery({
    queryKey: ['jobs', 'detail', id],
    queryFn: ({ signal }) => apiRequest<Opportunity>(`/api/v1/opportunities/${encodeURIComponent(id)}`, { signal }),
  })
}

// useApplication applies to an Opportunity, or withdraws. Either returns the
// student's Application, which goes straight into the job; the lists and
// Home refresh in the background.
export function useApplication(id: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (action: 'apply' | 'withdraw') =>
      apiRequest<MyApplication>(`/api/v1/opportunities/${encodeURIComponent(id)}/${action}`, { method: 'POST' }),
    onSuccess: (application) => {
      client.setQueryData<Opportunity>(['jobs', 'detail', id], (job) => (job ? { ...job, my_application: application } : job))
      client.invalidateQueries({ queryKey: ['jobs', 'feed'] })
      client.invalidateQueries({ queryKey: ['dashboard'] })
    },
    // A refusal (closed meanwhile, already applied) means the job is stale.
    onError: () => client.invalidateQueries({ queryKey: ['jobs', 'detail', id] }),
  })
}

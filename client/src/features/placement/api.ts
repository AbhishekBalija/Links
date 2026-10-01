import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { Opportunity } from '../jobs/types'
import type { OpportunityInput } from './opportunityForm'

export type ManagedStatus = 'published' | 'draft' | 'closed'

type FeedMeta = { next_cursor?: string }

// useManaged lists every Opportunity for placement staff in one status,
// newest first, with applicant counts.
export function useManaged(status: ManagedStatus) {
  return useInfiniteQuery({
    queryKey: ['placement', 'list', status],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: '20', status })
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<Opportunity[], FeedMeta>(`/api/v1/opportunities/manage?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

// useManagedOne is one Opportunity as staff see it, drafts and counts included.
export function useManagedOne(id: string | undefined) {
  return useQuery({
    queryKey: ['placement', 'one', id],
    enabled: Boolean(id),
    queryFn: ({ signal }) => apiRequest<Opportunity>(`/api/v1/opportunities/${encodeURIComponent(id ?? '')}`, { signal }),
  })
}

// After any change, the office's lists, students' Jobs and Home are stale.
function useRefresh() {
  const client = useQueryClient()
  return (saved?: Opportunity) => {
    if (saved) client.setQueryData(['placement', 'one', saved.id], (old: Opportunity | undefined) => ({ ...old, ...saved }))
    client.invalidateQueries({ queryKey: ['placement'] })
    client.invalidateQueries({ queryKey: ['jobs'] })
    client.invalidateQueries({ queryKey: ['dashboard'] })
  }
}

export function useCreateOpportunity() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: (input: OpportunityInput) => apiRequest<Opportunity>('/api/v1/opportunities', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: refresh,
  })
}

export function useUpdateOpportunity() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<OpportunityInput> }) =>
      apiRequest<Opportunity>(`/api/v1/opportunities/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(input) }),
    onSuccess: refresh,
  })
}

// usePublishing publishes a draft or closes a published Opportunity early.
// A refusal (someone else got there first) also refreshes, so the page
// shows where it stands now.
export function usePublishing() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: 'publish' | 'close' }) =>
      apiRequest<Opportunity>(`/api/v1/opportunities/${encodeURIComponent(id)}/${action}`, { method: 'POST' }),
    onSuccess: refresh,
    onError: () => refresh(),
  })
}

import { keepPreviousData, useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { Category, FeedMeta } from '../notices/types'
import type { Authored, Department, Draft, MineFilter, Preview, QueueItem, RuleInput } from './types'

const base = '/api/v1/announcements'

// useMine loads the author's own Announcements, one status filter at a time,
// so each tab pages through its own list.
export function useMine(filter: MineFilter | null) {
  return useInfiniteQuery({
    queryKey: ['mine', filter ?? 'all'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: '20' })
      if (pageParam) params.set('cursor', pageParam)
      if (filter) params.set('status', filter)
      return apiPage<Authored[], FeedMeta>(`${base}/mine?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

// useSentBack loads everything waiting on the author's changes. It is short,
// so one page of 50 is plenty.
export function useSentBack() {
  return useQuery({
    queryKey: ['mine', 'attention'],
    queryFn: ({ signal }) => apiRequest<Authored[]>(`${base}/mine?status=attention&limit=50`, { signal }),
  })
}

export function useAuthored(id: string | undefined) {
  return useQuery({
    queryKey: ['mine', 'one', id],
    enabled: Boolean(id),
    queryFn: ({ signal }) => apiRequest<Authored>(`${base}/${encodeURIComponent(id ?? '')}`, { signal }),
  })
}

export function useDepartments() {
  return useQuery({
    queryKey: ['departments'],
    staleTime: 10 * 60_000,
    queryFn: async ({ signal }) => {
      const result = await apiRequest<{ departments: Department[] }>('/api/v1/departments', { signal })
      return result.departments
    },
  })
}

// usePreview asks the server what posting would do (publish now or go to an
// approver) and how many people the Audience reaches.
export function usePreview(category: Category, audience: RuleInput[]) {
  return useQuery({
    queryKey: ['preview', category, audience],
    staleTime: 60_000,
    placeholderData: keepPreviousData,
    queryFn: ({ signal }) =>
      apiRequest<Preview>(`${base}/preview`, {
        method: 'POST',
        body: JSON.stringify({ category, audience }),
        signal,
      }),
  })
}

// After any change, the author's lists, Home counts and feeds are stale.
function useInvalidateAll() {
  const client = useQueryClient()
  return () => {
    client.invalidateQueries({ queryKey: ['mine'] })
    client.invalidateQueries({ queryKey: ['dashboard'] })
    client.invalidateQueries({ queryKey: ['notices'] })
  }
}

export function useCreate() {
  const invalidate = useInvalidateAll()
  return useMutation({
    mutationFn: (input: Draft & { draft: boolean }) =>
      apiRequest<Authored>(base, { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: invalidate,
  })
}

export function useUpdate() {
  const invalidate = useInvalidateAll()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Draft }) =>
      apiRequest<Authored>(`${base}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
    onSuccess: invalidate,
  })
}

export function useSubmit() {
  const invalidate = useInvalidateAll()
  return useMutation({
    mutationFn: (id: string) => apiRequest<Authored>(`${base}/${id}/submit-for-approval`, { method: 'POST' }),
    onSuccess: invalidate,
  })
}

export function useWithdraw() {
  const invalidate = useInvalidateAll()
  return useMutation({
    mutationFn: (id: string) => apiRequest<Authored>(`${base}/${id}/withdraw`, { method: 'POST' }),
    onSuccess: invalidate,
  })
}

// useQueue loads what the approver may act on, oldest first.
export function useQueue() {
  return useInfiniteQuery({
    queryKey: ['queue'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: '50' })
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<QueueItem[], FeedMeta>(`${base}/approvals?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

export function useReview() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, decision, note }: { id: string; decision: 'approve' | 'reject'; note?: string }) =>
      apiRequest<Authored>(`${base}/${id}/approval`, { method: 'PATCH', body: JSON.stringify({ decision, note: note ?? '' }) }),
    // Whatever happened (including "someone else acted first"), the queue,
    // Home counts and feeds may have changed.
    onSettled: () => {
      client.invalidateQueries({ queryKey: ['queue'] })
      client.invalidateQueries({ queryKey: ['dashboard'] })
      client.invalidateQueries({ queryKey: ['notices'] })
    },
  })
}

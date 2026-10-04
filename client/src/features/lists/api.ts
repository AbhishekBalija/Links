import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import { listParams } from './logic'
import type { ListFilter, ListMeta, WaitingPerson } from './types'

const base = '/api/v1/admin/users'

// useNotSignedIn pages through who hasn't signed in yet, oldest first.
export function useNotSignedIn(filter: ListFilter) {
  return useInfiniteQuery({
    queryKey: ['not-signed-in', filter],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = listParams(filter)
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<WaitingPerson[], ListMeta>(`${base}/not-signed-in?${params}`, { signal })
    },
    getNextPageParam: (last) => last.meta?.next_cursor || undefined,
  })
}

// notSignedInEmails is every matching email, for a reminder from the
// college's own mail.
export async function notSignedInEmails(filter: ListFilter): Promise<string[]> {
  const result = await apiRequest<{ emails: string[] }>(`${base}/not-signed-in/emails?${listParams(filter)}`)
  return result.emails
}

function useRefreshLists() {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ['not-signed-in'] })
    queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    queryClient.invalidateQueries({ queryKey: ['access-requests'] })
  }
}

// useFixEmail corrects a row's email (#174).
export function useFixEmail() {
  const refresh = useRefreshLists()
  return useMutation({
    mutationFn: ({ id, email }: { id: string; email: string }) =>
      apiRequest(`${base}/${encodeURIComponent(id)}/email`, { method: 'PATCH', body: JSON.stringify({ email }) }),
    onSettled: refresh,
  })
}

// useRemoveRow removes a row nobody has signed into, freeing its USN.
export function useRemoveRow() {
  const refresh = useRefreshLists()
  return useMutation({
    mutationFn: (id: string) => apiRequest(`${base}/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    onSettled: refresh,
  })
}

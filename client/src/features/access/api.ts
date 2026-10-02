import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { AccessRequest } from './types'

const queueKey = ['access-requests']

// useAccessRequests loads the requests waiting for the viewer, oldest first:
// every Department for admins, their own for an HOD.
// retryOnMount: false is for a view that only shows the count, so mounting it
// again doesn't re-request a failed queue (the queue's own screen does).
export function useAccessRequests({ retryOnMount = true }: { retryOnMount?: boolean } = {}) {
  return useQuery({
    queryKey: queueKey,
    retryOnMount,
    queryFn: ({ signal }) => apiRequest<{ users: AccessRequest[]; total: number }>('/api/v1/admin/users/review-queue', { signal }),
    select: (data) => data.users,
  })
}

// useDecide approves or rejects a request. Afterwards the queue and Home
// reload from the server; nothing is guessed in advance.
export function useDecide() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, decision, note }: { id: string; decision: 'approve' | 'reject'; note: string }) =>
      decision === 'approve'
        ? apiRequest(`/api/v1/admin/users/${id}/verify`, { method: 'PATCH', body: JSON.stringify({}) })
        : apiRequest(`/api/v1/admin/users/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'rejected', note }) }),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queueKey })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

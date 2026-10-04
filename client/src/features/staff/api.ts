import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { staffPayload } from './logic'

export type StaffAdded = { user_id: string; status: string; emailed: boolean }

// useAddStaff adds a staff member; LINKS emails them how to sign in.
export function useAddStaff() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: ReturnType<typeof staffPayload>) =>
      apiRequest<StaffAdded>('/api/v1/admin/users', { method: 'POST', body: JSON.stringify(payload) }),
    // Home counts departments with no HOD and who hasn't signed in yet.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
  })
}

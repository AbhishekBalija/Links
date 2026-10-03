import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { GrantInput, Handover, RoleAssignment } from './types'

const rolesKey = (userId: string) => ['roles', userId]
const base = (userId: string) => `/api/v1/admin/users/${encodeURIComponent(userId)}/roles`

// useRoles lists a person's Role assignments for someone who manages roles.
// An HOD gets 404 for anyone but their own Department's students, and the
// panel then stays hidden, so a 404 isn't retried.
export function useRoles(userId: string, enabled: boolean) {
  return useQuery({
    queryKey: rolesKey(userId),
    enabled,
    retry: false,
    queryFn: ({ signal }) => apiRequest<RoleAssignment[]>(base(userId), { signal }),
  })
}

function organiserQuery(organiserId: string | null) {
  return organiserId ? `?organiser_id=${encodeURIComponent(organiserId)}` : ''
}

// useEndingPreview asks the server what ending the role would do, without
// changing anything (it runs the real code and rolls it back).
export function useEndingPreview(userId: string, assignmentId: string, enabled: boolean) {
  return useQuery({
    queryKey: ['roles', userId, 'ending', assignmentId],
    enabled,
    retry: false,
    staleTime: 0,
    gcTime: 0,
    queryFn: ({ signal }) => apiRequest<Handover>(`${base(userId)}/${encodeURIComponent(assignmentId)}/ending`, { signal }),
  })
}

// After a change the roles, the profile and the person's lists reload from
// the server; nothing is guessed in advance.
function useReloadAfterRoleChange(userId: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: rolesKey(userId) })
    queryClient.invalidateQueries({ queryKey: ['profile'] })
    queryClient.invalidateQueries({ queryKey: ['directory'] })
    queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }
}

export function useEndRole(userId: string) {
  const reload = useReloadAfterRoleChange(userId)
  return useMutation({
    mutationFn: ({ assignmentId, organiserId }: { assignmentId: string; organiserId: string | null }) =>
      apiRequest<RoleAssignment & { handover: Handover }>(`${base(userId)}/${encodeURIComponent(assignmentId)}${organiserQuery(organiserId)}`, {
        method: 'DELETE',
      }),
    onSuccess: reload,
  })
}

export function useGrantRole(userId: string) {
  const reload = useReloadAfterRoleChange(userId)
  return useMutation({
    mutationFn: (input: GrantInput) => apiRequest<RoleAssignment>(base(userId), { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: reload,
  })
}

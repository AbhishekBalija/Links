import { keepPreviousData, useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDownload, apiPage, apiRequest } from '../../shared/api/client'
import type { ApplicationStatus, Opportunity } from '../jobs/types'
import { applicantParams, exportName, type Applicant, type ApplicantFilters } from './applicants'
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

// useApplicants pages through an Opportunity's applicants, in the order
// they applied, with the filters applied on the server.
export function useApplicants(id: string, filters: ApplicantFilters) {
  return useInfiniteQuery({
    queryKey: ['placement', 'applicants', id, filters],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) =>
      apiPage<Applicant[], FeedMeta>(`/api/v1/opportunities/${encodeURIComponent(id)}/applications?${applicantParams(filters, pageParam)}`, { signal }),
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
    placeholderData: keepPreviousData,
  })
}

// useApplicantStatus moves an Application. It sends the status the officer
// saw, so the server refuses (409) rather than overwrite someone else's
// change. Either way, the list, the counts and Home are refreshed.
export function useApplicantStatus(opportunityId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ applicant, to }: { applicant: Applicant; to: ApplicationStatus }) =>
      apiRequest<Applicant>(`/api/v1/opportunity-applications/${encodeURIComponent(applicant.id)}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ from: applicant.status, status: to }),
      }),
    onSettled: () => {
      client.invalidateQueries({ queryKey: ['placement', 'applicants', opportunityId] })
      client.invalidateQueries({ queryKey: ['placement', 'one', opportunityId] })
      client.invalidateQueries({ queryKey: ['placement', 'list'] })
      client.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

// useExportApplicants downloads the applicants as CSV, all or one status.
// Each download is audited on the server.
export function useExportApplicants(item: Pick<Opportunity, 'id' | 'title' | 'company'>) {
  return useMutation({
    mutationFn: async (status: ApplicationStatus | null) => {
      const query = status ? `?status=${status}` : ''
      const name = exportName(item.title, item.company, status)
      const file = await apiDownload(`/api/v1/opportunities/${encodeURIComponent(item.id)}/export${query}`, name)
      const url = URL.createObjectURL(file.blob)
      const link = document.createElement('a')
      link.href = url
      link.download = name
      link.click()
      URL.revokeObjectURL(url)
    },
  })
}

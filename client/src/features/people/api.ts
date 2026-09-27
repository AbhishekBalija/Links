import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { DirectoryMeta, Entry, Filters, Overview, PublicProfile } from './types'

const PAGE_SIZE = 30

// A search shorter than this is ignored by the server, so the plain list shows.
export const MIN_SEARCH = 2

function directoryParams(filters: Filters) {
  const params = new URLSearchParams()
  if (filters.department) params.set('department', filters.department)
  if (filters.role) params.set('role', filters.role)
  if (filters.batch) params.set('batch', filters.batch)
  if (filters.q && filters.q.trim().length >= MIN_SEARCH) params.set('q', filters.q.trim())
  return params
}

// useDirectory loads the directory a page at a time, alphabetical, or the
// best matches first when there is a search. While new filters load, the old
// list stays on screen instead of flashing a skeleton.
export function useDirectory(filters: Filters, enabled = true) {
  return useInfiniteQuery({
    queryKey: ['directory', filters],
    enabled,
    initialPageParam: '',
    placeholderData: keepPreviousData,
    queryFn: ({ pageParam, signal }) => {
      const params = directoryParams(filters)
      params.set('limit', String(PAGE_SIZE))
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<Entry[], DirectoryMeta>(`/api/v1/directory?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

// useDirectoryCount asks how many people some filters would show, for the
// filter sheet's "Show 104 people" before the filters are applied.
export function useDirectoryCount(filters: Filters, enabled: boolean) {
  return useQuery({
    queryKey: ['directory', 'count', filters],
    enabled,
    placeholderData: keepPreviousData,
    queryFn: async ({ signal }) => {
      const params = directoryParams(filters)
      params.set('limit', '1')
      const page = await apiPage<Entry[], DirectoryMeta>(`/api/v1/directory?${params}`, { signal })
      return page.meta?.total ?? 0
    },
  })
}

export function useProfile(username: string | null | undefined) {
  return useQuery({
    queryKey: ['profile', username],
    enabled: Boolean(username),
    queryFn: ({ signal }) => apiRequest<PublicProfile>(`/api/v1/profiles/${encodeURIComponent(username ?? '')}`, { signal }),
  })
}

export function useDepartmentOverview(code: string | null | undefined) {
  return useQuery({
    queryKey: ['department-overview', code],
    enabled: Boolean(code),
    queryFn: ({ signal }) => apiRequest<Overview>(`/api/v1/departments/${encodeURIComponent(code ?? '')}/overview`, { signal }),
  })
}

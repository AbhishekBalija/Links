import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { Category, FeedMeta, Notice } from './types'

const PAGE_SIZE = 20

// useNoticeFeed loads the reader's feed one page at a time. TanStack Query's
// infinite query keeps every loaded page and asks for the next one with the
// cursor the server returned, so scrolling only ever fetches new notices.
export function useNoticeFeed(category: Category | null) {
  return useInfiniteQuery({
    queryKey: ['notices', 'feed', category ?? 'all'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE) })
      if (pageParam) params.set('cursor', pageParam)
      if (category) params.set('category', category)
      return apiPage<Notice[], FeedMeta>(`/api/v1/announcements?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

export function useNotice(id: string) {
  return useQuery({
    queryKey: ['notices', 'detail', id],
    queryFn: ({ signal }) => apiRequest<Notice>(`/api/v1/announcements/${encodeURIComponent(id)}`, { signal }),
  })
}

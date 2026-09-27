import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiPage, apiRequest } from '../../shared/api/client'
import type { Answer, AnswerSummary, CampusEvent, EventType, FeedMeta, Show } from './types'

const PAGE_SIZE = 20

// useEventFeed loads the reader's Events a page at a time: upcoming soonest
// first, the ones they're going to, or past ones most recent first.
export function useEventFeed(show: Show, type: EventType | null, limit = PAGE_SIZE) {
  return useInfiniteQuery({
    queryKey: ['events', 'feed', show, type ?? 'all', limit],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: String(limit) })
      if (show !== 'upcoming') params.set('show', show)
      if (type) params.set('event_type', type)
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<CampusEvent[], FeedMeta>(`/api/v1/events?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

export function useEvent(id: string) {
  return useQuery({
    queryKey: ['events', 'detail', id],
    queryFn: ({ signal }) => apiRequest<CampusEvent>(`/api/v1/events/${encodeURIComponent(id)}`, { signal }),
  })
}

// useAnswers is the answer counts and the reader's own answer for one Event.
export function useAnswers(id: string, enabled = true) {
  return useQuery({
    queryKey: ['events', 'answers', id],
    enabled,
    queryFn: ({ signal }) => apiRequest<AnswerSummary>(`/api/v1/events/${encodeURIComponent(id)}/rsvps`, { signal }),
  })
}

// useAnswer records the reader's answer. The server counts seats under a
// lock, so "the event is full" comes back as an error to show.
export function useAnswer(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (status: Answer) =>
      apiRequest<AnswerSummary>(`/api/v1/events/${encodeURIComponent(id)}/rsvp`, { method: 'POST', body: JSON.stringify({ status }) }),
    onSuccess: (summary) => {
      queryClient.setQueryData(['events', 'answers', id], summary)
      // Lists show answers too, so they refresh in the background.
      queryClient.invalidateQueries({ queryKey: ['events', 'feed'] })
    },
    onError: () => queryClient.invalidateQueries({ queryKey: ['events', 'answers', id] }),
  })
}

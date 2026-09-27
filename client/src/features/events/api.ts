import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDownload, apiPage, apiRequest } from '../../shared/api/client'
import type { Answer, AnswerPerson, AnswerSummary, CampusEvent, EventInput, EventQueueItem, EventType, FeedMeta, MineEventFilter, Show } from './types'

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

export function useEvent(id: string, enabled = true) {
  return useQuery({
    queryKey: ['events', 'detail', id],
    enabled,
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

// useMyEvents loads the proposer's own Events for one status group, newest
// first, a page at a time.
export function useMyEvents(filter: MineEventFilter) {
  return useInfiniteQuery({
    queryKey: ['events', 'mine', filter],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), status: filter })
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<CampusEvent[], FeedMeta>(`/api/v1/events/mine?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

// After a proposal changes, the author's lists, the feed and Home are stale.
function useInvalidateEvents() {
  const client = useQueryClient()
  return () => {
    client.invalidateQueries({ queryKey: ['events'] })
    client.invalidateQueries({ queryKey: ['mine', 'any'] })
    client.invalidateQueries({ queryKey: ['dashboard'] })
  }
}

export function useCreateEvent() {
  const invalidate = useInvalidateEvents()
  return useMutation({
    mutationFn: (input: EventInput & { draft: boolean }) => apiRequest<CampusEvent>('/api/v1/events', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: invalidate,
  })
}

export function useUpdateEvent() {
  const invalidate = useInvalidateEvents()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<EventInput> }) =>
      apiRequest<CampusEvent>(`/api/v1/events/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(input) }),
    onSuccess: invalidate,
  })
}

export function useSubmitEvent() {
  const invalidate = useInvalidateEvents()
  return useMutation({
    mutationFn: (id: string) => apiRequest<CampusEvent>(`/api/v1/events/${encodeURIComponent(id)}/submit-for-approval`, { method: 'POST' }),
    onSuccess: invalidate,
  })
}

export function useDeleteDraft() {
  const invalidate = useInvalidateEvents()
  return useMutation({
    mutationFn: (id: string) => apiRequest<void>(`/api/v1/events/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    onSuccess: invalidate,
  })
}

// useCancelEvent cancels a proposal or a published Event. Everyone invited
// sees the reason.
export function useCancelEvent() {
  const invalidate = useInvalidateEvents()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      apiRequest<CampusEvent>(`/api/v1/events/${encodeURIComponent(id)}/cancel`, { method: 'POST', body: JSON.stringify({ reason }) }),
    onSuccess: invalidate,
  })
}

// useAnswerList pages through everyone who answered, earliest first, for the
// event's organisers.
export function useAnswerList(id: string) {
  return useInfiniteQuery({
    queryKey: ['events', 'answers', id, 'people'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: '50' })
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<AnswerSummary, FeedMeta>(`/api/v1/events/${encodeURIComponent(id)}/rsvps?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
    select: (data) => ({ counts: data.pages[0]?.data.counts, people: data.pages.flatMap((p): AnswerPerson[] => p.data.people ?? []) }),
  })
}

// useExportAnswers downloads the answers as a CSV. Each export is audited on
// the server.
export function useExportAnswers(id: string, title: string) {
  return useMutation({
    mutationFn: async () => {
      const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 60) || 'event'
      const file = await apiDownload(`/api/v1/events/${encodeURIComponent(id)}/export`, `${slug}-answers.csv`)
      const url = URL.createObjectURL(file.blob)
      const link = document.createElement('a')
      link.href = url
      link.download = file.name
      link.click()
      URL.revokeObjectURL(url)
    },
  })
}

// useEventReviews loads the event proposals waiting for the caller, oldest
// submission first.
export function useEventReviews() {
  return useInfiniteQuery({
    queryKey: ['events', 'reviews'],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: '50' })
      if (pageParam) params.set('cursor', pageParam)
      return apiPage<EventQueueItem[], FeedMeta>(`/api/v1/events/reviews?${params}`, { signal })
    },
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })
}

export type EventDecision = 'approve' | 'request_changes' | 'reject'

// useEventDecision records a reviewer's decision at the event's stage.
export function useEventDecision() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ item, decision, note }: { item: EventQueueItem; decision: EventDecision; note: string }) =>
      apiRequest<CampusEvent>(`/api/v1/events/${encodeURIComponent(item.id)}/${item.stage === 'hod' ? 'hod-review' : 'final-approval'}`, {
        method: 'PATCH',
        body: JSON.stringify({ decision, note }),
      }),
    // Whatever happened (including "someone else decided first"), the queue,
    // Home counts and the feed may have changed.
    onSettled: () => {
      client.invalidateQueries({ queryKey: ['events'] })
      client.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

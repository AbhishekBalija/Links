import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'

// useHasPosted is true once the author has any announcement or event, in any
// status, so a first visit to My posts can say what the page is for.
export function useHasPosted() {
  return useQuery({
    // Under 'mine', so any change to an announcement refreshes it; event
    // changes refresh it by name.
    queryKey: ['mine', 'any'],
    queryFn: async ({ signal }) => {
      const [announcements, events] = await Promise.all([
        apiRequest<unknown[]>('/api/v1/announcements/mine?limit=1', { signal }),
        apiRequest<unknown[]>('/api/v1/events/mine?limit=1', { signal }),
      ])
      return announcements.length > 0 || events.length > 0
    },
  })
}

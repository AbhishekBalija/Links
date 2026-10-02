import { useQueue } from '../../announcements/api'
import { useEventReviews } from '../../events/api'
import { useAccessRequests } from '../api'
import { WorkspaceTabs } from './WorkspaceTabs'

// QueueTabs splits an HOD's Approval queue into posts (announcements and
// event proposals) and Access requests, each with how many wait.
export function QueueTabs({ active }: { active: 'Posts' | 'Access requests' }) {
  // The tabs remount when the page switches between loading and an error, so
  // they mustn't re-request a failed list, or the page would loop.
  const queue = useQueue({ retryOnMount: false })
  const events = useEventReviews({ retryOnMount: false })
  const requests = useAccessRequests({ retryOnMount: false })
  const posts = (queue.data?.pages.flatMap((page) => page.data).length ?? 0) + (events.data?.pages.flatMap((page) => page.data).length ?? 0)
  const more = queue.hasNextPage || events.hasNextPage
  return (
    <WorkspaceTabs
      label="What's waiting"
      active={active}
      tabs={[
        { label: 'Posts', to: '/approvals', count: queue.data && events.data ? `${posts}${more ? '+' : ''}` : undefined },
        { label: 'Access requests', to: '/approvals/access', count: requests.data?.length },
      ]}
    />
  )
}

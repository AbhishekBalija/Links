import type { WorkspaceTab } from './components/WorkspaceTabs'

// The Admin workspace's tabs. Add staff and Not signed in join them with
// their own tickets.
export const adminTabs: WorkspaceTab[] = [
  { label: 'Access requests', to: '/admin/requests' },
  { label: 'Import students', to: '/admin/import' },
]

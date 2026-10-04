import type { WorkspaceTab } from './components/WorkspaceTabs'

// The Admin workspace's tabs. Not signed in joins them with its own ticket.
export const adminTabs: WorkspaceTab[] = [
  { label: 'Access requests', to: '/admin/requests' },
  { label: 'Import students', to: '/admin/import' },
  { label: 'Add staff', to: '/admin/staff' },
]

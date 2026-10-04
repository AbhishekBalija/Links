import type { WorkspaceTab } from './components/WorkspaceTabs'

// The Admin workspace's tabs.
export const adminTabs: WorkspaceTab[] = [
  { label: 'Access requests', to: '/admin/requests' },
  { label: 'Import students', to: '/admin/import' },
  { label: 'Add staff', to: '/admin/staff' },
  { label: 'Not signed in', to: '/admin/not-signed-in' },
]

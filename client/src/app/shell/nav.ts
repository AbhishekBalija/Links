import { BriefcaseBusiness, CalendarDays, ClipboardList, House, ListChecks, Newspaper, PenLine, UserRound, Users, type LucideIcon } from 'lucide-react'

export type NavItem = {
  to: string
  label: string
  // A shorter label for the phone's bottom bar.
  short?: string
  icon: LucideIcon
  // Roles that see this item; leave it out for everyone.
  roles?: string[]
  // Other paths that belong to this item, such as a Department page under People.
  also?: string[]
}

// Roles that can post Announcements (auth policy: post_announcement), which
// are also the roles that propose events (propose_event).
export const posterRoles = ['student_coordinator', 'faculty', 'hod', 'placement_officer', 'principal', 'admin']

// Placement staff: the placement officer, the principal and admins work as
// one office on Opportunities (auth policy: post_opportunity).
export const placementRoles = ['placement_officer', 'principal', 'admin']

// Roles that approve Announcements (auth policy: approve_announcement).
export const approverRoles = ['hod', 'principal', 'admin']

const items: NavItem[] = [
  { to: '/', label: 'Home', icon: House },
  { to: '/notices', label: 'Notices', icon: Newspaper },
  { to: '/events', label: 'Events', icon: CalendarDays },
  { to: '/jobs', label: 'Jobs', icon: BriefcaseBusiness, roles: ['student'] },
  { to: '/placement', label: 'Placement', icon: ClipboardList, roles: placementRoles },
  { to: '/people', label: 'People', icon: Users, also: ['/departments'] },
  { to: '/approvals', label: 'Approval queue', short: 'Approvals', icon: ListChecks, roles: approverRoles },
  { to: '/mine', label: 'My posts', short: 'Mine', icon: PenLine, roles: posterRoles },
  { to: '/profile', label: 'Profile', icon: UserRound },
]

export function canApprove(roles: string[]): boolean {
  return roles.some((role) => approverRoles.includes(role))
}

export function isPlacementStaff(roles: string[]): boolean {
  return roles.some((role) => placementRoles.includes(role))
}

export function canPost(roles: string[]): boolean {
  return roles.some((role) => posterRoles.includes(role))
}

export function navFor(roles: string[]): NavItem[] {
  return items.filter((item) => !item.roles || item.roles.some((role) => roles.includes(role)))
}

// The phone's bottom bar fits five tabs. When a role has more, Profile steps
// out first (it stays one tap away from the avatar on Home), then People
// (reachable from Home's department link), so HODs keep their work tabs.
// The principal and admins have more work tabs than fit, so Placement, which
// is desktop work for them, steps out next. Students have no work tabs; for
// them Jobs takes People's place and they keep Profile, which they edit more
// than staff do.
const maxTabs = 5

export function tabsFor(roles: string[]): NavItem[] {
  let tabs = navFor(roles)
  const staff = canPost(roles) || canApprove(roles)
  const leaveFirst = staff ? ['/profile', '/people', '/placement'] : ['/people', '/profile']
  for (const path of leaveFirst) {
    if (tabs.length <= maxTabs) break
    tabs = tabs.filter((item) => item.to !== path)
  }
  return tabs
}

const roleLabels: Record<string, string> = {
  student: 'Student',
  student_coordinator: 'Student coordinator',
  faculty: 'Faculty',
  hod: 'HOD',
  placement_officer: 'Placement officer',
  principal: 'Principal',
  alumni: 'Alumni',
  club_organizer: 'Club organiser',
  admin: 'Admin',
}

// The sidebar shows the most senior role the user holds.
const seniority = ['admin', 'principal', 'hod', 'placement_officer', 'faculty', 'student_coordinator', 'club_organizer', 'student', 'alumni']

export function roleLabel(role: string): string {
  return roleLabels[role] ?? role
}

export function mainRoleLabel(roles: string[]): string {
  const top = seniority.find((role) => roles.includes(role)) ?? roles[0]
  return top ? (roleLabels[top] ?? top) : ''
}

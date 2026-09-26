import { House, ListChecks, Newspaper, PenLine, UserRound, type LucideIcon } from 'lucide-react'

export type NavItem = {
  to: string
  label: string
  // A shorter label for the phone's bottom bar.
  short?: string
  icon: LucideIcon
  // Roles that see this item; leave it out for everyone.
  roles?: string[]
}

// Roles that can post Announcements (auth policy: post_announcement).
export const posterRoles = ['student_coordinator', 'faculty', 'hod', 'placement_officer', 'principal', 'admin']

// Roles that approve Announcements (auth policy: approve_announcement).
export const approverRoles = ['hod', 'principal', 'admin']

const items: NavItem[] = [
  { to: '/', label: 'Home', icon: House },
  { to: '/notices', label: 'Notices', icon: Newspaper },
  { to: '/approvals', label: 'Approval queue', short: 'Approvals', icon: ListChecks, roles: approverRoles },
  { to: '/mine', label: 'My announcements', short: 'Mine', icon: PenLine, roles: posterRoles },
  { to: '/profile/edit', label: 'Profile', icon: UserRound },
]

export function canApprove(roles: string[]): boolean {
  return roles.some((role) => approverRoles.includes(role))
}

export function canPost(roles: string[]): boolean {
  return roles.some((role) => posterRoles.includes(role))
}

export function navFor(roles: string[]): NavItem[] {
  return items.filter((item) => !item.roles || item.roles.some((role) => roles.includes(role)))
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

export function mainRoleLabel(roles: string[]): string {
  const top = seniority.find((role) => roles.includes(role)) ?? roles[0]
  return top ? (roleLabels[top] ?? top) : ''
}

import { House, Newspaper, UserRound, type LucideIcon } from 'lucide-react'

export type NavItem = {
  to: string
  label: string
  icon: LucideIcon
  // Roles that see this item; leave it out for everyone.
  roles?: string[]
}

// Every signed-in user reads notices. The composer (#41) and the approval
// queue (#42) add their items here with the roles that can use them.
const items: NavItem[] = [
  { to: '/', label: 'Home', icon: House },
  { to: '/notices', label: 'Notices', icon: Newspaper },
  { to: '/profile/edit', label: 'Profile', icon: UserRound },
]

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

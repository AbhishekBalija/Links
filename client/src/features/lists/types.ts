// WaitingPerson is someone a class list or a staff invite let in who hasn't
// signed in yet (GET /admin/users/not-signed-in).
export type WaitingPerson = {
  user_id: string
  full_name: string
  email: string
  kind: 'student' | 'staff'
  role: string
  usn: string
  batch_year: number
  department_code: string
  added_at: string
  added_by: { full_name: string }
}

export type ListFilter = { department: string; kind: '' | 'student' | 'staff' }

export type ListMeta = { total: number; next_cursor?: string }

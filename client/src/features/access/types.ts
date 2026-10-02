// AccessRequest is one waiting request from GET /api/v1/admin/users/review-queue.
export type AccessRequest = {
  id: string
  email: string | null
  profile?: { full_name: string; username: string }
  student_identity?: {
    usn: string
    department_code?: string
    department_name?: string
    department_has_hod: boolean
    batch_year: number
  }
  created_at: string
  // Set when the row came from a class list and the person said "Not you?".
  reported_at: string | null
}

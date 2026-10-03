// The import's answer (POST /api/v1/admin/users/import), for a check
// (dry_run) or a real import.
export type ImportRow = {
  row: number
  email: string
  full_name: string
  usn: string
  department_code?: string
  batch_year?: number
  status: 'ready' | 'created' | 'failed'
  outside?: boolean
  user_id?: string
  error?: string
}

export type ImportGroup = {
  department_code: string
  department_name: string
  batch_year: number
  rows: number
  ready: number
  created: number
  failed: number
  outside: boolean
}

export type ImportResult = {
  dry_run: boolean
  department: { code: string; name: string } | null
  ready: number
  created: number
  failed: number
  groups: ImportGroup[]
  rows: ImportRow[]
}

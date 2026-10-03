import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '../../shared/api/client'
import type { ImportResult } from './types'

// useImport sends a class list: a check first (dryRun, nothing saved), then
// the real import. The Department comes from each USN; an HOD's import is
// held to their own Department by the server.
export function useImport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ file, dryRun }: { file: File; dryRun: boolean }) => {
      const form = new FormData()
      form.append('file', file)
      if (dryRun) form.append('dry_run', 'true')
      return apiRequest<ImportResult>('/api/v1/admin/users/import', { method: 'POST', body: form })
    },
    onSuccess: (result) => {
      // Home counts who hasn't signed in yet and lists recent imports.
      if (!result.dry_run) queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

// saveText hands the browser a text file to save, such as the rows to fix.
export function saveText(name: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.click()
  URL.revokeObjectURL(url)
}

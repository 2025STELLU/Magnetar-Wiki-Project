import { wFetch } from '../util/fetch-util'

export type AdminLogAction = '' | 'create' | 'change' | 'delete'

export interface AdminLogEntry {
  id: number
  actor: string
  action: string
  target: string
  detail: string
  created_at: string
}

export async function listAdminLog(actor: string, action: AdminLogAction, offset = 0) {
  const params = new URLSearchParams()
  if (actor) params.set('actor', actor)
  if (action) params.set('action', action)
  params.set('offset', String(offset))
  return await wFetch<{ entries: AdminLogEntry[]; total: number }>(
    `/pw-api/console/admin-log/list?${params.toString()}`
  )
}

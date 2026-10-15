import { wFetch } from '../util/fetch-util'

export interface ConsoleRole {
  id: number
  slug: string
  name: string
  description: string
  is_level_role: boolean
  level: number
  permissions: string[]
}

export interface PermissionInfo {
  key: string
  description: string
  group: string
}

export interface RoleSaveBody {
  id?: number
  slug: string
  name: string
  description: string
  permissions: string[]
}

export async function listRoles() {
  return await wFetch<{ roles: ConsoleRole[] }>('/pw-api/console/roles/list')
}

export async function listPermissions() {
  return await wFetch<{ permissions: PermissionInfo[] }>('/pw-api/console/roles/permissions')
}

export async function saveRole(body: RoleSaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/roles/save', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deleteRole(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/roles/delete', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

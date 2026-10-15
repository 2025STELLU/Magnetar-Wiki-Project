import { wFetch } from '../util/fetch-util'

export interface ConsoleInvite {
  id: number
  code: string
  url: string
  email: string
  kind: string
  created_at: string
  activated_at: string | null
  activated_by: string
}

export interface SuspiciousUser {
  id: number
  username: string
  ip: string | null
}

export async function listInvites() {
  return await wFetch<{ invites: ConsoleInvite[] }>('/pw-api/console/invites/list')
}

export async function createInvite(email: string) {
  return await wFetch<{ ok: boolean; url: string }>('/pw-api/console/invites/create', {
    method: 'POST',
    sendJson: true,
    body: { email },
  })
}

export async function revokeInvite(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/invites/revoke', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

export async function listSuspiciousUsers() {
  return await wFetch<{ users: SuspiciousUser[] }>('/pw-api/console/invites/suspicious')
}

export async function clearSuspiciousAddresses(userId?: number) {
  return await wFetch<{ ok: boolean; cleared: number }>(
    '/pw-api/console/invites/suspicious',
    {
      method: 'POST',
      sendJson: true,
      body: userId ? { user_id: userId } : {},
    }
  )
}

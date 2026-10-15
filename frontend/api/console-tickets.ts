import { wFetch } from '../util/fetch-util'

export type TicketKind = 'ticket' | 'application'
export type TicketStatus = 'pending' | 'approved' | 'rejected' | 'closed' | 'all'
export type TicketAction = 'approve' | 'reject' | 'close'

export interface ConsoleTicket {
  id: number
  kind: string
  author: string
  subject: string
  status: string
  created_at: string
}

export interface TicketMessage {
  sender_name: string
  body: string
  created_at: string
}

export interface ConsoleTicketDetail extends ConsoleTicket {
  body: string
  source_page: string
  admin_notes: string
  reply: string
  messages: TicketMessage[]
  reviewed_at: string | null
  reviewed_by: string
  granted_role_id: number | null
}

export async function listTickets(kind: TicketKind, status: TicketStatus, offset = 0) {
  return await wFetch<{ items: ConsoleTicket[]; total: number }>(
    `/pw-api/console/tickets/list?type=${encodeURIComponent(kind)}&status=${encodeURIComponent(status)}&offset=${offset}`
  )
}

export async function getTicketDetail(id: number) {
  return await wFetch<{ ticket: ConsoleTicketDetail }>(
    `/pw-api/console/tickets/detail?id=${encodeURIComponent(String(id))}`
  )
}

export async function replyTicket(id: number, message: string) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tickets/reply', {
    method: 'POST',
    sendJson: true,
    body: { id, message },
  })
}

export async function handleTicket(
  id: number,
  action: TicketAction,
  note: string,
  granted_role_id?: number
) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tickets/handle', {
    method: 'POST',
    sendJson: true,
    body: { id, action, note, granted_role_id },
  })
}

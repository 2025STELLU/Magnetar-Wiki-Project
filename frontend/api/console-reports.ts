import { wFetch } from '../util/fetch-util'

export type ReportTab = 'pending' | 'handled' | 'dismissed' | 'all'

export interface ConsoleReport {
  id: number
  reporter: string
  reported: string
  reason: string
  status: string
  created_at: string
}

export interface ReportMessage {
  sender_name: string
  body: string
  created_at: string
}

export interface ConsoleReportDetail extends ConsoleReport {
  messages: ReportMessage[] | string
  admin_notes: string
  reply: string
  reviewed_at: string | null
  reviewed_by: string
}

export async function listReports(status: ReportTab, offset = 0) {
  return await wFetch<{ reports: ConsoleReport[]; total: number }>(
    `/pw-api/console/reports/list?status=${encodeURIComponent(status)}&offset=${offset}`
  )
}

export async function getReportDetail(id: number) {
  return await wFetch<{ report: ConsoleReportDetail }>(
    `/pw-api/console/reports/detail?id=${encodeURIComponent(String(id))}`
  )
}

export async function handleReport(id: number, action: 'dismiss' | 'action_taken', note: string) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/reports/handle', {
    method: 'POST',
    sendJson: true,
    body: { id, action, note },
  })
}

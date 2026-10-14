import { wFetch } from '../util/fetch-util'

export interface CheckinStatus {
  checked_in: boolean
  streak: number
  total_days: number
}

export interface CheckinResult {
  ok: boolean
  streak: number
  total_days: number
  xp_awarded: number
}

export async function getCheckinStatus() {
  return await wFetch<CheckinStatus>('/pw-api/checkin/status')
}

export async function doCheckin() {
  return await wFetch<CheckinResult>('/pw-api/checkin/', { method: 'POST', sendJson: true, body: {} })
}

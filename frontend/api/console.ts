import { wFetch } from '../util/fetch-util'

export interface ConsoleOverview {
  users: number
  articles: number
  pending_reports: number
  pending_tickets: number
}

export interface LevelConfig {
  thresholds: number[]
  prefix: string
}

export async function getConsoleOverview() {
  return await wFetch<ConsoleOverview>('/pw-api/console/overview')
}

export async function getLevelConfig() {
  return await wFetch<LevelConfig>('/pw-api/console/levels')
}

export async function saveLevelConfig(cfg: LevelConfig) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/levels', { method: 'POST', sendJson: true, body: cfg })
}

export interface ConsoleUserEntry {
  id: number
  username: string
  display_name: string
  level: number
}

export interface NameStyleConfig {
  font_family: string
  color: string
  gradient_from: string
  gradient_to: string
  gradient_direction: string
}

export const EMPTY_NAME_STYLE: NameStyleConfig = {
  font_family: '',
  color: '',
  gradient_from: '',
  gradient_to: '',
  gradient_direction: '',
}

export async function searchConsoleUsers(q: string) {
  return await wFetch<{ users: ConsoleUserEntry[] }>(`/pw-api/console/users/search?q=${encodeURIComponent(q)}`)
}

export async function getUserNameStyle(userId: number) {
  return await wFetch<{ style: NameStyleConfig | null }>(
    `/pw-api/console/users/style?user_id=${encodeURIComponent(String(userId))}`
  )
}

export async function saveUserNameStyle(userId: number, style: NameStyleConfig) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/users/style', {
    method: 'POST',
    sendJson: true,
    body: { user_id: userId, ...style },
  })
}

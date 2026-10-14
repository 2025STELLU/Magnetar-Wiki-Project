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

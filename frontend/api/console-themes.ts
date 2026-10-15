import { wFetch } from '../util/fetch-util'

export interface ThemeInfo {
  id: number
  name: string
  slug: string
  active: boolean
}

export interface UpdateInfo {
  current_version: string
  latest_version: string
  has_update: boolean
}

export interface ThemesResponse {
  themes: ThemeInfo[]
  update: UpdateInfo
}

export interface CheckUpdateResult {
  ok: boolean
  has_update: boolean
  latest_version: string
}

export async function getThemes() {
  return await wFetch<ThemesResponse>('/pw-api/console/themes/list')
}

export async function activateTheme(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/themes/activate', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

export async function checkUpdate() {
  return await wFetch<CheckUpdateResult>('/pw-api/console/themes/check_update', {
    method: 'POST',
    sendJson: true,
    body: {},
  })
}

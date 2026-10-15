import { wFetch } from '../util/fetch-util'

export interface SiteSettingsData {
  slug: string
  title: string
  headline: string
  domain: string
  media_domain: string
  home_page: string
  footer_license: string
  signup_notice: string
  password_help: string
  email_policy: string
  language: string
  time_zone: string
  theme_id: number | null
  system_theme_id: number | null
  rating_mode: string
  create_tags: string
}

export interface ThemeOption {
  id: number
  name: string
  slug: string
}

export interface LanguageOption {
  tag: string
  name: string
}

export interface SiteSettingsResponse {
  site: SiteSettingsData
  themes: ThemeOption[]
  languages: LanguageOption[]
  rating_modes: string[]
  tag_modes: string[]
  email_policies: string[]
}

export interface SaveSiteSettingsResult {
  ok: boolean
  warning?: string
}

export async function getSiteSettings() {
  return await wFetch<SiteSettingsResponse>('/pw-api/console/site/settings')
}

export async function saveSiteSettings(s: SiteSettingsData) {
  return await wFetch<SaveSiteSettingsResult>('/pw-api/console/site/settings', {
    method: 'POST',
    sendJson: true,
    body: s,
  })
}

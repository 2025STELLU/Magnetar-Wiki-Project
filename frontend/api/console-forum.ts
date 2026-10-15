import { wFetch } from '../util/fetch-util'

export interface ConsoleForumCategory {
  id: number
  name: string
  description: string
}

export interface ConsoleForumSection {
  id: number
  name: string
  description: string
  categories: ConsoleForumCategory[]
}

export interface ForumSectionSaveBody {
  id?: number
  name: string
  description: string
}

export interface ForumCategorySaveBody {
  id?: number
  section_id: number
  name: string
  description: string
}

export async function listForumSections() {
  return await wFetch<{ sections: ConsoleForumSection[] }>('/pw-api/console/forum/list')
}

export async function saveForumSection(body: ForumSectionSaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum/save_section', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deleteForumSection(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum/delete_section', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

export async function saveForumCategory(body: ForumCategorySaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum/save_category', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deleteForumCategory(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum/delete_category', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

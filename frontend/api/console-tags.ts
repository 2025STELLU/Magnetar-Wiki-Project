import { wFetch } from '../util/fetch-util'

export interface ConsoleTag {
  id: number
  name: string
  category_id: number | null
  category_name: string
  articles: number
  is_indexed: boolean
}

export interface ConsoleTagCategory {
  id: number
  name: string
  slug: string
  description: string
  priority: number | null
  tags: number
}

export interface TagSaveBody {
  id?: number
  name: string
  category_id?: number | null
  is_indexed?: boolean
}

export interface TagCategorySaveBody {
  id?: number
  name: string
  slug: string
  description: string
  priority?: number | null
}

export async function listTags() {
  return await wFetch<{ tags: ConsoleTag[]; categories: ConsoleTagCategory[] }>(
    '/pw-api/console/tags/list'
  )
}

export async function saveTag(body: TagSaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tags/save', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deleteTag(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tags/delete', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

export async function saveTagCategory(body: TagCategorySaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tags/save_category', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deleteTagCategory(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/tags/delete_category', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

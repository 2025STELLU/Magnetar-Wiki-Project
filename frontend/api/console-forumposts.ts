import { wFetch } from '../util/fetch-util'

export interface ConsoleForumPost {
  id: number
  title: string
  author: string
  category: string
  locked: boolean
  created_at: string
}

export interface ConsoleForumPostDetail extends ConsoleForumPost {
  summary: string
  post_count: number
}

export interface ConsoleForumCategory {
  id: number
  name: string
}

export async function listForumPosts(q = '', categoryId = 0, offset = 0) {
  const params = new URLSearchParams({ q, offset: String(offset) })
  if (categoryId > 0) params.set('category_id', String(categoryId))
  return await wFetch<{ posts: ConsoleForumPost[]; total: number }>(
    `/pw-api/console/forum-posts/list?${params.toString()}`
  )
}

export async function listForumCategories() {
  return await wFetch<{ categories: ConsoleForumCategory[] }>(
    '/pw-api/console/forum-posts/categories'
  )
}

export async function getForumPostDetail(id: number) {
  return await wFetch<{ post: ConsoleForumPostDetail }>(
    `/pw-api/console/forum-posts/detail?id=${id}`
  )
}

export async function deleteForumPost(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum-posts/delete', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

export async function lockForumPost(id: number, locked: boolean) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/forum-posts/lock', {
    method: 'POST',
    sendJson: true,
    body: { id, locked },
  })
}

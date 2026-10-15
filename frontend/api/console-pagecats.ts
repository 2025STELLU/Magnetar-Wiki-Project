import { wFetch } from '../util/fetch-util'

export interface PageCategory {
  id: number
  name: string
  description: string
}

export interface PageCatSaveBody {
  id?: number
  name: string
  description: string
}

export async function listPageCats() {
  return await wFetch<{ categories: PageCategory[] }>('/pw-api/console/page-categories/list')
}

export async function savePageCat(body: PageCatSaveBody) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/page-categories/save', {
    method: 'POST',
    sendJson: true,
    body,
  })
}

export async function deletePageCat(id: number) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/page-categories/delete', {
    method: 'POST',
    sendJson: true,
    body: { id },
  })
}

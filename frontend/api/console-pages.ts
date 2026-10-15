import { wFetch } from '../util/fetch-util'

export interface ConsolePage {
  id: number
  name: string
  title: string
  revisions: number
  updated_at: string
}

export interface ConsolePageDetail extends ConsolePage {
  summary: string
  locked: boolean
}

export async function listPages(q = '', offset = 0) {
  return await wFetch<{ pages: ConsolePage[]; total: number }>(
    `/pw-api/console/pages/list?q=${encodeURIComponent(q)}&offset=${offset}`
  )
}

export async function getPageDetail(name: string) {
  return await wFetch<{ page: ConsolePageDetail }>(
    `/pw-api/console/pages/detail?name=${encodeURIComponent(name)}`
  )
}

export async function deletePage(name: string) {
  return await wFetch<{ ok: boolean }>('/pw-api/console/pages/delete', {
    method: 'POST',
    sendJson: true,
    body: { name },
  })
}

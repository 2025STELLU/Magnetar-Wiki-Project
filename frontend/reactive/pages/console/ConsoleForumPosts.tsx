import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleForumCategory,
  ConsoleForumPost,
  ConsoleForumPostDetail,
  deleteForumPost,
  getForumPostDetail,
  listForumCategories,
  listForumPosts,
  lockForumPost,
} from '~api/console-forumposts'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const PAGE_SIZE = 20

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

const ConsoleForumPosts: React.FC = () => {
  const [query, setQuery] = useState('')
  const [categoryId, setCategoryId] = useState(0)
  const [categories, setCategories] = useState<ConsoleForumCategory[]>([])
  const [posts, setPosts] = useState<ConsoleForumPost[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detail, setDetail] = useState<ConsoleForumPostDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [acting, setActing] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const loadList = useConstCallback((q: string, cat: number, off: number) => {
    setLoading(true)
    setNotice(null)
    listForumPosts(q, cat, off)
      .then(resp => {
        setPosts(resp.posts)
        setTotal(resp.total)
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.forumposts.load_failed') }))
      .finally(() => setLoading(false))
  })

  useEffect(() => {
    listForumCategories()
      .then(resp => setCategories(resp.categories))
      .catch(() => {})
    loadList('', 0, 0)
  }, [loadList])

  const onSearch = useConstCallback(() => {
    setOffset(0)
    setSelectedId(null)
    setDetail(null)
    loadList(query, categoryId, 0)
  })

  const pickPost = useConstCallback((id: number) => {
    setSelectedId(id)
    setDetail(null)
    setConfirmingDelete(false)
    setNotice(null)
    setDetailLoading(true)
    getForumPostDetail(id)
      .then(resp => setDetail(resp.post))
      .catch(() => setNotice({ kind: 'error', text: t('console.forumposts.detail_failed') }))
      .finally(() => setDetailLoading(false))
  })

  const refreshDetail = useConstCallback((id: number) => {
    getForumPostDetail(id)
      .then(resp => {
        setDetail(resp.post)
        setPosts(prev => prev.map(p => (p.id === id ? { ...p, locked: resp.post.locked } : p)))
      })
      .catch(() => {})
  })

  const onDelete = useConstCallback(async () => {
    if (selectedId == null || acting) return
    if (!confirmingDelete) {
      setConfirmingDelete(true)
      return
    }
    setActing(true)
    try {
      await deleteForumPost(selectedId)
      setNotice({ kind: 'ok', text: t('console.forumposts.deleted_ok') })
      setSelectedId(null)
      setDetail(null)
      setConfirmingDelete(false)
      loadList(query, categoryId, offset)
    } catch {
      setNotice({ kind: 'error', text: t('console.forumposts.delete_failed') })
    } finally {
      setActing(false)
    }
  })

  const onLock = useConstCallback(async () => {
    if (detail == null || acting) return
    setActing(true)
    try {
      await lockForumPost(detail.id, !detail.locked)
      setNotice({
        kind: 'ok',
        text: detail.locked
          ? t('console.forumposts.unlocked_ok')
          : t('console.forumposts.locked_ok'),
      })
      refreshDetail(detail.id)
    } catch {
      setNotice({ kind: 'error', text: t('console.forumposts.lock_failed') })
    } finally {
      setActing(false)
    }
  })

  const goPage = useConstCallback((next: number) => {
    setOffset(next)
    setSelectedId(null)
    setDetail(null)
    loadList(query, categoryId, next)
  })

  return (
    <>
      <Styled.H1>{t('console.forumposts.title')}</Styled.H1>
      <Styled.Desc>{t('console.forumposts.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 8, margin: '12px 0' }}>
        <Styled.Input
          value={query}
          placeholder={t('console.forumposts.search_placeholder')}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') onSearch()
          }}
          style={{ flex: 1 }}
        />
        <select
          value={categoryId}
          onChange={e => setCategoryId(Number(e.target.value))}
          style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #ccc', fontSize: 14 }}
        >
          <option value={0}>{t('console.forumposts.all_categories')}</option>
          {categories.map(c => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <Styled.Button onClick={onSearch}>{t('console.forumposts.search')}</Styled.Button>
      </div>

      {loading ? (
        <Styled.Loading>{t('console.forumposts.loading')}</Styled.Loading>
      ) : posts.length === 0 ? (
        <Styled.Desc>{t('console.forumposts.no_posts')}</Styled.Desc>
      ) : (
        <>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.forumposts.thread_col')}</th>
                <th>{t('console.forumposts.author_col')}</th>
                <th>{t('console.forumposts.category_col')}</th>
                <th>{t('console.forumposts.status_col')}</th>
                <th>{t('console.forumposts.time_col')}</th>
              </tr>
            </thead>
            <tbody>
              {posts.map(p => (
                <tr
                  key={p.id}
                  onClick={() => pickPost(p.id)}
                  style={{ cursor: 'pointer', background: selectedId === p.id ? '#f0f6ff' : undefined }}
                >
                  <td>{p.title || '—'}</td>
                  <td>{p.author || '—'}</td>
                  <td>{p.category || '—'}</td>
                  <td>{p.locked ? t('console.forumposts.locked') : '—'}</td>
                  <td>{formatTime(p.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 12 }}>
            <Styled.Button onClick={() => goPage(offset - PAGE_SIZE)} disabled={offset === 0}>
              {t('console.forumposts.prev_page')}
            </Styled.Button>
            <span style={{ fontSize: 14, color: '#666' }}>
              {t('console.forumposts.page_info', { from: total === 0 ? 0 : offset + 1, to: Math.min(offset + PAGE_SIZE, total), total })}
            </span>
            <Styled.Button onClick={() => goPage(offset + PAGE_SIZE)} disabled={offset + PAGE_SIZE >= total}>
              {t('console.forumposts.next_page')}
            </Styled.Button>
          </div>
        </>
      )}

      {selectedId != null && (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 16 }}>
            {t('console.forumposts.detail_title', { id: selectedId })}
          </Styled.H1>
          {detailLoading ? (
            <Styled.Loading>{t('console.forumposts.loading')}</Styled.Loading>
          ) : detail ? (
            <>
              <Styled.Table>
                <tbody>
                  <tr>
                    <td style={{ width: 120 }}><strong>{t('console.forumposts.detail_thread')}</strong></td>
                    <td>{detail.title || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.forumposts.detail_author')}</strong></td>
                    <td>{detail.author || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.forumposts.detail_category')}</strong></td>
                    <td>{detail.category || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.forumposts.detail_replies')}</strong></td>
                    <td>{detail.post_count}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.forumposts.detail_locked')}</strong></td>
                    <td>{detail.locked ? t('console.forumposts.locked_yes') : t('console.forumposts.locked_no')}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.forumposts.detail_time')}</strong></td>
                    <td>{formatTime(detail.created_at)}</td>
                  </tr>
                </tbody>
              </Styled.Table>

              {detail.summary && (
                <Styled.Field>
                  <Styled.Label>{t('console.forumposts.detail_summary')}</Styled.Label>
                  <div style={{ background: '#f7f7f7', padding: 12, borderRadius: 6, fontSize: 14, whiteSpace: 'pre-wrap' }}>
                    {detail.summary}
                  </div>
                </Styled.Field>
              )}

              <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                <Styled.Button onClick={onLock} disabled={acting}>
                  {acting
                    ? t('console.forumposts.working')
                    : detail.locked
                      ? t('console.forumposts.unlock')
                      : t('console.forumposts.lock')}
                </Styled.Button>
                <Styled.Button
                  onClick={onDelete}
                  disabled={acting}
                  style={{ background: confirmingDelete ? '#c0392b' : '#999' }}
                >
                  {acting
                    ? t('console.forumposts.working')
                    : confirmingDelete
                      ? t('console.forumposts.delete_confirm')
                      : t('console.forumposts.delete')}
                </Styled.Button>
                {confirmingDelete && !acting && (
                  <Styled.Button onClick={() => setConfirmingDelete(false)}>
                    {t('console.forumposts.delete_cancel')}
                  </Styled.Button>
                )}
              </div>
            </>
          ) : null}
        </>
      )}
    </>
  )
}

export default ConsoleForumPosts

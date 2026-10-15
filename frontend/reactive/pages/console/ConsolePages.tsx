import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsolePage,
  ConsolePageDetail,
  deletePage,
  getPageDetail,
  listPages,
} from '~api/console-pages'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const PAGE_SIZE = 20

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

const ConsolePages: React.FC = () => {
  const [query, setQuery] = useState('')
  const [pages, setPages] = useState<ConsolePage[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [selectedName, setSelectedName] = useState<string | null>(null)
  const [detail, setDetail] = useState<ConsolePageDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const loadList = useConstCallback((q: string, off: number) => {
    setLoading(true)
    setNotice(null)
    listPages(q, off)
      .then(resp => {
        setPages(resp.pages)
        setTotal(resp.total)
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.pages.load_failed') }))
      .finally(() => setLoading(false))
  })

  useEffect(() => {
    loadList('', 0)
  }, [loadList])

  const onSearch = useConstCallback(() => {
    setOffset(0)
    setSelectedName(null)
    setDetail(null)
    loadList(query, 0)
  })

  const pickPage = useConstCallback((name: string) => {
    setSelectedName(name)
    setDetail(null)
    setConfirmingDelete(false)
    setNotice(null)
    setDetailLoading(true)
    getPageDetail(name)
      .then(resp => setDetail(resp.page))
      .catch(() => setNotice({ kind: 'error', text: t('console.pages.detail_failed') }))
      .finally(() => setDetailLoading(false))
  })

  const onDelete = useConstCallback(async () => {
    if (selectedName == null || deleting) return
    if (!confirmingDelete) {
      setConfirmingDelete(true)
      return
    }
    setDeleting(true)
    try {
      await deletePage(selectedName)
      setNotice({ kind: 'ok', text: t('console.pages.deleted_ok') })
      setSelectedName(null)
      setDetail(null)
      setConfirmingDelete(false)
      loadList(query, offset)
    } catch {
      setNotice({ kind: 'error', text: t('console.pages.delete_failed') })
    } finally {
      setDeleting(false)
    }
  })

  const goPage = useConstCallback((next: number) => {
    setOffset(next)
    setSelectedName(null)
    setDetail(null)
    loadList(query, next)
  })

  return (
    <>
      <Styled.H1>{t('console.pages.title')}</Styled.H1>
      <Styled.Desc>{t('console.pages.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 8, margin: '12px 0' }}>
        <Styled.Input
          value={query}
          placeholder={t('console.pages.search_placeholder')}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') onSearch()
          }}
          style={{ flex: 1 }}
        />
        <Styled.Button onClick={onSearch}>{t('console.pages.search')}</Styled.Button>
      </div>

      {loading ? (
        <Styled.Loading>{t('console.pages.loading')}</Styled.Loading>
      ) : pages.length === 0 ? (
        <Styled.Desc>{t('console.pages.no_pages')}</Styled.Desc>
      ) : (
        <>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.pages.name_col')}</th>
                <th>{t('console.pages.title_col')}</th>
                <th>{t('console.pages.revisions_col')}</th>
                <th>{t('console.pages.time_col')}</th>
              </tr>
            </thead>
            <tbody>
              {pages.map(p => (
                <tr
                  key={p.id}
                  onClick={() => pickPage(p.name)}
                  style={{ cursor: 'pointer', background: selectedName === p.name ? '#f0f6ff' : undefined }}
                >
                  <td style={{ fontFamily: 'monospace' }}>{p.name}</td>
                  <td>{p.title || '—'}</td>
                  <td>{p.revisions}</td>
                  <td>{formatTime(p.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 12 }}>
            <Styled.Button onClick={() => goPage(offset - PAGE_SIZE)} disabled={offset === 0}>
              {t('console.pages.prev_page')}
            </Styled.Button>
            <span style={{ fontSize: 14, color: '#666' }}>
              {t('console.pages.page_info', { from: total === 0 ? 0 : offset + 1, to: Math.min(offset + PAGE_SIZE, total), total })}
            </span>
            <Styled.Button onClick={() => goPage(offset + PAGE_SIZE)} disabled={offset + PAGE_SIZE >= total}>
              {t('console.pages.next_page')}
            </Styled.Button>
          </div>
        </>
      )}

      {selectedName != null && (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 16 }}>
            {t('console.pages.detail_title', { name: selectedName })}
          </Styled.H1>
          {detailLoading ? (
            <Styled.Loading>{t('console.pages.loading')}</Styled.Loading>
          ) : detail ? (
            <>
              <Styled.Table>
                <tbody>
                  <tr>
                    <td style={{ width: 120 }}><strong>{t('console.pages.detail_title_label')}</strong></td>
                    <td>{detail.title || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.pages.detail_revisions')}</strong></td>
                    <td>{detail.revisions}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.pages.detail_locked')}</strong></td>
                    <td>{detail.locked ? t('console.pages.locked_yes') : t('console.pages.locked_no')}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.pages.detail_updated')}</strong></td>
                    <td>{formatTime(detail.updated_at)}</td>
                  </tr>
                </tbody>
              </Styled.Table>

              {detail.summary && (
                <Styled.Field>
                  <Styled.Label>{t('console.pages.detail_summary')}</Styled.Label>
                  <div style={{ background: '#f7f7f7', padding: 12, borderRadius: 6, fontSize: 14, whiteSpace: 'pre-wrap' }}>
                    {detail.summary}
                  </div>
                </Styled.Field>
              )}

              <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                <Styled.Button
                  onClick={onDelete}
                  disabled={deleting}
                  style={{ background: confirmingDelete ? '#c0392b' : '#999' }}
                >
                  {deleting
                    ? t('console.pages.deleting')
                    : confirmingDelete
                      ? t('console.pages.delete_confirm')
                      : t('console.pages.delete')}
                </Styled.Button>
                {confirmingDelete && !deleting && (
                  <Styled.Button onClick={() => setConfirmingDelete(false)}>
                    {t('console.pages.delete_cancel')}
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

export default ConsolePages

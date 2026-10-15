import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleReport,
  ConsoleReportDetail,
  getReportDetail,
  handleReport,
  listReports,
  ReportTab,
} from '~api/console-reports'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const TABS: ReportTab[] = ['pending', 'handled', 'dismissed', 'all']
const PAGE_SIZE = 20

function statusLabel(status: string): string {
  if (status === 'pending') return t('console.reports.status_pending')
  if (status === 'reviewed') return t('console.reports.status_handled')
  return t('console.reports.status_dismissed')
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

const ConsoleReports: React.FC = () => {
  const [tab, setTab] = useState<ReportTab>('pending')
  const [reports, setReports] = useState<ConsoleReport[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detail, setDetail] = useState<ConsoleReportDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [note, setNote] = useState('')
  const [handling, setHandling] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const loadList = useConstCallback((s: ReportTab, off: number) => {
    setLoading(true)
    setNotice(null)
    listReports(s, off)
      .then(resp => {
        setReports(resp.reports)
        setTotal(resp.total)
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.reports.load_failed') }))
      .finally(() => setLoading(false))
  })

  // 切换 tab 时回到第一页并清空详情
  useEffect(() => {
    setOffset(0)
    setSelectedId(null)
    setDetail(null)
    loadList(tab, 0)
  }, [tab, loadList])

  const pickReport = useConstCallback((id: number) => {
    setSelectedId(id)
    setDetail(null)
    setNote('')
    setNotice(null)
    setDetailLoading(true)
    getReportDetail(id)
      .then(resp => {
        setDetail(resp.report)
        setNote(resp.report.admin_notes || '')
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.reports.detail_failed') }))
      .finally(() => setDetailLoading(false))
  })

  const onHandle = useConstCallback(async (action: 'dismiss' | 'action_taken') => {
    if (selectedId == null || handling) return
    setHandling(true)
    try {
      await handleReport(selectedId, action, note)
      setNotice({ kind: 'ok', text: t('console.reports.handled_ok') })
      loadList(tab, offset)
      // 刷新详情，显示最新状态
      const resp = await getReportDetail(selectedId)
      setDetail(resp.report)
    } catch {
      setNotice({ kind: 'error', text: t('console.reports.handle_failed') })
    } finally {
      setHandling(false)
    }
  })

  const goPage = useConstCallback((next: number) => {
    setOffset(next)
    setSelectedId(null)
    setDetail(null)
    loadList(tab, next)
  })

  return (
    <>
      <Styled.H1>{t('console.reports.title')}</Styled.H1>
      <Styled.Desc>{t('console.reports.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 8, margin: '12px 0' }}>
        {TABS.map(s => (
          <Styled.Button
            key={s}
            onClick={() => setTab(s)}
            style={tab === s ? undefined : { background: '#999' }}
          >
            {t(`console.reports.tab_${s}`)}
          </Styled.Button>
        ))}
      </div>

      {loading ? (
        <Styled.Loading>{t('console.reports.loading')}</Styled.Loading>
      ) : reports.length === 0 ? (
        <Styled.Desc>{t('console.reports.no_reports')}</Styled.Desc>
      ) : (
        <>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.reports.reporter_col')}</th>
                <th>{t('console.reports.reported_col')}</th>
                <th>{t('console.reports.reason_col')}</th>
                <th>{t('console.reports.status_col')}</th>
                <th>{t('console.reports.time_col')}</th>
              </tr>
            </thead>
            <tbody>
              {reports.map(r => (
                <tr
                  key={r.id}
                  onClick={() => pickReport(r.id)}
                  style={{ cursor: 'pointer', background: selectedId === r.id ? '#f0f6ff' : undefined }}
                >
                  <td>{r.reporter || '—'}</td>
                  <td>{r.reported || '—'}</td>
                  <td style={{ maxWidth: 320, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {r.reason}
                  </td>
                  <td>{statusLabel(r.status)}</td>
                  <td>{formatTime(r.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 12 }}>
            <Styled.Button onClick={() => goPage(offset - PAGE_SIZE)} disabled={offset === 0}>
              {t('console.reports.prev_page')}
            </Styled.Button>
            <span style={{ fontSize: 14, color: '#666' }}>
              {t('console.reports.page_info', { from: total === 0 ? 0 : offset + 1, to: Math.min(offset + PAGE_SIZE, total), total })}
            </span>
            <Styled.Button onClick={() => goPage(offset + PAGE_SIZE)} disabled={offset + PAGE_SIZE >= total}>
              {t('console.reports.next_page')}
            </Styled.Button>
          </div>
        </>
      )}

      {selectedId != null && (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 16 }}>
            {t('console.reports.detail_title', { id: selectedId })}
          </Styled.H1>
          {detailLoading ? (
            <Styled.Loading>{t('console.reports.loading')}</Styled.Loading>
          ) : detail ? (
            <>
              <Styled.Table>
                <tbody>
                  <tr>
                    <td style={{ width: 120 }}><strong>{t('console.reports.detail_reporter')}</strong></td>
                    <td>{detail.reporter || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.reports.detail_reported')}</strong></td>
                    <td>{detail.reported || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.reports.detail_reason')}</strong></td>
                    <td>{detail.reason}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.reports.detail_status')}</strong></td>
                    <td>{statusLabel(detail.status)}</td>
                  </tr>
                  {detail.reviewed_by && (
                    <tr>
                      <td><strong>{t('console.reports.detail_reviewed')}</strong></td>
                      <td>
                        {detail.reviewed_by}
                        {detail.reviewed_at && ` · ${formatTime(detail.reviewed_at)}`}
                      </td>
                    </tr>
                  )}
                </tbody>
              </Styled.Table>

              {Array.isArray(detail.messages) && detail.messages.length > 0 && (
                <Styled.Field>
                  <Styled.Label>{t('console.reports.detail_messages')}</Styled.Label>
                  <div style={{ background: '#f7f7f7', padding: 12, borderRadius: 6, fontSize: 14 }}>
                    {detail.messages.map((m, i) => (
                      <div key={i} style={{ marginBottom: 8 }}>
                        <strong>{m.sender_name}</strong>
                        <span style={{ color: '#888', marginLeft: 8 }}>{formatTime(m.created_at)}</span>
                        <div>{m.body}</div>
                      </div>
                    ))}
                  </div>
                </Styled.Field>
              )}

              <Styled.Field>
                <Styled.Label>{t('console.reports.detail_notes_label')}</Styled.Label>
                <textarea
                  value={note}
                  placeholder={t('console.reports.detail_notes_placeholder')}
                  onChange={e => setNote(e.target.value)}
                  rows={3}
                  style={{ width: '100%', padding: 8, fontSize: 14, border: '1px solid #ccc', borderRadius: 4 }}
                />
              </Styled.Field>

              <div style={{ display: 'flex', gap: 12 }}>
                <Styled.Button onClick={() => onHandle('action_taken')} disabled={handling}>
                  {handling ? t('console.reports.handling') : t('console.reports.action_taken')}
                </Styled.Button>
                <Styled.Button onClick={() => onHandle('dismiss')} disabled={handling} style={{ background: '#999' }}>
                  {t('console.reports.dismiss')}
                </Styled.Button>
              </div>
            </>
          ) : null}
        </>
      )}
    </>
  )
}

export default ConsoleReports

import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleTicket,
  ConsoleTicketDetail,
  getTicketDetail,
  handleTicket,
  listTickets,
  replyTicket,
  TicketAction,
  TicketKind,
  TicketStatus,
} from '~api/console-tickets'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const KINDS: TicketKind[] = ['ticket', 'application']
const STATUSES: TicketStatus[] = ['pending', 'approved', 'rejected', 'closed', 'all']
const PAGE_SIZE = 20

function statusLabel(status: string): string {
  if (status === 'pending') return t('console.tickets.status_pending')
  if (status === 'approved') return t('console.tickets.status_approved')
  if (status === 'rejected') return t('console.tickets.status_rejected')
  if (status === 'closed') return t('console.tickets.status_closed')
  return status
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

const ConsoleTickets: React.FC = () => {
  const [kind, setKind] = useState<TicketKind>('ticket')
  const [status, setStatus] = useState<TicketStatus>('pending')
  const [tickets, setTickets] = useState<ConsoleTicket[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detail, setDetail] = useState<ConsoleTicketDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [note, setNote] = useState('')
  const [replyText, setReplyText] = useState('')
  const [roleId, setRoleId] = useState('')
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const loadList = useConstCallback((k: TicketKind, s: TicketStatus, off: number) => {
    setLoading(true)
    setNotice(null)
    listTickets(k, s, off)
      .then(resp => {
        setTickets(resp.items)
        setTotal(resp.total)
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.tickets.load_failed') }))
      .finally(() => setLoading(false))
  })

  useEffect(() => {
    setOffset(0)
    setSelectedId(null)
    setDetail(null)
    loadList(kind, status, 0)
  }, [kind, status, loadList])

  const pickTicket = useConstCallback((id: number) => {
    setSelectedId(id)
    setDetail(null)
    setNote('')
    setReplyText('')
    setRoleId('')
    setNotice(null)
    setDetailLoading(true)
    getTicketDetail(id)
      .then(resp => {
        setDetail(resp.ticket)
        setNote(resp.ticket.admin_notes || '')
        setReplyText(resp.ticket.reply || '')
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.tickets.detail_failed') }))
      .finally(() => setDetailLoading(false))
  })

  const refreshDetail = useConstCallback(async (id: number) => {
    try {
      const resp = await getTicketDetail(id)
      setDetail(resp.ticket)
    } catch {
      // 列表已刷新，详情刷新失败不打扰
    }
  })

  const onReply = useConstCallback(async () => {
    if (selectedId == null || busy || replyText.trim() === '') return
    setBusy(true)
    try {
      await replyTicket(selectedId, replyText.trim())
      setNotice({ kind: 'ok', text: t('console.tickets.reply_ok') })
      await refreshDetail(selectedId)
    } catch {
      setNotice({ kind: 'error', text: t('console.tickets.reply_failed') })
    } finally {
      setBusy(false)
    }
  })

  const onHandle = useConstCallback(async (action: TicketAction) => {
    if (selectedId == null || busy) return
    setBusy(true)
    try {
      const rid = roleId.trim() === '' ? undefined : Number(roleId)
      await handleTicket(selectedId, action, note, rid)
      setNotice({ kind: 'ok', text: t('console.tickets.handle_ok') })
      loadList(kind, status, offset)
      await refreshDetail(selectedId)
    } catch {
      setNotice({ kind: 'error', text: t('console.tickets.handle_failed') })
    } finally {
      setBusy(false)
    }
  })

  const goPage = useConstCallback((next: number) => {
    setOffset(next)
    setSelectedId(null)
    setDetail(null)
    loadList(kind, status, next)
  })

  const isApplication = kind === 'application'

  return (
    <>
      <Styled.H1>{t('console.tickets.title')}</Styled.H1>
      <Styled.Desc>{t('console.tickets.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 8, margin: '12px 0' }}>
        {KINDS.map(k => (
          <Styled.Button
            key={k}
            onClick={() => setKind(k)}
            style={kind === k ? undefined : { background: '#999' }}
          >
            {t(`console.tickets.tab_${k}`)}
          </Styled.Button>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 8, margin: '0 0 12px' }}>
        {STATUSES.map(s => (
          <Styled.Button
            key={s}
            onClick={() => setStatus(s)}
            style={status === s ? undefined : { background: '#999' }}
          >
            {s === 'all' ? t('console.tickets.status_all') : statusLabel(s)}
          </Styled.Button>
        ))}
      </div>

      {loading ? (
        <Styled.Loading>{t('console.tickets.loading')}</Styled.Loading>
      ) : tickets.length === 0 ? (
        <Styled.Desc>{t('console.tickets.no_items')}</Styled.Desc>
      ) : (
        <>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.tickets.author_col')}</th>
                <th>{t('console.tickets.subject_col')}</th>
                <th>{t('console.tickets.status_col')}</th>
                <th>{t('console.tickets.time_col')}</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map(item => (
                <tr
                  key={item.id}
                  onClick={() => pickTicket(item.id)}
                  style={{ cursor: 'pointer', background: selectedId === item.id ? '#f0f6ff' : undefined }}
                >
                  <td>{item.author || '—'}</td>
                  <td style={{ maxWidth: 320, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {item.subject || '—'}
                  </td>
                  <td>{statusLabel(item.status)}</td>
                  <td>{formatTime(item.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 12 }}>
            <Styled.Button onClick={() => goPage(offset - PAGE_SIZE)} disabled={offset === 0}>
              {t('console.tickets.prev_page')}
            </Styled.Button>
            <span style={{ fontSize: 14, color: '#666' }}>
              {t('console.tickets.page_info', { from: total === 0 ? 0 : offset + 1, to: Math.min(offset + PAGE_SIZE, total), total })}
            </span>
            <Styled.Button onClick={() => goPage(offset + PAGE_SIZE)} disabled={offset + PAGE_SIZE >= total}>
              {t('console.tickets.next_page')}
            </Styled.Button>
          </div>
        </>
      )}

      {selectedId != null && (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 16 }}>
            {t('console.tickets.detail_title', { id: selectedId })}
          </Styled.H1>
          {detailLoading ? (
            <Styled.Loading>{t('console.tickets.loading')}</Styled.Loading>
          ) : detail ? (
            <>
              <Styled.Table>
                <tbody>
                  <tr>
                    <td style={{ width: 120 }}><strong>{t('console.tickets.detail_author')}</strong></td>
                    <td>{detail.author || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.tickets.detail_subject')}</strong></td>
                    <td>{detail.subject || '—'}</td>
                  </tr>
                  <tr>
                    <td><strong>{t('console.tickets.detail_status')}</strong></td>
                    <td>{statusLabel(detail.status)}</td>
                  </tr>
                  {detail.source_page && (
                    <tr>
                      <td><strong>{t('console.tickets.detail_source')}</strong></td>
                      <td>{detail.source_page}</td>
                    </tr>
                  )}
                  {detail.reviewed_by && (
                    <tr>
                      <td><strong>{t('console.tickets.detail_reviewed')}</strong></td>
                      <td>
                        {detail.reviewed_by}
                        {detail.reviewed_at && ` · ${formatTime(detail.reviewed_at)}`}
                      </td>
                    </tr>
                  )}
                </tbody>
              </Styled.Table>

              <Styled.Field>
                <Styled.Label>{t('console.tickets.conversation_label')}</Styled.Label>
                <div style={{ background: '#f7f7f7', padding: 12, borderRadius: 6, fontSize: 14 }}>
                  {detail.messages.map((m, i) => (
                    <div key={i} style={{ marginBottom: 8 }}>
                      <strong>{m.sender_name || '—'}</strong>
                      <span style={{ color: '#888', marginLeft: 8 }}>{formatTime(m.created_at)}</span>
                      <div style={{ whiteSpace: 'pre-wrap' }}>{m.body}</div>
                    </div>
                  ))}
                </div>
              </Styled.Field>

              <Styled.Field>
                <Styled.Label>{t('console.tickets.reply_label')}</Styled.Label>
                <textarea
                  value={replyText}
                  placeholder={t('console.tickets.reply_placeholder')}
                  onChange={e => setReplyText(e.target.value)}
                  rows={3}
                  style={{ width: '100%', padding: 8, fontSize: 14, border: '1px solid #ccc', borderRadius: 4 }}
                />
                <div style={{ marginTop: 8 }}>
                  <Styled.Button onClick={onReply} disabled={busy || replyText.trim() === ''}>
                    {busy ? t('console.tickets.reply_sending') : t('console.tickets.reply_send')}
                  </Styled.Button>
                </div>
              </Styled.Field>

              <Styled.Field>
                <Styled.Label>{t('console.tickets.detail_notes_label')}</Styled.Label>
                <textarea
                  value={note}
                  placeholder={t('console.tickets.detail_notes_placeholder')}
                  onChange={e => setNote(e.target.value)}
                  rows={2}
                  style={{ width: '100%', padding: 8, fontSize: 14, border: '1px solid #ccc', borderRadius: 4 }}
                />
              </Styled.Field>

              {isApplication && (
                <Styled.Field>
                  <Styled.Label>{t('console.tickets.role_id_label')}</Styled.Label>
                  <Styled.Input
                    value={roleId}
                    placeholder={t('console.tickets.role_id_placeholder')}
                    onChange={e => setRoleId(e.target.value.replace(/[^0-9]/g, ''))}
                    style={{ width: 160 }}
                  />
                </Styled.Field>
              )}

              <div style={{ display: 'flex', gap: 12 }}>
                {isApplication && (
                  <>
                    <Styled.Button onClick={() => onHandle('approve')} disabled={busy}>
                      {busy ? t('console.tickets.handling') : t('console.tickets.approve')}
                    </Styled.Button>
                    <Styled.Button onClick={() => onHandle('reject')} disabled={busy} style={{ background: '#c05353' }}>
                      {t('console.tickets.reject')}
                    </Styled.Button>
                  </>
                )}
                <Styled.Button onClick={() => onHandle('close')} disabled={busy} style={{ background: '#999' }}>
                  {t('console.tickets.close')}
                </Styled.Button>
              </div>
            </>
          ) : null}
        </>
      )}
    </>
  )
}

export default ConsoleTickets

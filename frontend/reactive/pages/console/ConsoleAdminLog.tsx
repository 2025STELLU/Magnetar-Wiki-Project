import * as React from 'react'
import { useEffect, useState } from 'react'
import { AdminLogAction, AdminLogEntry, listAdminLog } from '~api/console-adminlog'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const PAGE_SIZE = 50
const ACTIONS: AdminLogAction[] = ['', 'create', 'change', 'delete']

function actionLabel(action: string): string {
  if (action === 'create') return t('console.adminlog.action_create')
  if (action === 'change') return t('console.adminlog.action_change')
  if (action === 'delete') return t('console.adminlog.action_delete')
  return action || '—'
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

const ConsoleAdminLog: React.FC = () => {
  const [actorInput, setActorInput] = useState('')
  const [actor, setActor] = useState('')
  const [action, setAction] = useState<AdminLogAction>('')
  const [entries, setEntries] = useState<AdminLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const loadList = useConstCallback((a: string, act: AdminLogAction, off: number) => {
    setLoading(true)
    setNotice(null)
    listAdminLog(a, act, off)
      .then(resp => {
        setEntries(resp.entries)
        setTotal(resp.total)
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.adminlog.load_failed') }))
      .finally(() => setLoading(false))
  })

  // 初次加载
  useEffect(() => {
    loadList('', '', 0)
  }, [loadList])

  const onSearch = useConstCallback(() => {
    const a = actorInput.trim()
    setActor(a)
    setOffset(0)
    loadList(a, action, 0)
  })

  const onActionChange = useConstCallback((act: AdminLogAction) => {
    setAction(act)
    setOffset(0)
    loadList(actor, act, 0)
  })

  const goPage = useConstCallback((next: number) => {
    setOffset(next)
    loadList(actor, action, next)
  })

  return (
    <>
      <Styled.H1>{t('console.adminlog.title')}</Styled.H1>
      <Styled.Desc>{t('console.adminlog.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 12, alignItems: 'flex-end', margin: '12px 0', flexWrap: 'wrap' }}>
        <Styled.Field style={{ margin: 0 }}>
          <Styled.Label>{t('console.adminlog.actor_filter')}</Styled.Label>
          <Styled.Input
            value={actorInput}
            placeholder={t('console.adminlog.actor_placeholder')}
            onChange={e => setActorInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') onSearch()
            }}
            style={{ width: 200 }}
          />
        </Styled.Field>
        <Styled.Field style={{ margin: 0 }}>
          <Styled.Label>{t('console.adminlog.action_filter')}</Styled.Label>
          <select
            value={action}
            onChange={e => onActionChange(e.target.value as AdminLogAction)}
            style={{ padding: '8px', fontSize: 14, border: '1px solid #ccc', borderRadius: 4 }}
          >
            {ACTIONS.map(a => (
              <option key={a} value={a}>
                {a === '' ? t('console.adminlog.action_all') : actionLabel(a)}
              </option>
            ))}
          </select>
        </Styled.Field>
        <Styled.Button onClick={onSearch}>{t('console.adminlog.search')}</Styled.Button>
      </div>

      {loading ? (
        <Styled.Loading>{t('console.adminlog.loading')}</Styled.Loading>
      ) : entries.length === 0 ? (
        <Styled.Desc>{t('console.adminlog.no_entries')}</Styled.Desc>
      ) : (
        <>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.adminlog.time_col')}</th>
                <th>{t('console.adminlog.actor_col')}</th>
                <th>{t('console.adminlog.action_col')}</th>
                <th>{t('console.adminlog.target_col')}</th>
                <th>{t('console.adminlog.detail_col')}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map(e => (
                <tr key={e.id}>
                  <td style={{ whiteSpace: 'nowrap' }}>{formatTime(e.created_at)}</td>
                  <td>{e.actor || '—'}</td>
                  <td>{actionLabel(e.action)}</td>
                  <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {e.target || '—'}
                  </td>
                  <td style={{ maxWidth: 360, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {e.detail || '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginTop: 12 }}>
            <Styled.Button onClick={() => goPage(offset - PAGE_SIZE)} disabled={offset === 0}>
              {t('console.adminlog.prev_page')}
            </Styled.Button>
            <span style={{ fontSize: 14, color: '#666' }}>
              {t('console.adminlog.page_info', { from: total === 0 ? 0 : offset + 1, to: Math.min(offset + PAGE_SIZE, total), total })}
            </span>
            <Styled.Button onClick={() => goPage(offset + PAGE_SIZE)} disabled={offset + PAGE_SIZE >= total}>
              {t('console.adminlog.next_page')}
            </Styled.Button>
          </div>
        </>
      )}
    </>
  )
}

export default ConsoleAdminLog

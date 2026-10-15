import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  clearSuspiciousAddresses,
  ConsoleInvite,
  createInvite,
  listInvites,
  listSuspiciousUsers,
  revokeInvite,
  SuspiciousUser,
} from '~api/console-invites'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

type Tab = 'invites' | 'suspicious'

const ConsoleInvites: React.FC = () => {
  const [tab, setTab] = useState<Tab>('invites')
  const [invites, setInvites] = useState<ConsoleInvite[]>([])
  const [susUsers, setSusUsers] = useState<SuspiciousUser[]>([])
  const [susForbidden, setSusForbidden] = useState(false)
  const [loading, setLoading] = useState(true)
  const [email, setEmail] = useState('')
  const [creating, setCreating] = useState(false)
  const [newUrl, setNewUrl] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const reloadInvites = useConstCallback(async () => {
    try {
      const r = await listInvites()
      setInvites(r.invites)
    } catch {
      setNotice({ kind: 'error', text: t('console.invites.load_failed') })
    }
  })

  const reloadSuspicious = useConstCallback(async () => {
    try {
      const r = await listSuspiciousUsers()
      setSusUsers(r.users)
      setSusForbidden(false)
    } catch (e: any) {
      // 403 = 不是超管：只显示提示，不报错
      if (e && (e.status === 403 || e.status === 401)) {
        setSusForbidden(true)
      } else {
        setNotice({ kind: 'error', text: t('console.invites.load_failed') })
      }
    }
  })

  useEffect(() => {
    setLoading(true)
    Promise.all([reloadInvites(), reloadSuspicious()]).finally(() => setLoading(false))
  }, [reloadInvites, reloadSuspicious])

  const onCreate = useConstCallback(async () => {
    if (creating) return
    const mail = email.trim()
    if (!mail || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(mail)) {
      setNotice({ kind: 'error', text: t('console.invites.bad_email') })
      return
    }
    setCreating(true)
    setNotice(null)
    setNewUrl(null)
    try {
      const r = await createInvite(mail)
      setNewUrl(r.url)
      setEmail('')
      setNotice({ kind: 'ok', text: t('console.invites.created') })
      await reloadInvites()
    } catch {
      setNotice({ kind: 'error', text: t('console.invites.create_failed') })
    } finally {
      setCreating(false)
    }
  })

  const onCopy = useConstCallback(async () => {
    if (!newUrl) return
    try {
      await navigator.clipboard.writeText(newUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // 剪贴板不可用时，用户手动复制
    }
  })

  const onRevoke = useConstCallback(async (id: number) => {
    if (!window.confirm(t('console.invites.revoke_confirm'))) return
    try {
      await revokeInvite(id)
      setNotice({ kind: 'ok', text: t('console.invites.revoked') })
      await reloadInvites()
    } catch {
      setNotice({ kind: 'error', text: t('console.invites.revoke_failed') })
    }
  })

  const onClearSus = useConstCallback(async (userId?: number) => {
    if (!window.confirm(t('console.invites.clear_confirm'))) return
    try {
      const r = await clearSuspiciousAddresses(userId)
      setNotice({
        kind: 'ok',
        text: t('console.invites.cleared', { n: String(r.cleared) }),
      })
      await reloadSuspicious()
    } catch {
      setNotice({ kind: 'error', text: t('console.invites.clear_failed') })
    }
  })

  const tabBtn = (key: Tab, label: string) => (
    <Styled.Button
      key={key}
      onClick={() => setTab(key)}
      style={{
        background: tab === key ? undefined : '#e8e8e8',
        color: tab === key ? undefined : '#333',
      }}
    >
      {label}
    </Styled.Button>
  )

  return (
    <>
      <Styled.H1>{t('console.invites.title')}</Styled.H1>
      <Styled.Desc>{t('console.invites.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        {tabBtn('invites', t('console.invites.tab_invites'))}
        {tabBtn('suspicious', t('console.invites.tab_suspicious'))}
      </div>

      {loading ? (
        <Styled.Loading>{t('console.invites.loading')}</Styled.Loading>
      ) : tab === 'invites' ? (
        <>
          <Styled.Field>
            <Styled.Label>{t('console.invites.email_label')}</Styled.Label>
            <div style={{ display: 'flex', gap: 8 }}>
              <Styled.Input
                value={email}
                placeholder="user@example.com"
                onChange={e => setEmail(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') onCreate()
                }}
                style={{ flex: 1 }}
              />
              <Styled.Button onClick={onCreate} disabled={creating}>
                {creating ? t('console.invites.creating') : t('console.invites.create')}
              </Styled.Button>
            </div>
            <Styled.Desc>{t('console.invites.create_hint')}</Styled.Desc>
          </Styled.Field>

          {newUrl && (
            <Styled.Notice $kind="ok">
              <div style={{ marginBottom: 8 }}>{t('console.invites.new_url')}</div>
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <code
                  style={{
                    flex: 1,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    background: '#f4f4f4',
                    padding: '6px 8px',
                    borderRadius: 4,
                  }}
                >
                  {newUrl}
                </code>
                <Styled.Button onClick={onCopy}>
                  {copied ? t('console.invites.copied') : t('console.invites.copy')}
                </Styled.Button>
              </div>
            </Styled.Notice>
          )}

          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.invites.col_email')}</th>
                <th>{t('console.invites.col_status')}</th>
                <th>{t('console.invites.col_created')}</th>
                <th>{t('console.invites.col_action')}</th>
              </tr>
            </thead>
            <tbody>
              {invites.length === 0 && (
                <tr>
                  <td colSpan={4} style={{ color: '#888' }}>
                    {t('console.invites.empty')}
                  </td>
                </tr>
              )}
              {invites.map(inv => (
                <tr key={inv.id}>
                  <td>{inv.email || <span style={{ color: '#888' }}>—</span>}</td>
                  <td>
                    {inv.activated_at ? (
                      <span style={{ color: '#888' }}>
                        {t('console.invites.used', { by: inv.activated_by || '' })}
                      </span>
                    ) : (
                      <span style={{ color: '#2a7' }}>{t('console.invites.unused')}</span>
                    )}
                  </td>
                  <td style={{ color: '#888', fontSize: 13 }}>
                    {new Date(inv.created_at).toLocaleString()}
                  </td>
                  <td>
                    {!inv.activated_at && (
                      <Styled.Button
                        onClick={() => onRevoke(inv.id)}
                        style={{ background: '#a33', padding: '4px 10px', fontSize: 13 }}
                      >
                        {t('console.invites.revoke')}
                      </Styled.Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
        </>
      ) : susForbidden ? (
        <Styled.Notice>{t('console.invites.sus_forbidden')}</Styled.Notice>
      ) : (
        <>
          <Styled.Desc>{t('console.invites.sus_desc')}</Styled.Desc>
          <div style={{ marginBottom: 12 }}>
            <Styled.Button
              onClick={() => onClearSus()}
              style={{ background: '#a33' }}
            >
              {t('console.invites.clear_all')}
            </Styled.Button>
          </div>
          <Styled.Table>
            <thead>
              <tr>
                <th>{t('console.invites.col_user')}</th>
                <th>{t('console.invites.col_ip')}</th>
                <th>{t('console.invites.col_action')}</th>
              </tr>
            </thead>
            <tbody>
              {susUsers.length === 0 && (
                <tr>
                  <td colSpan={3} style={{ color: '#888' }}>
                    {t('console.invites.sus_empty')}
                  </td>
                </tr>
              )}
              {susUsers.map(u => (
                <tr key={`${u.id}-${u.ip || ''}`}>
                  <td>
                    <a href={`/-/users/${u.id}-${encodeURIComponent(u.username)}`}>
                      {u.username}
                    </a>
                  </td>
                  <td>
                    <code>{u.ip || <span style={{ color: '#888' }}>—</span>}</code>
                  </td>
                  <td>
                    <Styled.Button
                      onClick={() => onClearSus(u.id)}
                      style={{ background: '#a33', padding: '4px 10px', fontSize: 13 }}
                    >
                      {t('console.invites.clear_user')}
                    </Styled.Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </Styled.Table>
        </>
      )}
    </>
  )
}

export default ConsoleInvites

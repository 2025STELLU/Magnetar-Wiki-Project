import * as React from 'react'
import { useEffect, useState } from 'react'
import { t } from '~util/i18n'
import { activateTheme, checkUpdate, getThemes, ThemeInfo, ThemesResponse } from '~api/console-themes'
import useConstCallback from '~util/const-callback'
import * as Styled from './Console.styles'

const ConsoleThemes: React.FC = () => {
  const [themes, setThemes] = useState<ThemeInfo[]>([])
  const [update, setUpdate] = useState<ThemesResponse['update'] | null>(null)
  const [isSuperuser, setIsSuperuser] = useState(false)
  const [loading, setLoading] = useState(true)
  const [activating, setActivating] = useState<number | null>(null)
  const [checking, setChecking] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    let live = true
    getThemes()
      .then(resp => {
        if (live) {
          setThemes(resp.themes)
          setUpdate(resp.update)
        }
      })
      .catch((e: any) => {
        if (live) {
          // Non-superusers without ManageSite land here; superuser-only
          // update section is hidden when the whole page is forbidden.
          if (e && e.status === 403) setIsSuperuser(false)
          setError(t('console.themes.load_failed'))
        }
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [])

  // The update section is superuser-only; probe it once via check_update.
  // A 403 there simply hides the section instead of failing the page.
  useEffect(() => {
    let live = true
    checkUpdate()
      .then(resp => {
        if (live) {
          setIsSuperuser(true)
          setUpdate(prev =>
            prev ? { ...prev, latest_version: resp.latest_version, has_update: resp.has_update } : prev
          )
        }
      })
      .catch(() => {
        if (live) setIsSuperuser(false)
      })
    return () => {
      live = false
    }
  }, [])

  const onActivate = useConstCallback(async (id: number) => {
    if (activating !== null) return
    setActivating(id)
    setError(null)
    setNotice(null)
    try {
      await activateTheme(id)
      setThemes(prev => prev.map(th => ({ ...th, active: th.id === id })))
      setNotice(t('console.themes.activated'))
    } catch {
      setError(t('console.themes.activate_failed'))
    } finally {
      setActivating(null)
    }
  })

  const onCheck = useConstCallback(async () => {
    if (checking) return
    setChecking(true)
    setError(null)
    setNotice(null)
    try {
      const resp = await checkUpdate()
      setUpdate(prev =>
        prev ? { ...prev, latest_version: resp.latest_version, has_update: resp.has_update } : prev
      )
      setNotice(
        resp.has_update
          ? t('console.themes.update_available', { version: resp.latest_version })
          : t('console.themes.update_newest')
      )
    } catch {
      setError(t('console.themes.check_failed'))
    } finally {
      setChecking(false)
    }
  })

  if (loading) return <Styled.Loading>{t('console.themes.loading')}</Styled.Loading>

  return (
    <div>
      <Styled.H1>{t('console.themes.title')}</Styled.H1>
      <Styled.Desc>{t('console.themes.desc')}</Styled.Desc>
      {error && <Styled.Notice $kind="error">{error}</Styled.Notice>}
      {notice && <Styled.Notice $kind="ok">{notice}</Styled.Notice>}

      <Styled.Table>
        <thead>
          <tr>
            <th>{t('console.themes.col_name')}</th>
            <th>{t('console.themes.col_slug')}</th>
            <th>{t('console.themes.col_status')}</th>
            <th>{t('console.themes.col_action')}</th>
          </tr>
        </thead>
        <tbody>
          {themes.map(th => (
            <tr key={th.id}>
              <td>{th.name}</td>
              <td>
                <code>{th.slug}</code>
              </td>
              <td>{th.active ? t('console.themes.active') : '—'}</td>
              <td>
                {!th.active && (
                  <Styled.Button onClick={() => onActivate(th.id)} disabled={activating !== null}>
                    {activating === th.id ? t('console.themes.activating') : t('console.themes.activate')}
                  </Styled.Button>
                )}
              </td>
            </tr>
          ))}
          {themes.length === 0 && (
            <tr>
              <td colSpan={4}>{t('console.themes.empty')}</td>
            </tr>
          )}
        </tbody>
      </Styled.Table>

      {isSuperuser && update && (
        <div style={{ marginTop: '2rem' }}>
          <Styled.H1 style={{ fontSize: '1.1rem' }}>{t('console.themes.update_title')}</Styled.H1>
          <Styled.Desc>{t('console.themes.update_desc')}</Styled.Desc>
          <Styled.CardGrid>
            <Styled.Card>
              <Styled.CardLabel>{t('console.themes.current_version')}</Styled.CardLabel>
              <Styled.CardValue>{update.current_version || '—'}</Styled.CardValue>
            </Styled.Card>
            <Styled.Card>
              <Styled.CardLabel>{t('console.themes.latest_version')}</Styled.CardLabel>
              <Styled.CardValue>{update.latest_version || t('console.themes.unknown')}</Styled.CardValue>
            </Styled.Card>
          </Styled.CardGrid>
          <Styled.Field>
            <Styled.Button onClick={onCheck} disabled={checking}>
              {checking ? t('console.themes.checking') : t('console.themes.check_update')}
            </Styled.Button>
            {update.has_update && (
              <Styled.Button
                as="a"
                href="/-/admin/update"
                style={{ marginLeft: '0.5rem', textDecoration: 'none', display: 'inline-block' }}
              >
                {t('console.themes.go_update')}
              </Styled.Button>
            )}
          </Styled.Field>
          {update.has_update && (
            <Styled.Notice $kind="error">
              {t('console.themes.update_available', { version: update.latest_version })}
            </Styled.Notice>
          )}
        </div>
      )}
    </div>
  )
}

export default ConsoleThemes

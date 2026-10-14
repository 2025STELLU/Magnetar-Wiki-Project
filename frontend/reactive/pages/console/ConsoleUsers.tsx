import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import {
  EMPTY_NAME_STYLE,
  getUserNameStyle,
  NameStyleConfig,
  saveUserNameStyle,
  searchConsoleUsers,
  ConsoleUserEntry,
} from '~api/console'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import { nameStyleProps } from '~util/user-view'
import * as Styled from './Console.styles'

const DIRECTIONS = [
  'to right',
  'to left',
  'to top',
  'to bottom',
  'to top right',
  'to bottom right',
  'to bottom left',
  'to top left',
]

const ConsoleUsers: React.FC = () => {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<ConsoleUserEntry[]>([])
  const [searching, setSearching] = useState(false)
  const [selected, setSelected] = useState<ConsoleUserEntry | null>(null)
  const [style, setStyle] = useState<NameStyleConfig>({ ...EMPTY_NAME_STYLE })
  const [useGradient, setUseGradient] = useState(false)
  const [styleLoading, setStyleLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // 搜索框 debounce 300ms，至少 2 个字符才搜
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current)
    const q = query.trim()
    if (q.length < 2) {
      setResults([])
      setSearching(false)
      return
    }
    setSearching(true)
    timer.current = setTimeout(() => {
      searchConsoleUsers(q)
        .then(resp => setResults(resp.users))
        .catch(() => setNotice({ kind: 'error', text: t('console.users.search_failed') }))
        .finally(() => setSearching(false))
    }, 300)
    return () => {
      if (timer.current) clearTimeout(timer.current)
    }
  }, [query])

  const pickUser = useConstCallback((u: ConsoleUserEntry) => {
    setSelected(u)
    setNotice(null)
    setStyleLoading(true)
    getUserNameStyle(u.id)
      .then(resp => {
        const s = resp.style || { ...EMPTY_NAME_STYLE }
        setStyle(s)
        setUseGradient(!!(s.gradient_from && s.gradient_to))
      })
      .catch(() => setNotice({ kind: 'error', text: t('console.users.load_failed') }))
      .finally(() => setStyleLoading(false))
  })

  const set = useConstCallback((patch: Partial<NameStyleConfig>) => {
    setStyle(prev => ({ ...prev, ...patch }))
    setNotice(null)
  })

  const onSave = useConstCallback(async () => {
    if (!selected || saving) return
    const payload = useGradient ? style : { ...style, gradient_from: '', gradient_to: '', gradient_direction: '' }
    setSaving(true)
    try {
      await saveUserNameStyle(selected.id, payload)
      setNotice({ kind: 'ok', text: t('console.users.saved') })
    } catch {
      setNotice({ kind: 'error', text: t('console.users.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onClear = useConstCallback(async () => {
    if (!selected || saving) return
    setSaving(true)
    try {
      await saveUserNameStyle(selected.id, { ...EMPTY_NAME_STYLE })
      setStyle({ ...EMPTY_NAME_STYLE })
      setUseGradient(false)
      setNotice({ kind: 'ok', text: t('console.users.cleared') })
    } catch {
      setNotice({ kind: 'error', text: t('console.users.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const previewName = selected ? selected.display_name || selected.username : ''

  return (
    <>
      <Styled.H1>{t('console.users.title')}</Styled.H1>
      <Styled.Desc>{t('console.users.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      <Styled.Field>
        <Styled.Label>{t('console.users.search_label')}</Styled.Label>
        <Styled.Input
          value={query}
          placeholder={t('console.users.search_placeholder')}
          onChange={e => setQuery(e.target.value)}
        />
      </Styled.Field>

      {searching && <Styled.Loading>{t('console.users.searching')}</Styled.Loading>}
      {!searching && query.trim().length >= 2 && results.length === 0 && (
        <Styled.Desc>{t('console.users.no_results')}</Styled.Desc>
      )}
      {results.length > 0 && (
        <Styled.Table>
          <tbody>
            {results.map(u => (
              <tr
                key={u.id}
                onClick={() => pickUser(u)}
                style={{ cursor: 'pointer', background: selected?.id === u.id ? '#f0f6ff' : undefined }}
              >
                <td>
                  <strong>{u.username}</strong>
                  {u.display_name && u.display_name !== u.username && (
                    <span style={{ marginLeft: 8, color: '#888' }}>{u.display_name}</span>
                  )}
                </td>
                <td>Lv{u.level}</td>
              </tr>
            ))}
          </tbody>
        </Styled.Table>
      )}

      {selected && (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 8 }}>
            {t('console.users.editing', { name: selected.username })}
          </Styled.H1>
          {styleLoading ? (
            <Styled.Loading>{t('console.users.loading')}</Styled.Loading>
          ) : (
            <>
              <Styled.Field>
                <Styled.Label>{t('console.users.preview_label')}</Styled.Label>
                <div style={{ fontSize: 28, fontWeight: 700, padding: '12px 0' }}>
                  <span style={nameStyleProps(useGradient ? style : { ...style, gradient_from: '', gradient_to: '' })}>
                    {previewName}
                  </span>
                </div>
              </Styled.Field>

              <Styled.Field>
                <Styled.Label>{t('console.users.font_label')}</Styled.Label>
                <Styled.Input
                  value={style.font_family}
                  placeholder={t('console.users.font_placeholder')}
                  onChange={e => set({ font_family: e.target.value })}
                />
              </Styled.Field>

              <Styled.Field>
                <Styled.Label>{t('console.users.color_label')}</Styled.Label>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <input
                    type="color"
                    value={/^#[0-9a-fA-F]{6}$/.test(style.color) ? style.color : '#000000'}
                    onChange={e => set({ color: e.target.value })}
                  />
                  <Styled.Input
                    value={style.color}
                    placeholder="#rrggbb"
                    maxLength={7}
                    style={{ maxWidth: 140 }}
                    onChange={e => set({ color: e.target.value })}
                  />
                </div>
              </Styled.Field>

              <Styled.Field>
                <label style={{ fontSize: 14, display: 'flex', gap: 8, alignItems: 'center', cursor: 'pointer' }}>
                  <input type="checkbox" checked={useGradient} onChange={e => setUseGradient(e.target.checked)} />
                  {t('console.users.gradient_label')}
                </label>
              </Styled.Field>

              {useGradient && (
                <>
                  <Styled.Field>
                    <Styled.Label>{t('console.users.gradient_from')}</Styled.Label>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      <input
                        type="color"
                        value={/^#[0-9a-fA-F]{6}$/.test(style.gradient_from) ? style.gradient_from : '#000000'}
                        onChange={e => set({ gradient_from: e.target.value })}
                      />
                      <Styled.Input
                        value={style.gradient_from}
                        placeholder="#rrggbb"
                        maxLength={7}
                        style={{ maxWidth: 140 }}
                        onChange={e => set({ gradient_from: e.target.value })}
                      />
                    </div>
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.users.gradient_to')}</Styled.Label>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      <input
                        type="color"
                        value={/^#[0-9a-fA-F]{6}$/.test(style.gradient_to) ? style.gradient_to : '#000000'}
                        onChange={e => set({ gradient_to: e.target.value })}
                      />
                      <Styled.Input
                        value={style.gradient_to}
                        placeholder="#rrggbb"
                        maxLength={7}
                        style={{ maxWidth: 140 }}
                        onChange={e => set({ gradient_to: e.target.value })}
                      />
                    </div>
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.users.gradient_direction')}</Styled.Label>
                    <select
                      value={style.gradient_direction || 'to right'}
                      onChange={e => set({ gradient_direction: e.target.value })}
                      style={{ padding: '8px 10px', fontSize: 14 }}
                    >
                      {DIRECTIONS.map(d => (
                        <option key={d} value={d}>
                          {t(`console.users.dir_${d.replace(/ /g, '_')}`)}
                        </option>
                      ))}
                    </select>
                  </Styled.Field>
                </>
              )}

              <div style={{ display: 'flex', gap: 12 }}>
                <Styled.Button onClick={onSave} disabled={saving}>
                  {saving ? t('console.users.saving') : t('console.users.save')}
                </Styled.Button>
                <Styled.Button onClick={onClear} disabled={saving} style={{ background: '#999' }}>
                  {t('console.users.clear')}
                </Styled.Button>
              </div>
            </>
          )}
        </>
      )}
    </>
  )
}

export default ConsoleUsers

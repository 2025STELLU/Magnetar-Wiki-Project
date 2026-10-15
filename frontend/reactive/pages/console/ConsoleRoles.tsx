import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleRole,
  deleteRole,
  listPermissions,
  listRoles,
  PermissionInfo,
  saveRole,
} from '~api/console-roles'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

// 后端 catalog group key → 展示分组
const GROUP_ORDER = ['site', 'users', 'content', 'forum', 'other'] as const
const GROUP_OF: Record<string, (typeof GROUP_ORDER)[number]> = {
  admin: 'site',
  members: 'users',
  social: 'users',
  articles: 'content',
  forum: 'forum',
  tickets: 'other',
}

const groupLabel = (g: string) => t(`console.roles.group_${g}`)

const ConsoleRoles: React.FC = () => {
  const [roles, setRoles] = useState<ConsoleRole[]>([])
  const [permissions, setPermissions] = useState<PermissionInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedId, setSelectedId] = useState<number | 'new' | null>(null)
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const reload = useConstCallback(async () => {
    setLoading(true)
    try {
      const [r, p] = await Promise.all([listRoles(), listPermissions()])
      setRoles(r.roles)
      setPermissions(p.permissions)
    } catch {
      setNotice({ kind: 'error', text: t('console.roles.load_failed') })
    } finally {
      setLoading(false)
    }
  })

  useEffect(() => {
    reload()
  }, [reload])

  const selected: ConsoleRole | null =
    selectedId === 'new' ? null : roles.find(r => r.id === selectedId) || null

  const pick = useConstCallback((r: ConsoleRole) => {
    setSelectedId(r.id)
    setSlug(r.slug)
    setName(r.name)
    setDescription(r.description)
    setChecked(new Set(r.permissions))
    setNotice(null)
  })

  const startNew = useConstCallback(() => {
    setSelectedId('new')
    setSlug('')
    setName('')
    setDescription('')
    setChecked(new Set())
    setNotice(null)
  })

  const togglePerm = useConstCallback((key: string) => {
    setChecked(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
    setNotice(null)
  })

  const onSave = useConstCallback(async () => {
    if (saving) return
    if (!/^[a-z0-9_]+$/.test(slug.trim())) {
      setNotice({ kind: 'error', text: t('console.roles.bad_slug') })
      return
    }
    if (!name.trim()) {
      setNotice({ kind: 'error', text: t('console.roles.bad_name') })
      return
    }
    setSaving(true)
    try {
      await saveRole({
        id: selectedId === 'new' ? undefined : (selectedId as number),
        slug: slug.trim(),
        name: name.trim(),
        description: description.trim(),
        permissions: [...checked],
      })
      setNotice({ kind: 'ok', text: t('console.roles.saved') })
      await reload()
      setSelectedId(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.roles.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDelete = useConstCallback(async () => {
    if (selectedId === 'new' || selectedId == null || saving) return
    if (!window.confirm(t('console.roles.delete_confirm'))) return
    setSaving(true)
    try {
      await deleteRole(selectedId as number)
      setNotice({ kind: 'ok', text: t('console.roles.deleted') })
      await reload()
      setSelectedId(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.roles.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  const levelRoles = roles.filter(r => r.is_level_role).sort((a, b) => a.level - b.level)
  const otherRoles = roles.filter(r => !r.is_level_role)

  const grouped = GROUP_ORDER.map(g => ({
    group: g,
    items: permissions.filter(p => (GROUP_OF[p.group] || 'other') === g),
  })).filter(g => g.items.length > 0)

  const isLevelRole = !!selected?.is_level_role

  return (
    <>
      <Styled.H1>{t('console.roles.title')}</Styled.H1>
      <Styled.Desc>{t('console.roles.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      {loading ? (
        <Styled.Loading>{t('console.roles.loading')}</Styled.Loading>
      ) : (
        <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
          <div style={{ minWidth: 240, flexShrink: 0 }}>
            <Styled.Button onClick={startNew} style={{ marginBottom: 12 }}>
              {t('console.roles.new_role')}
            </Styled.Button>
            <Styled.Label>{t('console.roles.level_roles')}</Styled.Label>
            <Styled.Table>
              <tbody>
                {levelRoles.map(r => (
                  <tr
                    key={r.id}
                    onClick={() => pick(r)}
                    style={{
                      cursor: 'pointer',
                      background: selectedId === r.id ? '#f0f6ff' : undefined,
                    }}
                  >
                    <td>
                      <strong>Lv{r.level}</strong>
                      <span style={{ marginLeft: 8, color: '#888' }}>{r.name}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Styled.Table>
            <Styled.Label style={{ marginTop: 12 }}>{t('console.roles.other_roles')}</Styled.Label>
            <Styled.Table>
              <tbody>
                {otherRoles.map(r => (
                  <tr
                    key={r.id}
                    onClick={() => pick(r)}
                    style={{
                      cursor: 'pointer',
                      background: selectedId === r.id ? '#f0f6ff' : undefined,
                    }}
                  >
                    <td>
                      <strong>{r.slug}</strong>
                      <span style={{ marginLeft: 8, color: '#888' }}>{r.name}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Styled.Table>
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            {selectedId == null ? (
              <Styled.Desc>{t('console.roles.pick_hint')}</Styled.Desc>
            ) : (
              <>
                <Styled.H1 style={{ fontSize: 18 }}>
                  {selectedId === 'new'
                    ? t('console.roles.creating')
                    : t('console.roles.editing', { name: selected?.name || '' })}
                </Styled.H1>

                <Styled.Field>
                  <Styled.Label>{t('console.roles.slug_label')}</Styled.Label>
                  <Styled.Input
                    value={slug}
                    disabled={isLevelRole}
                    placeholder="my_role"
                    onChange={e => setSlug(e.target.value)}
                  />
                  {isLevelRole && <Styled.Desc>{t('console.roles.level_slug_locked')}</Styled.Desc>}
                </Styled.Field>

                <Styled.Field>
                  <Styled.Label>{t('console.roles.name_label')}</Styled.Label>
                  <Styled.Input value={name} onChange={e => setName(e.target.value)} />
                </Styled.Field>

                <Styled.Field>
                  <Styled.Label>{t('console.roles.desc_label')}</Styled.Label>
                  <Styled.Input value={description} onChange={e => setDescription(e.target.value)} />
                </Styled.Field>

                {grouped.map(g => (
                  <div key={g.group} style={{ marginBottom: 16 }}>
                    <Styled.Label>{groupLabel(g.group)}</Styled.Label>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 16px' }}>
                      {g.items.map(p => (
                        <label
                          key={p.key}
                          style={{ fontSize: 14, display: 'flex', gap: 6, alignItems: 'center', cursor: 'pointer' }}
                          title={p.key}
                        >
                          <input
                            type="checkbox"
                            checked={checked.has(p.key)}
                            onChange={() => togglePerm(p.key)}
                          />
                          {p.description}
                        </label>
                      ))}
                    </div>
                  </div>
                ))}

                <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                  <Styled.Button onClick={onSave} disabled={saving}>
                    {saving ? t('console.roles.saving') : t('console.roles.save')}
                  </Styled.Button>
                  {selectedId !== 'new' && (
                    <Styled.Button
                      onClick={onDelete}
                      disabled={saving || isLevelRole}
                      title={isLevelRole ? t('console.roles.level_delete_locked') : undefined}
                      style={{ background: isLevelRole ? '#ccc' : '#a33' }}
                    >
                      {t('console.roles.delete')}
                    </Styled.Button>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </>
  )
}

export default ConsoleRoles

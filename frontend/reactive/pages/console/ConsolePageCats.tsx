import * as React from 'react'
import { useEffect, useState } from 'react'
import { deletePageCat, listPageCats, PageCategory, savePageCat } from '~api/console-pagecats'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

const ConsolePageCats: React.FC = () => {
  const [cats, setCats] = useState<PageCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedId, setSelectedId] = useState<number | 'new' | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const reload = useConstCallback(async () => {
    setLoading(true)
    try {
      const r = await listPageCats()
      setCats(r.categories)
    } catch {
      setNotice({ kind: 'error', text: t('console.pagecats.load_failed') })
    } finally {
      setLoading(false)
    }
  })

  useEffect(() => {
    reload()
  }, [reload])

  const selected: PageCategory | null =
    selectedId === 'new' ? null : cats.find(c => c.id === selectedId) || null

  const pick = useConstCallback((c: PageCategory) => {
    setSelectedId(c.id)
    setName(c.name)
    setDescription(c.description)
    setNotice(null)
  })

  const startNew = useConstCallback(() => {
    setSelectedId('new')
    setName('')
    setDescription('')
    setNotice(null)
  })

  const onSave = useConstCallback(async () => {
    if (saving) return
    if (!name.trim()) {
      setNotice({ kind: 'error', text: t('console.pagecats.bad_name') })
      return
    }
    setSaving(true)
    try {
      await savePageCat({
        id: selectedId === 'new' ? undefined : (selectedId as number),
        name: name.trim(),
        description: description.trim(),
      })
      setNotice({ kind: 'ok', text: t('console.pagecats.saved') })
      await reload()
      setSelectedId(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.pagecats.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDelete = useConstCallback(async () => {
    if (selectedId === 'new' || selectedId == null || saving) return
    if (!window.confirm(t('console.pagecats.delete_confirm'))) return
    setSaving(true)
    try {
      await deletePageCat(selectedId as number)
      setNotice({ kind: 'ok', text: t('console.pagecats.deleted') })
      await reload()
      setSelectedId(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.pagecats.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  return (
    <>
      <Styled.H1>{t('console.pagecats.title')}</Styled.H1>
      <Styled.Desc>{t('console.pagecats.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      {loading ? (
        <Styled.Loading>{t('console.pagecats.loading')}</Styled.Loading>
      ) : (
        <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
          <div style={{ minWidth: 240, flexShrink: 0 }}>
            <Styled.Button onClick={startNew} style={{ marginBottom: 12 }}>
              {t('console.pagecats.new_cat')}
            </Styled.Button>
            <Styled.Table>
              <tbody>
                {cats.map(c => (
                  <tr
                    key={c.id}
                    onClick={() => pick(c)}
                    style={{
                      cursor: 'pointer',
                      background: selectedId === c.id ? '#f0f6ff' : undefined,
                    }}
                  >
                    <td>
                      <strong>{c.name}</strong>
                      {c.description && (
                        <span style={{ marginLeft: 8, color: '#888' }}>{c.description}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </Styled.Table>
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            {selectedId == null ? (
              <Styled.Desc>{t('console.pagecats.pick_hint')}</Styled.Desc>
            ) : (
              <>
                <Styled.H1 style={{ fontSize: 18 }}>
                  {selectedId === 'new'
                    ? t('console.pagecats.creating')
                    : t('console.pagecats.editing', { name: selected?.name || '' })}
                </Styled.H1>

                <Styled.Field>
                  <Styled.Label>{t('console.pagecats.name_label')}</Styled.Label>
                  <Styled.Input value={name} onChange={e => setName(e.target.value)} />
                </Styled.Field>

                <Styled.Field>
                  <Styled.Label>{t('console.pagecats.desc_label')}</Styled.Label>
                  <Styled.Input
                    value={description}
                    onChange={e => setDescription(e.target.value)}
                    placeholder={t('console.pagecats.desc_placeholder')}
                  />
                </Styled.Field>

                <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                  <Styled.Button onClick={onSave} disabled={saving}>
                    {saving ? t('console.pagecats.saving') : t('console.pagecats.save')}
                  </Styled.Button>
                  {selectedId !== 'new' && (
                    <Styled.Button
                      onClick={onDelete}
                      disabled={saving}
                      style={{ background: '#a33' }}
                    >
                      {t('console.pagecats.delete')}
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

export default ConsolePageCats

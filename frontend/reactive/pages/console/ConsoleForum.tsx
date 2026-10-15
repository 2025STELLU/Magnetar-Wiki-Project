import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleForumCategory,
  ConsoleForumSection,
  deleteForumCategory,
  deleteForumSection,
  listForumSections,
  saveForumCategory,
  saveForumSection,
} from '~api/console-forum'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

type SectionSel = number | 'new' | null
type CatSel = number | 'new' | null

const selectStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  fontSize: 14,
  border: '1px solid #ccc',
  borderRadius: 4,
  background: '#fff',
}

const ConsoleForum: React.FC = () => {
  const [sections, setSections] = useState<ConsoleForumSection[]>([])
  const [loading, setLoading] = useState(true)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  // 版块编辑器状态
  const [sectionSel, setSectionSel] = useState<SectionSel>(null)
  const [secName, setSecName] = useState('')
  const [secDesc, setSecDesc] = useState('')
  // 分类编辑器状态
  const [catSel, setCatSel] = useState<CatSel>(null)
  const [catName, setCatName] = useState('')
  const [catDesc, setCatDesc] = useState('')
  const [catSection, setCatSection] = useState('')
  const [saving, setSaving] = useState(false)

  const reload = useConstCallback(async () => {
    setLoading(true)
    try {
      const r = await listForumSections()
      setSections(r.sections)
    } catch {
      setNotice({ kind: 'error', text: t('console.forum.load_failed') })
    } finally {
      setLoading(false)
    }
  })

  useEffect(() => {
    reload()
  }, [reload])

  const findCategory = (id: number): { section: ConsoleForumSection; cat: ConsoleForumCategory } | null => {
    for (const s of sections) {
      const cat = s.categories.find(c => c.id === id)
      if (cat) return { section: s, cat }
    }
    return null
  }

  const selectedSection: ConsoleForumSection | null =
    sectionSel === 'new' ? null : sections.find(x => x.id === sectionSel) || null

  const pickSection = useConstCallback((x: ConsoleForumSection) => {
    setSectionSel(x.id)
    setSecName(x.name)
    setSecDesc(x.description)
    setCatSel(null)
    setNotice(null)
  })

  const startNewSection = useConstCallback(() => {
    setSectionSel('new')
    setSecName('')
    setSecDesc('')
    setCatSel(null)
    setNotice(null)
  })

  const onSaveSection = useConstCallback(async () => {
    if (saving) return
    if (!secName.trim()) {
      setNotice({ kind: 'error', text: t('console.forum.bad_name') })
      return
    }
    setSaving(true)
    try {
      await saveForumSection({
        id: sectionSel === 'new' ? undefined : (sectionSel as number),
        name: secName.trim(),
        description: secDesc.trim(),
      })
      setNotice({ kind: 'ok', text: t('console.forum.saved') })
      await reload()
      setSectionSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.forum.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDeleteSection = useConstCallback(async () => {
    if (sectionSel === 'new' || sectionSel == null || saving) return
    if (!window.confirm(t('console.forum.delete_section_confirm'))) return
    setSaving(true)
    try {
      await deleteForumSection(sectionSel as number)
      setNotice({ kind: 'ok', text: t('console.forum.deleted') })
      await reload()
      setSectionSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.forum.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  const pickCategory = useConstCallback((s: ConsoleForumSection, c: ConsoleForumCategory) => {
    setCatSel(c.id)
    setCatName(c.name)
    setCatDesc(c.description)
    setCatSection(String(s.id))
    setSectionSel(null)
    setNotice(null)
  })

  const startNewCategory = useConstCallback((sectionId?: number) => {
    setCatSel('new')
    setCatName('')
    setCatDesc('')
    setCatSection(sectionId != null ? String(sectionId) : sections.length > 0 ? String(sections[0].id) : '')
    setSectionSel(null)
    setNotice(null)
  })

  const onSaveCategory = useConstCallback(async () => {
    if (saving) return
    if (!catName.trim()) {
      setNotice({ kind: 'error', text: t('console.forum.bad_name') })
      return
    }
    if (!catSection) {
      setNotice({ kind: 'error', text: t('console.forum.bad_section') })
      return
    }
    setSaving(true)
    try {
      await saveForumCategory({
        id: catSel === 'new' ? undefined : (catSel as number),
        section_id: Number(catSection),
        name: catName.trim(),
        description: catDesc.trim(),
      })
      setNotice({ kind: 'ok', text: t('console.forum.saved') })
      await reload()
      setCatSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.forum.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDeleteCategory = useConstCallback(async () => {
    if (catSel === 'new' || catSel == null || saving) return
    if (!window.confirm(t('console.forum.delete_category_confirm'))) return
    setSaving(true)
    try {
      await deleteForumCategory(catSel as number)
      setNotice({ kind: 'ok', text: t('console.forum.deleted') })
      await reload()
      setCatSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.forum.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  const editingCategory = catSel !== null ? findCategory(catSel === 'new' ? -1 : (catSel as number)) : null

  return (
    <>
      <Styled.H1>{t('console.forum.title')}</Styled.H1>
      <Styled.Desc>{t('console.forum.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      {loading ? (
        <Styled.Loading>{t('console.forum.loading')}</Styled.Loading>
      ) : (
        <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
          <div style={{ minWidth: 300, flexShrink: 0 }}>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              <Styled.Button onClick={startNewSection}>{t('console.forum.new_section')}</Styled.Button>
              <Styled.Button onClick={() => startNewCategory()} disabled={sections.length === 0}>
                {t('console.forum.new_category')}
              </Styled.Button>
            </div>
            {sections.map(s => (
              <div key={s.id} style={{ marginBottom: 12 }}>
                <Styled.Table>
                  <tbody>
                    <tr
                      onClick={() => pickSection(s)}
                      style={{
                        cursor: 'pointer',
                        background: sectionSel === s.id ? '#f0f6ff' : undefined,
                      }}
                    >
                      <td>
                        <strong>{s.name}</strong>
                        <span style={{ marginLeft: 8, color: '#888', fontSize: 12 }}>
                          {t('console.forum.categories_count', { n: String(s.categories.length) })}
                        </span>
                      </td>
                    </tr>
                    {s.categories.map(c => (
                      <tr
                        key={c.id}
                        onClick={() => pickCategory(s, c)}
                        style={{
                          cursor: 'pointer',
                          background: catSel === c.id ? '#f0f6ff' : undefined,
                        }}
                      >
                        <td style={{ paddingLeft: 24 }}>
                          <span style={{ color: '#888', marginRight: 6 }}>↳</span>
                          {c.name}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Styled.Table>
              </div>
            ))}
            {sections.length === 0 && <Styled.Desc>{t('console.forum.no_sections')}</Styled.Desc>}
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            {sectionSel == null && catSel == null ? (
              <Styled.Desc>{t('console.forum.pick_hint')}</Styled.Desc>
            ) : sectionSel != null ? (
              <>
                <Styled.H1 style={{ fontSize: 18 }}>
                  {sectionSel === 'new'
                    ? t('console.forum.creating_section')
                    : t('console.forum.editing_section', { name: selectedSection?.name || '' })}
                </Styled.H1>
                <Styled.Field>
                  <Styled.Label>{t('console.forum.name_label')}</Styled.Label>
                  <Styled.Input value={secName} onChange={e => setSecName(e.target.value)} />
                </Styled.Field>
                <Styled.Field>
                  <Styled.Label>{t('console.forum.description_label')}</Styled.Label>
                  <Styled.Input value={secDesc} onChange={e => setSecDesc(e.target.value)} />
                </Styled.Field>
                <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                  <Styled.Button onClick={onSaveSection} disabled={saving}>
                    {saving ? t('console.forum.saving') : t('console.forum.save')}
                  </Styled.Button>
                  {sectionSel !== 'new' && (
                    <Styled.Button onClick={onDeleteSection} disabled={saving} style={{ background: '#a33' }}>
                      {t('console.forum.delete')}
                    </Styled.Button>
                  )}
                </div>
              </>
            ) : (
              <>
                <Styled.H1 style={{ fontSize: 18 }}>
                  {catSel === 'new'
                    ? t('console.forum.creating_category')
                    : t('console.forum.editing_category', { name: editingCategory?.cat.name || '' })}
                </Styled.H1>
                <Styled.Field>
                  <Styled.Label>{t('console.forum.section_label')}</Styled.Label>
                  <select
                    value={catSection}
                    onChange={e => setCatSection(e.target.value)}
                    style={selectStyle}
                  >
                    {sections.map(s => (
                      <option key={s.id} value={s.id}>
                        {s.name}
                      </option>
                    ))}
                  </select>
                </Styled.Field>
                <Styled.Field>
                  <Styled.Label>{t('console.forum.name_label')}</Styled.Label>
                  <Styled.Input value={catName} onChange={e => setCatName(e.target.value)} />
                </Styled.Field>
                <Styled.Field>
                  <Styled.Label>{t('console.forum.description_label')}</Styled.Label>
                  <Styled.Input value={catDesc} onChange={e => setCatDesc(e.target.value)} />
                </Styled.Field>
                <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                  <Styled.Button onClick={onSaveCategory} disabled={saving}>
                    {saving ? t('console.forum.saving') : t('console.forum.save')}
                  </Styled.Button>
                  {catSel !== 'new' && (
                    <Styled.Button onClick={onDeleteCategory} disabled={saving} style={{ background: '#a33' }}>
                      {t('console.forum.delete')}
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

export default ConsoleForum

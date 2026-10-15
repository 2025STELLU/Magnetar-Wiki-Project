import * as React from 'react'
import { useEffect, useState } from 'react'
import {
  ConsoleTag,
  ConsoleTagCategory,
  deleteTag,
  deleteTagCategory,
  listTags,
  saveTag,
  saveTagCategory,
} from '~api/console-tags'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import * as Styled from './Console.styles'

type TagSel = number | 'new' | null
type CatSel = number | 'new' | null

const selectStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  fontSize: 14,
  border: '1px solid #ccc',
  borderRadius: 4,
  background: '#fff',
}

const ConsoleTags: React.FC = () => {
  const [tags, setTags] = useState<ConsoleTag[]>([])
  const [categories, setCategories] = useState<ConsoleTagCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  // 标签编辑器状态
  const [tagSel, setTagSel] = useState<TagSel>(null)
  const [tagName, setTagName] = useState('')
  const [tagCategory, setTagCategory] = useState('')
  const [tagIndexed, setTagIndexed] = useState(true)
  // 分类编辑器状态
  const [catSel, setCatSel] = useState<CatSel>(null)
  const [catName, setCatName] = useState('')
  const [catSlug, setCatSlug] = useState('')
  const [catDesc, setCatDesc] = useState('')
  const [catPriority, setCatPriority] = useState('')
  const [saving, setSaving] = useState(false)

  const reload = useConstCallback(async () => {
    setLoading(true)
    try {
      const r = await listTags()
      setTags(r.tags)
      setCategories(r.categories)
    } catch {
      setNotice({ kind: 'error', text: t('console.tags.load_failed') })
    } finally {
      setLoading(false)
    }
  })

  useEffect(() => {
    reload()
  }, [reload])

  const selectedTag: ConsoleTag | null =
    tagSel === 'new' ? null : tags.find(x => x.id === tagSel) || null

  const pickTag = useConstCallback((x: ConsoleTag) => {
    setTagSel(x.id)
    setTagName(x.name)
    setTagCategory(x.category_id ? String(x.category_id) : '')
    setTagIndexed(x.is_indexed)
    setNotice(null)
  })

  const startNewTag = useConstCallback(() => {
    setTagSel('new')
    setTagName('')
    setTagCategory('')
    setTagIndexed(true)
    setNotice(null)
  })

  const onSaveTag = useConstCallback(async () => {
    if (saving) return
    if (!tagName.trim()) {
      setNotice({ kind: 'error', text: t('console.tags.bad_name') })
      return
    }
    setSaving(true)
    try {
      await saveTag({
        id: tagSel === 'new' ? undefined : (tagSel as number),
        name: tagName.trim(),
        category_id: tagCategory ? Number(tagCategory) : null,
        is_indexed: tagIndexed,
      })
      setNotice({ kind: 'ok', text: t('console.tags.saved') })
      await reload()
      setTagSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.tags.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDeleteTag = useConstCallback(async () => {
    if (tagSel === 'new' || tagSel == null || saving) return
    if (!window.confirm(t('console.tags.delete_confirm'))) return
    setSaving(true)
    try {
      await deleteTag(tagSel as number)
      setNotice({ kind: 'ok', text: t('console.tags.deleted') })
      await reload()
      setTagSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.tags.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  const selectedCat: ConsoleTagCategory | null =
    catSel === 'new' ? null : categories.find(x => x.id === catSel) || null

  const pickCat = useConstCallback((x: ConsoleTagCategory) => {
    setCatSel(x.id)
    setCatName(x.name)
    setCatSlug(x.slug)
    setCatDesc(x.description)
    setCatPriority(x.priority != null ? String(x.priority) : '')
    setNotice(null)
  })

  const startNewCat = useConstCallback(() => {
    setCatSel('new')
    setCatName('')
    setCatSlug('')
    setCatDesc('')
    setCatPriority('')
    setNotice(null)
  })

  const onSaveCat = useConstCallback(async () => {
    if (saving) return
    if (!catName.trim()) {
      setNotice({ kind: 'error', text: t('console.tags.bad_name') })
      return
    }
    if (!/^[A-Za-z0-9_-]+$/.test(catSlug.trim())) {
      setNotice({ kind: 'error', text: t('console.tags.bad_slug') })
      return
    }
    setSaving(true)
    try {
      const prio = catPriority.trim()
      await saveTagCategory({
        id: catSel === 'new' ? undefined : (catSel as number),
        name: catName.trim(),
        slug: catSlug.trim(),
        description: catDesc.trim(),
        priority: prio === '' ? null : Number(prio),
      })
      setNotice({ kind: 'ok', text: t('console.tags.saved') })
      await reload()
      setCatSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.tags.save_failed') })
    } finally {
      setSaving(false)
    }
  })

  const onDeleteCat = useConstCallback(async () => {
    if (catSel === 'new' || catSel == null || saving) return
    if (!window.confirm(t('console.tags.delete_category_confirm'))) return
    setSaving(true)
    try {
      await deleteTagCategory(catSel as number)
      setNotice({ kind: 'ok', text: t('console.tags.deleted') })
      await reload()
      setCatSel(null)
    } catch {
      setNotice({ kind: 'error', text: t('console.tags.delete_failed') })
    } finally {
      setSaving(false)
    }
  })

  // 标签按分类分组展示
  const q = query.trim().toLowerCase()
  const visibleTags = q ? tags.filter(x => x.name.toLowerCase().includes(q)) : tags
  const groups: { key: string; label: string; items: ConsoleTag[] }[] = []
  const uncategorized = visibleTags.filter(x => !x.category_id)
  if (uncategorized.length > 0) {
    groups.push({ key: '', label: t('console.tags.uncategorized'), items: uncategorized })
  }
  for (const c of categories) {
    const items = visibleTags.filter(x => x.category_id === c.id)
    if (items.length > 0) {
      groups.push({ key: String(c.id), label: c.name, items })
    }
  }

  return (
    <>
      <Styled.H1>{t('console.tags.title')}</Styled.H1>
      <Styled.Desc>{t('console.tags.desc')}</Styled.Desc>
      {notice && <Styled.Notice $kind={notice.kind === 'ok' ? 'ok' : undefined}>{notice.text}</Styled.Notice>}

      {loading ? (
        <Styled.Loading>{t('console.tags.loading')}</Styled.Loading>
      ) : (
        <>
          <Styled.H1 style={{ fontSize: 18, marginTop: 8 }}>{t('console.tags.tags_title')}</Styled.H1>
          <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
            <div style={{ minWidth: 260, flexShrink: 0 }}>
              <Styled.Button onClick={startNewTag} style={{ marginBottom: 12 }}>
                {t('console.tags.new_tag')}
              </Styled.Button>
              <Styled.Input
                value={query}
                placeholder={t('console.tags.search_placeholder')}
                onChange={e => setQuery(e.target.value)}
                style={{ marginBottom: 12 }}
              />
              {groups.map(g => (
                <div key={g.key} style={{ marginBottom: 12 }}>
                  <Styled.Label>{g.label}</Styled.Label>
                  <Styled.Table>
                    <tbody>
                      {g.items.map(x => (
                        <tr
                          key={x.id}
                          onClick={() => pickTag(x)}
                          style={{
                            cursor: 'pointer',
                            background: tagSel === x.id ? '#f0f6ff' : undefined,
                          }}
                        >
                          <td>
                            <strong>{x.name}</strong>
                            <span style={{ marginLeft: 8, color: '#888', fontSize: 12 }}>
                              {t('console.tags.articles_count', { n: String(x.articles) })}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </Styled.Table>
                </div>
              ))}
            </div>

            <div style={{ flex: 1, minWidth: 0 }}>
              {tagSel == null ? (
                <Styled.Desc>{t('console.tags.pick_hint')}</Styled.Desc>
              ) : (
                <>
                  <Styled.H1 style={{ fontSize: 18 }}>
                    {tagSel === 'new'
                      ? t('console.tags.creating_tag')
                      : t('console.tags.editing_tag', { name: selectedTag?.name || '' })}
                  </Styled.H1>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.name_label')}</Styled.Label>
                    <Styled.Input value={tagName} onChange={e => setTagName(e.target.value)} />
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.category_label')}</Styled.Label>
                    <select
                      value={tagCategory}
                      onChange={e => setTagCategory(e.target.value)}
                      style={selectStyle}
                    >
                      <option value="">{t('console.tags.no_category')}</option>
                      {categories.map(c => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </select>
                  </Styled.Field>
                  <Styled.Field>
                    <label style={{ fontSize: 14, display: 'flex', gap: 6, alignItems: 'center', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={tagIndexed}
                        onChange={e => setTagIndexed(e.target.checked)}
                      />
                      {t('console.tags.indexed_label')}
                    </label>
                  </Styled.Field>
                  <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                    <Styled.Button onClick={onSaveTag} disabled={saving}>
                      {saving ? t('console.tags.saving') : t('console.tags.save')}
                    </Styled.Button>
                    {tagSel !== 'new' && (
                      <Styled.Button onClick={onDeleteTag} disabled={saving} style={{ background: '#a33' }}>
                        {t('console.tags.delete')}
                      </Styled.Button>
                    )}
                  </div>
                </>
              )}
            </div>
          </div>

          <Styled.H1 style={{ fontSize: 18, marginTop: 32 }}>{t('console.tags.categories_title')}</Styled.H1>
          <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
            <div style={{ minWidth: 260, flexShrink: 0 }}>
              <Styled.Button onClick={startNewCat} style={{ marginBottom: 12 }}>
                {t('console.tags.new_category')}
              </Styled.Button>
              <Styled.Table>
                <tbody>
                  {categories.map(c => (
                    <tr
                      key={c.id}
                      onClick={() => pickCat(c)}
                      style={{
                        cursor: 'pointer',
                        background: catSel === c.id ? '#f0f6ff' : undefined,
                      }}
                    >
                      <td>
                        <strong>{c.name}</strong>
                        <span style={{ marginLeft: 8, color: '#888', fontSize: 12 }}>{c.slug}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Styled.Table>
            </div>

            <div style={{ flex: 1, minWidth: 0 }}>
              {catSel == null ? (
                <Styled.Desc>{t('console.tags.pick_category_hint')}</Styled.Desc>
              ) : (
                <>
                  <Styled.H1 style={{ fontSize: 18 }}>
                    {catSel === 'new'
                      ? t('console.tags.creating_category')
                      : t('console.tags.editing_category', { name: selectedCat?.name || '' })}
                  </Styled.H1>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.category_name_label')}</Styled.Label>
                    <Styled.Input value={catName} onChange={e => setCatName(e.target.value)} />
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.slug_label')}</Styled.Label>
                    <Styled.Input
                      value={catSlug}
                      placeholder="my-category"
                      onChange={e => setCatSlug(e.target.value)}
                    />
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.description_label')}</Styled.Label>
                    <Styled.Input value={catDesc} onChange={e => setCatDesc(e.target.value)} />
                  </Styled.Field>
                  <Styled.Field>
                    <Styled.Label>{t('console.tags.priority_label')}</Styled.Label>
                    <Styled.Input
                      value={catPriority}
                      placeholder="0"
                      onChange={e => setCatPriority(e.target.value)}
                    />
                  </Styled.Field>
                  <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                    <Styled.Button onClick={onSaveCat} disabled={saving}>
                      {saving ? t('console.tags.saving') : t('console.tags.save')}
                    </Styled.Button>
                    {catSel !== 'new' && (
                      <Styled.Button onClick={onDeleteCat} disabled={saving} style={{ background: '#a33' }}>
                        {t('console.tags.delete')}
                      </Styled.Button>
                    )}
                  </div>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </>
  )
}

export default ConsoleTags

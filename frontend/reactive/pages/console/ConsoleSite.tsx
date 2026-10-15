import * as React from 'react'
import { useEffect, useState } from 'react'
import { t } from '~util/i18n'
import { getSiteSettings, saveSiteSettings, SiteSettingsData, SiteSettingsResponse } from '~api/console-site'
import useConstCallback from '~util/const-callback'
import * as Styled from './Console.styles'

const emptyForm: SiteSettingsData = {
  slug: '',
  title: '',
  headline: '',
  domain: '',
  media_domain: '',
  home_page: '',
  footer_license: '',
  signup_notice: '',
  password_help: '',
  email_policy: '',
  language: '',
  time_zone: '',
  theme_id: null,
  system_theme_id: null,
  rating_mode: '',
  create_tags: '',
}

const ConsoleSite: React.FC = () => {
  const [form, setForm] = useState<SiteSettingsData>(emptyForm)
  const [meta, setMeta] = useState<SiteSettingsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    let live = true
    getSiteSettings()
      .then(resp => {
        if (live) {
          setForm({
            slug: resp.site.slug,
            title: resp.site.title,
            headline: resp.site.headline,
            domain: resp.site.domain,
            media_domain: resp.site.media_domain,
            home_page: resp.site.home_page,
            footer_license: resp.site.footer_license,
            signup_notice: resp.site.signup_notice,
            password_help: resp.site.password_help,
            email_policy: resp.site.email_policy,
            language: resp.site.language,
            time_zone: resp.site.time_zone,
            theme_id: resp.site.theme_id,
            system_theme_id: resp.site.system_theme_id,
            rating_mode: resp.site.rating_mode,
            create_tags: resp.site.create_tags,
          })
          setMeta(resp)
        }
      })
      .catch(() => {
        if (live) setError(t('console.site.load_failed'))
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [])

  const set = useConstCallback((key: keyof SiteSettingsData, value: string | number | null) => {
    setForm(prev => ({ ...prev, [key]: value }))
    setNotice(null)
  })

  const onSave = useConstCallback(async () => {
    if (saving) return
    setSaving(true)
    setError(null)
    setNotice(null)
    try {
      const resp = await saveSiteSettings(form)
      if (resp.warning) {
        setNotice(resp.warning)
      } else {
        setNotice(t('console.site.saved'))
      }
    } catch (e) {
      setError(t('console.site.save_failed'))
    } finally {
      setSaving(false)
    }
  })

  if (loading) return <Styled.Loading>{t('console.site.loading')}</Styled.Loading>

  const themeId = (v: string) => (v === '' ? null : Number(v))

  return (
    <>
      <Styled.H1>{t('console.site.title')}</Styled.H1>
      <Styled.Desc>{t('console.site.desc')}</Styled.Desc>
      {error && <Styled.Notice>{error}</Styled.Notice>}
      {notice && <Styled.Notice $kind="ok">{notice}</Styled.Notice>}

      <Styled.H1 style={{ fontSize: '1.1rem', marginTop: '1.5rem' }}>{t('console.site.group_basic')}</Styled.H1>
      <Styled.Field>
        <Styled.Label>{t('console.site.slug_label')}</Styled.Label>
        <Styled.Input value={form.slug} onChange={e => set('slug', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.title_label')}</Styled.Label>
        <Styled.Input value={form.title} onChange={e => set('title', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.headline_label')}</Styled.Label>
        <Styled.Input value={form.headline} onChange={e => set('headline', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.domain_label')}</Styled.Label>
        <Styled.Input value={form.domain} onChange={e => set('domain', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.media_domain_label')}</Styled.Label>
        <Styled.Input value={form.media_domain} onChange={e => set('media_domain', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.home_page_label')}</Styled.Label>
        <Styled.Input value={form.home_page} onChange={e => set('home_page', e.target.value)} />
      </Styled.Field>

      <Styled.H1 style={{ fontSize: '1.1rem', marginTop: '1.5rem' }}>{t('console.site.group_appearance')}</Styled.H1>
      <Styled.Field>
        <Styled.Label>{t('console.site.theme_label')}</Styled.Label>
        <Styled.Input
          as="select"
          value={form.theme_id === null ? '' : String(form.theme_id)}
          onChange={e => set('theme_id', themeId(e.target.value))}
        >
          <option value="">{t('console.site.no_theme')}</option>
          {(meta?.themes ?? []).map(th => (
            <option key={th.id} value={th.id}>{th.name}</option>
          ))}
        </Styled.Input>
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.system_theme_label')}</Styled.Label>
        <Styled.Input
          as="select"
          value={form.system_theme_id === null ? '' : String(form.system_theme_id)}
          onChange={e => set('system_theme_id', themeId(e.target.value))}
        >
          <option value="">{t('console.site.no_theme')}</option>
          {(meta?.themes ?? []).map(th => (
            <option key={th.id} value={th.id}>{th.name}</option>
          ))}
        </Styled.Input>
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.footer_license_label')}</Styled.Label>
        <Styled.Input value={form.footer_license} onChange={e => set('footer_license', e.target.value)} />
      </Styled.Field>

      <Styled.H1 style={{ fontSize: '1.1rem', marginTop: '1.5rem' }}>{t('console.site.group_content')}</Styled.H1>
      <Styled.Field>
        <Styled.Label>{t('console.site.signup_notice_label')}</Styled.Label>
        <Styled.Input as="textarea" rows={3} value={form.signup_notice} onChange={e => set('signup_notice', e.target.value)} />
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.password_help_label')}</Styled.Label>
        <Styled.Input as="textarea" rows={3} value={form.password_help} onChange={e => set('password_help', e.target.value)} />
      </Styled.Field>

      <Styled.H1 style={{ fontSize: '1.1rem', marginTop: '1.5rem' }}>{t('console.site.group_policy')}</Styled.H1>
      <Styled.Field>
        <Styled.Label>{t('console.site.email_policy_label')}</Styled.Label>
        <Styled.Input as="select" value={form.email_policy} onChange={e => set('email_policy', e.target.value)}>
          {(meta?.email_policies ?? []).map(p => (
            <option key={p} value={p}>{p}</option>
          ))}
        </Styled.Input>
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.language_label')}</Styled.Label>
        <Styled.Input as="select" value={form.language} onChange={e => set('language', e.target.value)}>
          {(meta?.languages ?? []).map(l => (
            <option key={l.tag} value={l.tag}>{l.name}</option>
          ))}
        </Styled.Input>
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.time_zone_label')}</Styled.Label>
        <Styled.Input value={form.time_zone} onChange={e => set('time_zone', e.target.value)} />
      </Styled.Field>

      <Styled.H1 style={{ fontSize: '1.1rem', marginTop: '1.5rem' }}>{t('console.site.group_features')}</Styled.H1>
      <Styled.Field>
        <Styled.Label>{t('console.site.rating_mode_label')}</Styled.Label>
        <Styled.Input as="select" value={form.rating_mode} onChange={e => set('rating_mode', e.target.value)}>
          {(meta?.rating_modes ?? []).map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </Styled.Input>
      </Styled.Field>
      <Styled.Field>
        <Styled.Label>{t('console.site.create_tags_label')}</Styled.Label>
        <Styled.Input as="select" value={form.create_tags} onChange={e => set('create_tags', e.target.value)}>
          {(meta?.tag_modes ?? []).map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </Styled.Input>
      </Styled.Field>

      <Styled.Button onClick={onSave} disabled={saving} style={{ marginTop: '1rem' }}>
        {saving ? t('console.site.saving') : t('console.site.save')}
      </Styled.Button>
    </>
  )
}

export default ConsoleSite

import * as React from 'react'
import { useEffect, useState } from 'react'
import { t } from '~util/i18n'
import { getLevelConfig, saveLevelConfig } from '~api/console'
import useConstCallback from '~util/const-callback'
import * as Styled from './Console.styles'

const ConsoleLevels: React.FC = () => {
  const [thresholds, setThresholds] = useState<number[]>([])
  const [prefix, setPrefix] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    let live = true
    getLevelConfig()
      .then(resp => {
        if (live) {
          setThresholds(resp.thresholds)
          setPrefix(resp.prefix)
        }
      })
      .catch(() => {
        if (live) setError(t('console.levels.load_failed'))
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [])

  const onThreshold = useConstCallback((i: number, raw: string) => {
    const next = thresholds.slice()
    next[i] = Number(raw)
    setThresholds(next)
    setSaved(false)
  })

  const valid = (): string | null => {
    if (!prefix.trim()) return t('console.levels.empty_prefix')
    for (let i = 1; i < thresholds.length; i++) {
      if (!(thresholds[i] > thresholds[i - 1])) return t('console.levels.invalid_order')
    }
    return null
  }

  const onSave = useConstCallback(async () => {
    const problem = valid()
    if (problem) {
      setError(problem)
      return
    }
    setSaving(true)
    setError(null)
    try {
      await saveLevelConfig({ thresholds, prefix: prefix.trim() })
      setSaved(true)
    } catch (e) {
      setError(t('console.levels.save_failed'))
    } finally {
      setSaving(false)
    }
  })

  if (loading) return <Styled.Loading>{t('console.levels.loading')}</Styled.Loading>

  return (
    <>
      <Styled.H1>{t('console.levels.title')}</Styled.H1>
      <Styled.Desc>{t('console.levels.desc')}</Styled.Desc>
      {error && <Styled.Notice>{error}</Styled.Notice>}
      {saved && <Styled.Notice $kind="ok">{t('console.levels.saved')}</Styled.Notice>}
      <Styled.Field>
        <Styled.Label>{t('console.levels.prefix_label')}</Styled.Label>
        <Styled.Input
          value={prefix}
          maxLength={12}
          onChange={e => {
            setPrefix(e.target.value)
            setSaved(false)
          }}
        />
      </Styled.Field>
      <Styled.Table>
        <thead>
          <tr>
            <th>{t('console.levels.level_col')}</th>
            <th>{t('console.levels.xp_col')}</th>
            <th>{t('console.levels.preview_col')}</th>
          </tr>
        </thead>
        <tbody>
          {thresholds.map((xp, i) => (
            <tr key={i}>
              <td>{i + 1}</td>
              <td>
                <input
                  type="number"
                  min={0}
                  value={Number.isFinite(xp) ? xp : 0}
                  onChange={e => onThreshold(i, e.target.value)}
                />
              </td>
              <td>
                {(prefix || t('console.levels.prefix_fallback')) + (i + 1)}
              </td>
            </tr>
          ))}
        </tbody>
      </Styled.Table>
      <Styled.Button onClick={onSave} disabled={saving}>
        {saving ? t('console.levels.saving') : t('console.levels.save')}
      </Styled.Button>
    </>
  )
}

export default ConsoleLevels

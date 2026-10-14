import * as React from 'react'
import { useEffect, useState } from 'react'
import { doCheckin, getCheckinStatus } from '~api/checkin'
import { APIError } from '~util/fetch-util'
import { t } from '~util/i18n'
import useConstCallback from '~util/const-callback'

// 顶栏签到按钮：每天点一次领经验，顺带显示连续签到天数。未登录（401）时不渲染。
const CheckinButton: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [hidden, setHidden] = useState(false)
  const [checkedIn, setCheckedIn] = useState(false)
  const [streak, setStreak] = useState(0)
  const [saving, setSaving] = useState(false)
  const [hint, setHint] = useState<string | null>(null)

  useEffect(() => {
    let live = true
    getCheckinStatus()
      .then(s => {
        if (!live) return
        setCheckedIn(s.checked_in)
        setStreak(s.streak)
      })
      .catch(e => {
        if (live && e instanceof APIError && e.status === 401) setHidden(true)
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [])

  const onCheckin = useConstCallback(async () => {
    if (saving || checkedIn) return
    setSaving(true)
    setHint(null)
    try {
      const r = await doCheckin()
      setCheckedIn(true)
      setStreak(r.streak)
      setHint(t('checkin.success', { xp: r.xp_awarded, streak: r.streak }))
    } catch {
      setHint(t('checkin.failed'))
    } finally {
      setSaving(false)
    }
  })

  if (hidden || loading) return null

  return (
    <span className="checkin-widget" style={{ marginLeft: 8 }}>
      <button type="button" className="btn btn-sm" onClick={onCheckin} disabled={checkedIn || saving}>
        {saving ? t('checkin.signing') : checkedIn ? t('checkin.done') : t('checkin.button')}
      </button>
      {streak > 0 && <span style={{ marginLeft: 6, fontSize: 12 }}>{t('checkin.streak', { days: streak })}</span>}
      {hint && <span style={{ marginLeft: 6, fontSize: 12 }}>{hint}</span>}
    </span>
  )
}

export default CheckinButton

import * as React from 'react'
import { useEffect, useState } from 'react'
import { t } from '~util/i18n'
import { getConsoleOverview, ConsoleOverview } from '~api/console'
import * as Styled from './Console.styles'

const ConsoleDashboard: React.FC = () => {
  const [stats, setStats] = useState<ConsoleOverview | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let live = true
    getConsoleOverview()
      .then(resp => {
        if (live) setStats(resp)
      })
      .catch(() => {
        if (live) setFailed(true)
      })
    return () => {
      live = false
    }
  }, [])

  return (
    <>
      <Styled.H1>{t('console.overview.title')}</Styled.H1>
      <Styled.Desc>{t('console.overview.desc')}</Styled.Desc>
      {failed && <Styled.Notice>{t('console.overview.failed')}</Styled.Notice>}
      {!failed && !stats && <Styled.Loading>{t('console.overview.loading')}</Styled.Loading>}
      {stats && (
        <Styled.CardGrid>
          <Styled.Card>
            <Styled.CardValue>{stats.users}</Styled.CardValue>
            <Styled.CardLabel>{t('console.overview.users')}</Styled.CardLabel>
          </Styled.Card>
          <Styled.Card>
            <Styled.CardValue>{stats.articles}</Styled.CardValue>
            <Styled.CardLabel>{t('console.overview.articles')}</Styled.CardLabel>
          </Styled.Card>
          <Styled.Card>
            <Styled.CardValue>{stats.pending_reports}</Styled.CardValue>
            <Styled.CardLabel>{t('console.overview.pending_reports')}</Styled.CardLabel>
          </Styled.Card>
          <Styled.Card>
            <Styled.CardValue>{stats.pending_tickets}</Styled.CardValue>
            <Styled.CardLabel>{t('console.overview.pending_tickets')}</Styled.CardLabel>
          </Styled.Card>
        </Styled.CardGrid>
      )}
    </>
  )
}

export default ConsoleDashboard

import * as React from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import { t } from '~util/i18n'
import ConsoleDashboard from './ConsoleDashboard'
import ConsoleLevels from './ConsoleLevels'
import * as Styled from './Console.styles'

const Console: React.FC = () => {
  const location = useLocation()
  const onLevels = location.pathname.endsWith('/levels')

  return (
    <Styled.Shell>
      <Styled.Sidebar>
        <Styled.SidebarTitle>{t('console.title')}</Styled.SidebarTitle>
        <Styled.NavItem href="/-/console" $active={!onLevels}>
          {t('console.nav.overview')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/levels" $active={onLevels}>
          {t('console.nav.levels')}
        </Styled.NavItem>
      </Styled.Sidebar>
      <Styled.Main>
        <Routes>
          <Route index element={<ConsoleDashboard />} />
          <Route path="levels" element={<ConsoleLevels />} />
        </Routes>
      </Styled.Main>
    </Styled.Shell>
  )
}

export default Console

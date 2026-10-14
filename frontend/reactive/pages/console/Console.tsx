import * as React from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import { t } from '~util/i18n'
import ConsoleDashboard from './ConsoleDashboard'
import ConsoleLevels from './ConsoleLevels'
import ConsoleUsers from './ConsoleUsers'
import * as Styled from './Console.styles'

const Console: React.FC = () => {
  const location = useLocation()
  // /-/console/levels、/-/console/users 等：取 console 之后的第一段路径决定高亮
  const section = location.pathname.replace(/\/$/, '').split('/').pop() || ''

  return (
    <Styled.Shell>
      <Styled.Sidebar>
        <Styled.SidebarTitle>{t('console.title')}</Styled.SidebarTitle>
        <Styled.NavItem href="/-/console" $active={section === 'console'}>
          {t('console.nav.overview')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/levels" $active={section === 'levels'}>
          {t('console.nav.levels')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/users" $active={section === 'users'}>
          {t('console.nav.users')}
        </Styled.NavItem>
      </Styled.Sidebar>
      <Styled.Main>
        <Routes>
          <Route index element={<ConsoleDashboard />} />
          <Route path="levels" element={<ConsoleLevels />} />
          <Route path="users" element={<ConsoleUsers />} />
        </Routes>
      </Styled.Main>
    </Styled.Shell>
  )
}

export default Console

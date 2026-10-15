import * as React from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import { t } from '~util/i18n'
import ConsoleDashboard from './ConsoleDashboard'
import ConsoleLevels from './ConsoleLevels'
import ConsoleUsers from './ConsoleUsers'
import ConsoleRoles from './ConsoleRoles'
import ConsoleSite from './ConsoleSite'
import ConsoleReports from './ConsoleReports'
import ConsolePages from './ConsolePages'
import ConsoleTags from './ConsoleTags'
import ConsolePageCats from './ConsolePageCats'
import ConsoleForum from './ConsoleForum'
import ConsoleForumPosts from './ConsoleForumPosts'
import ConsoleTickets from './ConsoleTickets'
import ConsoleInvites from './ConsoleInvites'
import ConsoleAdminLog from './ConsoleAdminLog'
import ConsoleThemes from './ConsoleThemes'
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
        <Styled.NavItem href="/-/console/roles" $active={section === 'roles'}>
          {t('console.nav.roles')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/site" $active={section === 'site'}>
          {t('console.nav.site')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/reports" $active={section === 'reports'}>
          {t('console.nav.reports')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/pages" $active={section === 'pages'}>
          {t('console.nav.pages')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/tags" $active={section === 'tags'}>
          {t('console.nav.tags')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/page-categories" $active={section === 'page-categories'}>
          {t('console.nav.pagecats')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/forum" $active={section === 'forum'}>
          {t('console.nav.forum')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/forum-posts" $active={section === 'forum-posts'}>
          {t('console.nav.forumposts')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/tickets" $active={section === 'tickets'}>
          {t('console.nav.tickets')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/invites" $active={section === 'invites'}>
          {t('console.nav.invites')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/admin-log" $active={section === 'admin-log'}>
          {t('console.nav.adminlog')}
        </Styled.NavItem>
        <Styled.NavItem href="/-/console/themes" $active={section === 'themes'}>
          {t('console.nav.themes')}
        </Styled.NavItem>
      </Styled.Sidebar>
      <Styled.Main>
        <Routes>
          <Route index element={<ConsoleDashboard />} />
          <Route path="levels" element={<ConsoleLevels />} />
          <Route path="users" element={<ConsoleUsers />} />
          <Route path="roles" element={<ConsoleRoles />} />
          <Route path="site" element={<ConsoleSite />} />
          <Route path="reports" element={<ConsoleReports />} />
          <Route path="pages" element={<ConsolePages />} />
          <Route path="tags" element={<ConsoleTags />} />
          <Route path="page-categories" element={<ConsolePageCats />} />
          <Route path="forum" element={<ConsoleForum />} />
          <Route path="forum-posts" element={<ConsoleForumPosts />} />
          <Route path="tickets" element={<ConsoleTickets />} />
          <Route path="invites" element={<ConsoleInvites />} />
          <Route path="admin-log" element={<ConsoleAdminLog />} />
          <Route path="themes" element={<ConsoleThemes />} />
        </Routes>
      </Styled.Main>
    </Styled.Shell>
  )
}

export default Console

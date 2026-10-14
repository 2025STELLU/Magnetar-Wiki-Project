import styled from 'styled-components'

export const Shell = styled.div`
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 0;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 4px;
  overflow: hidden;
  min-height: 560px;
  background: ${({ theme }) => theme.uiBackground};

  @media (max-width: 700px) {
    grid-template-columns: 1fr;
  }
`

export const Sidebar = styled.nav`
  background: ${({ theme }) => theme.bar};
  color: ${({ theme }) => theme.barForeground};
  padding: 20px 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
`

export const SidebarTitle = styled.div`
  font-size: 15px;
  font-weight: 600;
  padding: 0 20px 14px;
  color: ${({ theme }) => theme.barForeground};
`

export const NavItem = styled.a<{ $active?: boolean }>`
  display: block;
  padding: 9px 20px;
  font-size: 14px;
  color: ${({ theme }) => theme.barMuted};
  text-decoration: none;
  border-left: 3px solid transparent;

  ${({ $active, theme }) =>
    $active &&
    `
    color: ${theme.barForeground};
    background: ${theme.uiSelectionHighlight};
    border-left-color: ${theme.accent};
    font-weight: 600;
  `}

  &:hover {
    color: ${({ theme }) => theme.barForeground};
    text-decoration: none;
  }
`

export const Main = styled.main`
  padding: 28px 32px;
  min-width: 0;
`

export const H1 = styled.h1`
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 6px;
  color: ${({ theme }) => theme.foreground};
`

export const Desc = styled.p`
  font-size: 14px;
  color: ${({ theme }) => theme.uiForeground};
  margin: 0 0 24px;
`

export const CardGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
`

export const Card = styled.div`
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 4px;
  padding: 18px;
  background: ${({ theme }) => theme.uiBackground};
`

export const CardValue = styled.div`
  font-size: 30px;
  font-weight: 600;
  color: ${({ theme }) => theme.foreground};
  font-variant-numeric: tabular-nums;
`

export const CardLabel = styled.div`
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
  margin-top: 4px;
`

export const Field = styled.div`
  margin-bottom: 18px;
  max-width: 420px;
`

export const Label = styled.label`
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: ${({ theme }) => theme.foreground};
  margin-bottom: 6px;
`

export const Input = styled.input`
  width: 100%;
  padding: 8px 10px;
  font-size: 14px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 3px;
  color: ${({ theme }) => theme.foreground};
  background: ${({ theme }) => theme.uiBackground};
`

export const Table = styled.table`
  border-collapse: collapse;
  width: 100%;
  max-width: 480px;
  margin-bottom: 20px;

  th,
  td {
    text-align: left;
    padding: 8px 10px;
    font-size: 14px;
    border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  }

  th {
    color: ${({ theme }) => theme.uiForeground};
    font-weight: 600;
    font-size: 13px;
  }

  td input {
    width: 140px;
    padding: 6px 8px;
    font-size: 14px;
    font-variant-numeric: tabular-nums;
    border: 1px solid ${({ theme }) => theme.windowStrong};
    border-radius: 3px;
  }
`

export const Button = styled.button`
  padding: 9px 22px;
  font-size: 14px;
  font-weight: 600;
  color: ${({ theme }) => theme.uiSelectionForeground};
  background: ${({ theme }) => theme.accent};
  border: none;
  border-radius: 3px;
  cursor: pointer;

  &:disabled {
    opacity: 0.55;
    cursor: default;
  }
`

export const Notice = styled.div<{ $kind?: 'error' | 'ok' }>`
  margin: 0 0 16px;
  padding: 10px 14px;
  font-size: 14px;
  border-radius: 3px;
  color: ${({ $kind, theme }) => ($kind === 'ok' ? theme.foreground : '#a33')};
  background: ${({ $kind, theme }) => ($kind === 'ok' ? theme.higlightBackground : '#fdeeee')};
  border: 1px solid ${({ $kind, theme }) => ($kind === 'ok' ? theme.windowStrong : '#eecaca')};
`

export const Loading = styled.div`
  font-size: 14px;
  color: ${({ theme }) => theme.quiet};
  padding: 24px 0;
`

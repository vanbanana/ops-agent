// Data: None (xterm.js)
import { type FC } from 'react'
import { TerminalDrawer } from '../../TerminalDrawer'

export const TerminalApp: FC = () => {
  return <TerminalDrawer isFullscreen={true} onToggleFullscreen={() => {}} />
}

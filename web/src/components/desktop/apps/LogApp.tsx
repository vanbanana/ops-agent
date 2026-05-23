// Data: POST /api/v1/desktop/probe/logs_journal
import { type FC } from 'react'
import { ProbeApp } from './ProbeApp'

interface LogAppProps { connected: boolean }

export const LogApp: FC<LogAppProps> = ({ connected }) => (
  <ProbeApp probeName="logs_journal" connected={connected} />
)

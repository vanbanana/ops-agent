// Data: POST /api/v1/desktop/probe/process
import { type FC } from 'react'
import { ProbeApp } from './ProbeApp'

interface ProcessAppProps { connected: boolean }

export const ProcessApp: FC<ProcessAppProps> = ({ connected }) => (
  <ProbeApp probeName="process" connected={connected} />
)

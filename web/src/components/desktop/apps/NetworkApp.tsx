// Data: POST /api/v1/desktop/probe/network_connections
import { type FC } from 'react'
import { ProbeApp } from './ProbeApp'

interface NetworkAppProps { connected: boolean }

export const NetworkApp: FC<NetworkAppProps> = ({ connected }) => (
  <ProbeApp probeName="network_connections" connected={connected} />
)

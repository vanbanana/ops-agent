// Data: POST /api/v1/desktop/probe/service_status
import { type FC } from 'react'
import { ProbeApp } from './ProbeApp'

interface ServiceAppProps { connected: boolean }

export const ServiceApp: FC<ServiceAppProps> = ({ connected }) => (
  <ProbeApp probeName="service_status" connected={connected} />
)

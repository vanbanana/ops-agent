// Data: POST /api/v1/desktop/probe/large_files
import { type FC } from 'react'
import { ProbeApp } from './ProbeApp'

interface TrashAppProps { connected: boolean }

export const TrashApp: FC<TrashAppProps> = ({ connected }) => (
  <ProbeApp probeName="large_files" connected={connected} title="可清理的大文件 (>100MB)" />
)

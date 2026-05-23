// Data: POST /api/v1/desktop/probe/{name} (Task 14)
// 每30s轮询探针获取资源数据，不走 LLM
import { useEffect, useCallback } from 'react'
import type { ResourceData } from '../types/api'

const POLL_INTERVAL = 30_000

type Dispatch = (action: { type: 'UPDATE_RESOURCES'; data: Partial<ResourceData> }) => void

export function useResourcePolling(connected: boolean, dispatch: Dispatch) {
  const fetchProbe = useCallback(async (name: string) => {
    try {
      const res = await fetch(`/api/v1/desktop/probe/${name}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      })
      if (!res.ok) return null
      const json = await res.json()
      return json.data?.result ?? json.data?.summary ?? null
    } catch {
      return null
    }
  }, [])

  const pollAll = useCallback(async () => {
    if (!connected) return

    // Fetch all probes — backend returns raw text in `result` or `summary`
    const [diskRaw, topRaw, memRaw, procRaw, netRaw] = await Promise.all([
      fetchProbe('disk'),
      fetchProbe('top'),
      fetchProbe('memory'),
      fetchProbe('process'),
      fetchProbe('network_connections'),
    ])

    // Parse disk
    if (diskRaw && typeof diskRaw === 'string') {
      const disks = parseDfOutput(diskRaw)
      if (disks.length > 0) dispatch({ type: 'UPDATE_RESOURCES', data: { disk: disks } })
    }

    // Parse top — macOS format: "Load Avg: 3.45, ..." + "CPU usage: 16.5% user, ..."
    // + "Processes: 533 total" + "PhysMem: 7548M used (...), 86M unused."
    if (topRaw && typeof topRaw === 'string') {
      // Load
      const loadMatch = topRaw.match(/Load Avg:\s*([\d.]+)/i)
      if (loadMatch) {
        dispatch({ type: 'UPDATE_RESOURCES', data: { load: parseFloat(loadMatch[1]) } })
      }

      // Process count from top output
      const procMatch = topRaw.match(/Processes:\s*(\d+)\s*total/i)
      if (procMatch) {
        dispatch({ type: 'UPDATE_RESOURCES', data: { processes: parseInt(procMatch[1]) } })
      }

      // Memory from PhysMem line (macOS)
      const physMatch = topRaw.match(/PhysMem:\s*([\d.]+)([MG])\s*used.*?,\s*([\d.]+)([MG])\s*unused/i)
      if (physMatch) {
        const usedNum = parseFloat(physMatch[1]) * (physMatch[2] === 'G' ? 1024 : 1)
        const freeNum = parseFloat(physMatch[3]) * (physMatch[4] === 'G' ? 1024 : 1)
        const totalNum = usedNum + freeNum
        const percent = Math.round((usedNum / totalNum) * 100)
        const used = usedNum >= 1024 ? `${(usedNum / 1024).toFixed(1)}G` : `${Math.round(usedNum)}M`
        const total = totalNum >= 1024 ? `${(totalNum / 1024).toFixed(1)}G` : `${Math.round(totalNum)}M`
        dispatch({ type: 'UPDATE_RESOURCES', data: { memory: { used, total, percent } } })
      }
    }

    // Fallback: parse memory probe if top didn't provide it
    if (memRaw && typeof memRaw === 'string' && topRaw && !topRaw.includes('PhysMem')) {
      const mem = parseMemOutput(memRaw)
      if (mem) dispatch({ type: 'UPDATE_RESOURCES', data: { memory: mem } })
    }

    // Process count fallback
    if (procRaw && typeof procRaw === 'string' && !topRaw?.includes('Processes:')) {
      const lines = procRaw.trim().split('\n').length - 1
      if (lines > 0) dispatch({ type: 'UPDATE_RESOURCES', data: { processes: lines } })
    }

    // Network/ports
    if (netRaw && typeof netRaw === 'string') {
      // Count LISTEN lines for port count
      const listenLines = netRaw.split('\n').filter(l => l.includes('LISTEN') || l.includes('listen'))
      dispatch({ type: 'UPDATE_RESOURCES', data: { ports: listenLines.length || 0 } })
    }
  }, [connected, fetchProbe, dispatch])

  useEffect(() => {
    if (!connected) return

    // Initial fetch
    pollAll()

    // Poll every 30s
    const interval = setInterval(pollAll, POLL_INTERVAL)
    return () => clearInterval(interval)
  }, [connected, pollAll])
}


// Parse macOS/Linux df -h output into structured disk data
function parseDfOutput(raw: string): Array<{ mount: string; percent: number; used: string; total: string }> {
  const lines = raw.trim().split('\n')
  const results: Array<{ mount: string; percent: number; used: string; total: string }> = []

  for (const line of lines.slice(1)) { // skip header
    const parts = line.trim().split(/\s+/)
    if (parts.length < 5) continue

    const mount = parts[parts.length - 1]

    // Only keep meaningful mounts
    if (mount === '/' ||
        mount === '/home' ||
        mount.startsWith('/var') ||
        mount.startsWith('/tmp') ||
        mount.startsWith('/opt') ||
        mount.startsWith('/Volumes/Data')) {
      // Find capacity column (contains %)
      let percent = 0
      let total = ''
      let used = ''
      for (let i = 1; i < parts.length - 1; i++) {
        if (parts[i].endsWith('%')) {
          percent = parseInt(parts[i]) || 0
          total = parts[1] || ''
          used = parts[2] || ''
          break
        }
      }
      if (percent >= 0) {
        // Shorten mount for display
        const displayMount = mount === '/System/Volumes/Data' ? '/Data' : mount
        results.push({ mount: displayMount, percent, used, total })
      }
    }
  }

  // If no standard mounts found, take root and any > 50% usage
  if (results.length === 0) {
    for (const line of lines.slice(1)) {
      const parts = line.trim().split(/\s+/)
      if (parts.length < 5) continue
      const mount = parts[parts.length - 1]
      // Skip virtual/meta filesystems
      if (parts[0] === 'devfs' || parts[0] === 'map' || mount.startsWith('/System/Volumes/VM') ||
          mount.startsWith('/System/Volumes/Preboot') || mount.startsWith('/System/Volumes/Update') ||
          mount.startsWith('/System/Volumes/xarts') || mount.startsWith('/System/Volumes/iSCPreboot') ||
          mount.startsWith('/System/Volumes/Hardware') || mount.startsWith('/System/Volumes/Data/home') ||
          mount.startsWith('/private/var/folders')) continue

      let percent = 0
      let total = ''
      let used = ''
      for (let i = 1; i < parts.length - 1; i++) {
        if (parts[i].endsWith('%')) {
          percent = parseInt(parts[i]) || 0
          total = parts[1] || ''
          used = parts[2] || ''
          break
        }
      }
      if (percent > 0) {
        results.push({ mount, percent, used, total })
      }
    }
  }

  return results
}

// Parse free -h or vm_stat output
function parseMemOutput(raw: string): { used: string; total: string; percent: number } | null {
  // Try Linux free format: "Mem: total used free..."
  const memLine = raw.split('\n').find(l => l.startsWith('Mem:') || l.includes('Mem:'))
  if (memLine) {
    const parts = memLine.trim().split(/\s+/)
    // free: Mem: total used free shared buff/cache available
    const total = parts[1] || '--'
    const used = parts[2] || '--'
    const totalNum = parseFloat(total)
    const usedNum = parseFloat(used)
    const percent = totalNum > 0 ? Math.round((usedNum / totalNum) * 100) : 0
    return { used, total, percent }
  }

  // macOS: try to parse vm_stat or sysctl output
  const pageSize = 16384 // macOS default
  const pagesMatch = raw.match(/Pages\s+active:\s+(\d+)/i)
  const freeMatch = raw.match(/Pages\s+free:\s+(\d+)/i)
  if (pagesMatch && freeMatch) {
    const active = parseInt(pagesMatch[1]) * pageSize / (1024 * 1024 * 1024)
    const free = parseInt(freeMatch[1]) * pageSize / (1024 * 1024 * 1024)
    const total = active + free
    const percent = Math.round((active / total) * 100)
    return { used: `${active.toFixed(1)}G`, total: `${total.toFixed(1)}G`, percent }
  }

  return null
}

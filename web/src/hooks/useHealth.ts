// Data: GET /health
import { useState, useEffect, useCallback } from 'react'
import type { HealthResponse } from '../types/api'

const POLL_INTERVAL = 30_000 // 30s

export function useHealth() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [connected, setConnected] = useState(false)

  const fetchHealth = useCallback(async () => {
    try {
      const res = await fetch('/health')
      if (res.ok) {
        const data: HealthResponse = await res.json()
        setHealth(data)
        setConnected(true)
      } else {
        setHealth(null)
        setConnected(false)
      }
    } catch {
      setHealth(null)
      setConnected(false)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchHealth()
    const interval = setInterval(fetchHealth, POLL_INTERVAL)
    return () => clearInterval(interval)
  }, [fetchHealth])

  return { health, loading, connected, refresh: fetchHealth }
}

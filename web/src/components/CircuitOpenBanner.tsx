import React, { useEffect, useState } from 'react'

interface CircuitOpenBannerProps {
  retry_after_sec: number
  message: string
  onDismiss?: () => void
}

/**
 * CircuitOpenBanner -- shown when the LLM circuit breaker is open (API consistently failing).
 * Shows a countdown timer until automatic recovery.
 */
export const CircuitOpenBanner: React.FC<CircuitOpenBannerProps> = ({
  retry_after_sec,
  message,
  onDismiss,
}) => {
  const [remaining, setRemaining] = useState(retry_after_sec)

  useEffect(() => {
    if (remaining <= 0) {
      onDismiss?.()
      return
    }
    const timer = setInterval(() => {
      setRemaining(r => r - 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [remaining, onDismiss])

  return (
    <div className="bg-amber-50 dark:bg-amber-950 border border-amber-300 dark:border-amber-700 rounded-lg px-4 py-3 my-2 flex items-center gap-3">
      <span className="text-amber-600 text-xl">⚡</span>
      <div className="flex-1">
        <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
          熔断器已开启
        </p>
        <p className="text-xs text-amber-700 dark:text-amber-300 mt-0.5">
          {message} · {remaining}s 后自动恢复
        </p>
      </div>
    </div>
  )
}

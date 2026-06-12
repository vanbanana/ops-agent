import React from 'react'

interface DestructiveWarningProps {
  message: string
}

/**
 * DestructiveWarning — a yellow warning banner shown inside permission confirmation
 * dialogs or tool call cards when a command is allowed but carries risk.
 * This is informational only — it does not block execution.
 */
export const DestructiveWarning: React.FC<DestructiveWarningProps> = ({ message }) => {
  return (
    <div className="flex items-start gap-2 px-3 py-2 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded text-xs text-yellow-800 dark:text-yellow-200 mt-2">
      <span className="text-yellow-600 dark:text-yellow-400 mt-0.5">⚠️</span>
      <span>{message}</span>
    </div>
  )
}

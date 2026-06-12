import React from 'react'

interface OutputPersistedBadgeProps {
  path: string
  original_size: number
  tool: string
}

/**
 * OutputPersistedBadge -- displayed on a ToolCallCard when the output was too large
 * and has been persisted to disk. Shows file path and original size.
 */
export const OutputPersistedBadge: React.FC<OutputPersistedBadgeProps> = ({
  path,
  original_size,
}) => {
  const sizeKB = (original_size / 1024).toFixed(1)

  return (
    <div className="inline-flex items-center gap-1.5 px-2 py-0.5 bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded text-xs text-gray-600 dark:text-gray-400 mt-1">
      <span>💾</span>
      <span>
        输出已保存 ({sizeKB}KB) · <code className="font-mono text-[10px]">{path}</code>
      </span>
    </div>
  )
}

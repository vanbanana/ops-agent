import React from 'react'

interface PlanApprovalProps {
  planId: string
  planText: string
  steps: string[]
  onApprove: (planId: string) => void
  onReject: (planId: string) => void
}

/**
 * PlanApproval — shows the Agent's proposed operation plan and lets the user
 * approve or reject it as a whole (plan mode: read-only → plan → approve → execute).
 */
export const PlanApproval: React.FC<PlanApprovalProps> = ({
  planId,
  planText,
  steps,
  onApprove,
  onReject,
}) => {
  return (
    <div className="border border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950 rounded-lg p-4 my-3">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-blue-600 dark:text-blue-400 text-lg">📋</span>
        <h3 className="font-semibold text-blue-900 dark:text-blue-100 text-sm">
          操作计划待批准
        </h3>
      </div>

      {/* Plan summary */}
      {planText && (
        <p className="text-sm text-gray-700 dark:text-gray-300 mb-3 whitespace-pre-wrap">
          {planText}
        </p>
      )}

      {/* Steps list */}
      {steps.length > 0 && (
        <ol className="list-decimal list-inside space-y-1 mb-4 text-sm text-gray-800 dark:text-gray-200">
          {steps.map((step, i) => (
            <li key={i} className="py-0.5">{step}</li>
          ))}
        </ol>
      )}

      {/* Action buttons */}
      <div className="flex gap-3">
        <button
          onClick={() => onApprove(planId)}
          className="px-4 py-1.5 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded transition-colors"
        >
          ✓ 批准执行
        </button>
        <button
          onClick={() => onReject(planId)}
          className="px-4 py-1.5 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 text-sm font-medium rounded transition-colors"
        >
          ✕ 拒绝
        </button>
      </div>
    </div>
  )
}

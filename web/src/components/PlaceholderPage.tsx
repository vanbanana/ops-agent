// Placeholder for pages that are not yet implemented
import { type FC } from 'react'

interface PlaceholderPageProps {
  icon: string
  title: string
  description: string
}

export const PlaceholderPage: FC<PlaceholderPageProps> = ({ icon, title, description }) => {
  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 12,
        background: 'var(--ops-bg-canvas)',
      }}
    >
      <span
        className="material-symbols-outlined"
        style={{ fontSize: 40, color: 'var(--ops-fg-muted)', opacity: 0.5 }}
      >
        {icon}
      </span>
      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, fontWeight: 500, color: 'var(--ops-fg-secondary)' }}>
        {title}
      </span>
      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>
        {description}
      </span>
    </div>
  )
}

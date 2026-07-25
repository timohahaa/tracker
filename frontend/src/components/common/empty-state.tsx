import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

interface EmptyStateProps {
  /** Optional lucide icon rendered above the title. */
  icon?: LucideIcon
  title: string
  description?: string
  /** Optional action slot, e.g. a Button. */
  action?: ReactNode
  className?: string
}

// Neutral empty-collection placeholder (no items, no results, ...).
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-8 text-center',
        className,
      )}
    >
      {Icon ? (
        <Icon className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      ) : null}
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">{title}</h2>
        {description ? (
          <p className="text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {action ? <div className="pt-1">{action}</div> : null}
    </div>
  )
}

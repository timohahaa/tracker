import { STATUS_GROUPS, type StatusGroup } from '@/lib/status'
import { cn } from '@/lib/utils'

interface StatusBadgeProps {
  status: StatusGroup
  /** Hide the text label, showing only the icon. */
  iconOnly?: boolean
  className?: string
}

export function StatusBadge({ status, iconOnly, className }: StatusBadgeProps) {
  const token = STATUS_GROUPS[status]
  const Icon = token.icon

  return (
    <span className={cn('inline-flex items-center gap-1.5 text-sm', className)}>
      <Icon
        className="h-4 w-4 shrink-0"
        style={{ color: token.color }}
        aria-hidden="true"
      />
      {iconOnly ? (
        <span className="sr-only">{token.label}</span>
      ) : (
        <span>{token.label}</span>
      )}
    </span>
  )
}

import { TriangleAlert } from 'lucide-react'

import { PRIORITIES, type Priority } from '@/lib/priority'
import { cn } from '@/lib/utils'

interface PriorityIconProps {
  priority: Priority
  className?: string
}

// Linear-style "signal bars" glyph: three ascending bars, filled according to
// the priority level. Urgent uses a filled alert triangle instead.
function PriorityBars({ filled, color }: { filled: number; color: string }) {
  const bars = [
    { x: 1, y: 9, height: 5 },
    { x: 6, y: 6, height: 8 },
    { x: 11, y: 3, height: 11 },
  ]
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-4 w-4 shrink-0"
      fill="none"
      aria-hidden="true"
    >
      {bars.map((bar, i) => (
        <rect
          key={bar.x}
          x={bar.x}
          y={bar.y}
          width={3}
          height={bar.height}
          rx={1}
          fill={i < filled ? color : 'currentColor'}
          opacity={i < filled ? 1 : 0.3}
        />
      ))}
    </svg>
  )
}

export function PriorityIcon({ priority, className }: PriorityIconProps) {
  const token = PRIORITIES[priority]

  return (
    <span
      className={cn('inline-flex text-muted-foreground', className)}
      style={priority === 'urgent' ? { color: token.color } : undefined}
      title={token.label}
    >
      {priority === 'urgent' ? (
        <TriangleAlert className="h-4 w-4 shrink-0" aria-hidden="true" />
      ) : (
        <PriorityBars filled={token.filledBars} color={token.color} />
      )}
      <span className="sr-only">{token.label}</span>
    </span>
  )
}

interface PriorityBadgeProps {
  priority: Priority
  className?: string
}

/** Priority icon + text label, for menus and detail views. */
export function PriorityBadge({ priority, className }: PriorityBadgeProps) {
  const token = PRIORITIES[priority]
  return (
    <span className={cn('inline-flex items-center gap-1.5 text-sm', className)}>
      <PriorityIcon priority={priority} />
      <span>{token.label}</span>
    </span>
  )
}

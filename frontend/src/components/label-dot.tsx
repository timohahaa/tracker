import { readableTextColor } from '@/lib/label-color'
import { cn } from '@/lib/utils'

interface LabelDotProps {
  /** Hex color from the backend (`label.color`). */
  color: string
  className?: string
}

/** A small colored dot for an issue label. */
export function LabelDot({ color, className }: LabelDotProps) {
  return (
    <span
      className={cn(
        'inline-block h-2.5 w-2.5 shrink-0 rounded-full',
        className,
      )}
      style={{ backgroundColor: color }}
      aria-hidden="true"
    />
  )
}

interface LabelChipProps {
  name: string
  /** Hex color from the backend (`label.color`). */
  color: string
  /** Render as a solid filled chip with contrast-aware text. */
  filled?: boolean
  className?: string
}

/** Label rendered as dot + text, or as a solid contrast-aware chip. */
export function LabelChip({ name, color, filled, className }: LabelChipProps) {
  if (filled) {
    return (
      <span
        className={cn(
          'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
          className,
        )}
        style={{ backgroundColor: color, color: readableTextColor(color) }}
      >
        {name}
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs',
        className,
      )}
    >
      <LabelDot color={color} />
      {name}
    </span>
  )
}

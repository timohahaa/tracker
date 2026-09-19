import { Check } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { isValidHex, LABEL_COLORS, readableTextColor } from '@/lib/label-color'
import { cn } from '@/lib/utils'

interface ColorPickerProps {
  /** Current hex value (with leading `#`). */
  value: string
  onChange: (hex: string) => void
}

// Label color picker: a curated swatch grid plus a free hex input. The parent
// form owns validation (zod); this component only edits the raw string and
// highlights the swatch matching the current value.
export function ColorPicker({ value, onChange }: ColorPickerProps) {
  const normalized = value.trim().toLowerCase()

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        {LABEL_COLORS.map((color) => {
          const selected = color.toLowerCase() === normalized
          return (
            <button
              key={color}
              type="button"
              onClick={() => onChange(color)}
              className={cn(
                'flex h-7 w-7 items-center justify-center rounded-full ring-offset-background transition-shadow',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                selected && 'ring-2 ring-ring ring-offset-2',
              )}
              style={{ backgroundColor: color }}
              aria-label={`Use color ${color}`}
              aria-pressed={selected}
            >
              {selected ? (
                <Check
                  className="h-4 w-4"
                  style={{ color: readableTextColor(color) }}
                />
              ) : null}
            </button>
          )
        })}
      </div>

      <div className="flex items-center gap-2">
        <span
          className="h-7 w-7 shrink-0 rounded-md border"
          style={{
            backgroundColor: isValidHex(value) ? value : 'transparent',
          }}
          aria-hidden="true"
        />
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="#5e6ad2"
          className="font-mono"
          aria-label="Hex color"
          spellCheck={false}
        />
      </div>
    </div>
  )
}

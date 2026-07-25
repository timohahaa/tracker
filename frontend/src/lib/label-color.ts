// Helpers for rendering backend-provided label colors (`label.color`, a hex
// string) with a readable text color.

function normalizeHex(hex: string): string | null {
  const value = hex.trim().replace(/^#/, '')
  if (/^[0-9a-fA-F]{3}$/.test(value)) {
    return value
      .split('')
      .map((c) => c + c)
      .join('')
  }
  if (/^[0-9a-fA-F]{6}$/.test(value)) return value
  return null
}

/** Parse a hex color into [r, g, b] (0-255), or null if invalid. */
export function parseHex(hex: string): [number, number, number] | null {
  const normalized = normalizeHex(hex)
  if (!normalized) return null
  return [
    parseInt(normalized.slice(0, 2), 16),
    parseInt(normalized.slice(2, 4), 16),
    parseInt(normalized.slice(4, 6), 16),
  ]
}

/** Relative luminance per WCAG (0 = black, 1 = white). */
export function relativeLuminance(hex: string): number {
  const rgb = parseHex(hex)
  if (!rgb) return 0
  const [r, g, b] = rgb.map((channel) => {
    const c = channel / 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

/**
 * Pick a readable foreground (black or white) for text placed on the given
 * background hex.
 */
export function readableTextColor(backgroundHex: string): '#000000' | '#ffffff' {
  return relativeLuminance(backgroundHex) > 0.5 ? '#000000' : '#ffffff'
}

// Priority mirrors the backend enum (see docs/tech/api.md):
// urgent | high | medium | low | none.
export type Priority = 'urgent' | 'high' | 'medium' | 'low' | 'none'

export interface PriorityToken {
    value: Priority
    label: string
    /** Semantic color (hex, theme-independent). */
    color: string
    /**
     * Number of filled bars in the "signal-bars" scale (0-3). Urgent is rendered
     * as a filled alert glyph rather than bars.
     */
    filledBars: number
}

export const PRIORITIES: Record<Priority, PriorityToken> = {
    urgent: { value: 'urgent', label: 'Urgent', color: '#eb5757', filledBars: 3 },
    high: { value: 'high', label: 'High', color: '#e2b53d', filledBars: 3 },
    medium: { value: 'medium', label: 'Medium', color: '#e2b53d', filledBars: 2 },
    low: { value: 'low', label: 'Low', color: '#e2b53d', filledBars: 1 },
    none: { value: 'none', label: 'No priority', color: '#8a8f98', filledBars: 0 },
}

export const PRIORITY_ORDER: Priority[] = [
    'urgent',
    'high',
    'medium',
    'low',
    'none',
]

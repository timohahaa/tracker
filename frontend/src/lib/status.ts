import {
  Circle,
  CircleCheck,
  CircleDashed,
  CircleDot,
  CircleX,
  type LucideIcon,
} from 'lucide-react'

// Status groups mirror the backend `status_group` enum (see docs/tech/api.md):
// backlog | planned | in_progress | completed | canceled.
export type StatusGroup =
  | 'backlog'
  | 'planned'
  | 'in_progress'
  | 'completed'
  | 'canceled'

export interface StatusGroupToken {
  /** Backend enum value. */
  value: StatusGroup
  /** Human-readable label. */
  label: string
  /** Semantic color (hex, theme-independent) for the icon. */
  color: string
  icon: LucideIcon
}

// Linear-like semantics: backlog/canceled muted gray, planned neutral, in-progress
// amber, completed brand indigo. Colors are chosen to read on both light and dark
// backgrounds.
export const STATUS_GROUPS: Record<StatusGroup, StatusGroupToken> = {
  backlog: {
    value: 'backlog',
    label: 'Backlog',
    color: '#8a8f98',
    icon: CircleDashed,
  },
  planned: {
    value: 'planned',
    label: 'Planned',
    color: '#6b7280',
    icon: Circle,
  },
  in_progress: {
    value: 'in_progress',
    label: 'In Progress',
    color: '#f2c94c',
    icon: CircleDot,
  },
  completed: {
    value: 'completed',
    label: 'Completed',
    color: '#5e6ad2',
    icon: CircleCheck,
  },
  canceled: {
    value: 'canceled',
    label: 'Canceled',
    color: '#95a2b3',
    icon: CircleX,
  },
}

export const STATUS_GROUP_ORDER: StatusGroup[] = [
  'backlog',
  'planned',
  'in_progress',
  'completed',
  'canceled',
]

import type { ProjectStatus } from './use-projects'

// Project status is a subset of the shared status-group enum (planned |
// in_progress | completed | canceled — no backlog), so `StatusBadge` and the
// status-group icons/colors render it directly.
export const PROJECT_STATUS_ORDER: ProjectStatus[] = [
  'planned',
  'in_progress',
  'completed',
  'canceled',
]

export const PROJECT_STATUS_LABEL: Record<ProjectStatus, string> = {
  planned: 'Planned',
  in_progress: 'In Progress',
  completed: 'Completed',
  canceled: 'Canceled',
}

// Coerce a raw status string into a known ProjectStatus, defaulting to
// "planned" for unknown/missing values.
export function toProjectStatus(value: string | undefined): ProjectStatus {
  return value && (PROJECT_STATUS_ORDER as string[]).includes(value)
    ? (value as ProjectStatus)
    : 'planned'
}

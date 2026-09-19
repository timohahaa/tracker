import { CalendarDays } from 'lucide-react'

import { LabelChip } from '@/components/label-dot'
import { PriorityIcon } from '@/components/priority-icon'
import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import type { Member } from '@/features/workspace/use-members'
import { PRIORITIES,type Priority } from '@/lib/priority'
import { type StatusGroup } from '@/lib/status'
import { cn } from '@/lib/utils'

import type { Issue } from './use-issues'
import type { StatusInfo } from './use-workspace-statuses'

// Two-letter initials from a member's display name, falling back to email.
function memberInitials(member: Member | undefined): string {
  const source = member?.display_name?.trim() || member?.email || ''
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function memberName(member: Member | undefined, userId: string): string {
  return member?.display_name?.trim() || member?.email || userId
}

// Short human date (e.g. "Mar 4"), or null for missing/invalid input.
function formatDue(iso: string | undefined): string | null {
  if (!iso) return null
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return null
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
  }).format(date)
}

function toPriority(value: string | undefined): Priority {
  return value && value in PRIORITIES ? (value as Priority) : 'none'
}

const MAX_AVATARS = 3

interface IssueRowProps {
  issue: Issue
  statusMap: Map<string, StatusInfo>
  memberMap: Map<string, Member>
}

// A single issue line: status icon, identifier, title, priority, labels,
// assignee avatars and due date. Display-only in this slice (detail/edit is
// 2.4.2), so it is not a link yet.
export function IssueRow({ issue, statusMap, memberMap }: IssueRowProps) {
  const group: StatusGroup =
    (issue.status_id && statusMap.get(issue.status_id)?.group) || 'backlog'
  const statusName = issue.status_id
    ? statusMap.get(issue.status_id)?.name
    : undefined
  const priority = toPriority(issue.priority)
  const due = formatDue(issue.due_date)
  const assignees = issue.assignees ?? []
  const labels = issue.labels ?? []

  return (
    <div className="flex items-center gap-3 border-b px-4 py-2.5 last:border-b-0 hover:bg-accent/50">
      <span className="shrink-0" title={statusName || undefined}>
        <StatusBadge status={group} iconOnly />
      </span>

      <span className="w-20 shrink-0 truncate font-mono text-xs text-muted-foreground">
        {issue.identifier}
      </span>

      <PriorityIcon priority={priority} className="shrink-0" />

      <span className="min-w-0 flex-1 truncate text-sm">{issue.title}</span>

      {labels.length > 0 ? (
        <div className="hidden shrink-0 items-center gap-1 lg:flex">
          {labels.slice(0, 3).map((label) => (
            <LabelChip
              key={label.id}
              name={label.name ?? ''}
              color={label.color ?? '#8a8f98'}
            />
          ))}
          {labels.length > 3 ? (
            <span className="text-xs text-muted-foreground">
              +{labels.length - 3}
            </span>
          ) : null}
        </div>
      ) : null}

      {due ? (
        <span className="hidden shrink-0 items-center gap-1 text-xs text-muted-foreground sm:flex">
          <CalendarDays className="h-3.5 w-3.5" aria-hidden="true" />
          {due}
        </span>
      ) : null}

      {assignees.length > 0 ? (
        <div className="flex shrink-0 -space-x-1.5">
          {assignees.slice(0, MAX_AVATARS).map((assignee) => {
            const userId = assignee.user_id ?? ''
            const member = memberMap.get(userId)
            return (
              <Avatar
                key={userId}
                className={cn(
                  'h-6 w-6 border-2 border-background',
                  assignee.is_primary && 'ring-1 ring-primary',
                )}
                title={memberName(member, userId)}
              >
                <AvatarFallback className="text-[10px]">
                  {memberInitials(member)}
                </AvatarFallback>
              </Avatar>
            )
          })}
          {assignees.length > MAX_AVATARS ? (
            <span className="flex h-6 w-6 items-center justify-center rounded-full border-2 border-background bg-muted text-[10px] text-muted-foreground">
              +{assignees.length - MAX_AVATARS}
            </span>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

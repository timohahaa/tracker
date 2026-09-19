import { InfiniteScroll } from '@/components/common/infinite-scroll'
import { StatusBadge } from '@/components/status-badge'
import type { Member } from '@/features/workspace/use-members'
import { STATUS_GROUP_ORDER,type StatusGroup } from '@/lib/status'

import { IssueRow } from './issue-row'
import type { Issue } from './use-issues'
import type { StatusInfo } from './use-workspace-statuses'

interface IssuesListProps {
  issues: Issue[]
  statusMap: Map<string, StatusInfo>
  memberMap: Map<string, Member>
  /** Group rows under status-group headers. Falls back to a flat list while the
   *  per-team statuses needed to resolve groups are still loading. */
  grouped: boolean
  hasNextPage: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => void
}

function groupOf(
  issue: Issue,
  statusMap: Map<string, StatusInfo>,
): StatusGroup {
  return (issue.status_id && statusMap.get(issue.status_id)?.group) || 'backlog'
}

export function IssuesList({
  issues,
  statusMap,
  memberMap,
  grouped,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
}: IssuesListProps) {
  const canGroup = grouped && statusMap.size > 0

  if (!canGroup) {
    return (
      <div className="rounded-lg border">
        {issues.map((issue) => (
          <IssueRow
            key={issue.id}
            issue={issue}
            statusMap={statusMap}
            memberMap={memberMap}
          />
        ))}
        <InfiniteScroll
          hasNextPage={hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          fetchNextPage={fetchNextPage}
        />
      </div>
    )
  }

  const buckets = new Map<StatusGroup, Issue[]>()
  for (const issue of issues) {
    const group = groupOf(issue, statusMap)
    const bucket = buckets.get(group)
    if (bucket) bucket.push(issue)
    else buckets.set(group, [issue])
  }

  const sections = STATUS_GROUP_ORDER.map((group) => ({
    group,
    items: buckets.get(group) ?? [],
  })).filter((section) => section.items.length > 0)

  return (
    <div className="space-y-6">
      {sections.map((section) => (
        <section key={section.group}>
          <div className="mb-1.5 flex items-center gap-2 px-1">
            <StatusBadge status={section.group} />
            <span className="text-xs text-muted-foreground">
              {section.items.length}
            </span>
          </div>
          <div className="rounded-lg border">
            {section.items.map((issue) => (
              <IssueRow
                key={issue.id}
                issue={issue}
                statusMap={statusMap}
                memberMap={memberMap}
              />
            ))}
          </div>
        </section>
      ))}

      <InfiniteScroll
        hasNextPage={hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
        fetchNextPage={fetchNextPage}
      />
    </div>
  )
}

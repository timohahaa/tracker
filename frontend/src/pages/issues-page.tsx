import { ListTodo, SearchX } from 'lucide-react'
import { useMemo, useState } from 'react'

import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { IssuesFilterBar } from '@/features/issue/issues-filter-bar'
import { IssuesList } from '@/features/issue/issues-list'
import { useIssueFilters } from '@/features/issue/use-issue-filters'
import { useIssues } from '@/features/issue/use-issues'
import { useWorkspaceStatuses } from '@/features/issue/use-workspace-statuses'
import { useLabels } from '@/features/label/use-labels'
import { useProjectsList } from '@/features/project/use-projects-list'
import { useTeams } from '@/features/team/use-teams'
import { type Member, useMembers } from '@/features/workspace/use-members'

// Workspace-wide issues list: filters + sort + full-text search reflected in the
// URL, grouped by status, cursor-paginated via infinite scroll. Display-only in
// this slice (create/detail/edit arrive in 2.4.2).
export default function IssuesPage() {
  const { id: workspaceID } = useCurrentWorkspace()
  const { filters, setFilter, setLabels, clearAll, hasActiveFilters } =
    useIssueFilters()
  const [grouped, setGrouped] = useState(true)

  const teamsQuery = useTeams(workspaceID)
  const membersQuery = useMembers(workspaceID)
  const labelsQuery = useLabels(workspaceID)
  const projectsQuery = useProjectsList(workspaceID)

  const teams = useMemo(() => teamsQuery.data?.data ?? [], [teamsQuery.data])
  const members = useMemo(
    () => membersQuery.data?.data ?? [],
    [membersQuery.data],
  )
  const labels = labelsQuery.data?.data ?? []
  const projects = projectsQuery.data?.data ?? []

  const teamIDs = useMemo(
    () => teams.map((team) => team.id).filter((id): id is string => !!id),
    [teams],
  )
  const { statusMap } = useWorkspaceStatuses(workspaceID, teamIDs)

  const memberMap = useMemo(() => {
    const map = new Map<string, Member>()
    for (const member of members) {
      if (member.user_id) map.set(member.user_id, member)
    }
    return map
  }, [members])

  const issuesQuery = useIssues(workspaceID, filters)
  const issues = useMemo(
    () => issuesQuery.data?.pages.flatMap((page) => page.data ?? []) ?? [],
    [issuesQuery.data],
  )

  return (
    <div className="mx-auto w-full max-w-5xl px-6 py-8">
      <header className="mb-4">
        <h1 className="text-2xl font-semibold tracking-tight">Issues</h1>
        <p className="text-sm text-muted-foreground">
          All issues across the workspace.
        </p>
      </header>

      <div className="mb-4">
        <IssuesFilterBar
          workspaceID={workspaceID}
          filters={filters}
          setFilter={setFilter}
          setLabels={setLabels}
          clearAll={clearAll}
          hasActiveFilters={hasActiveFilters}
          teams={teams}
          members={members}
          labels={labels}
          projects={projects}
          grouped={grouped}
          onToggleGrouped={setGrouped}
        />
      </div>

      {issuesQuery.isLoading ? (
        <ListSkeleton rows={8} />
      ) : issuesQuery.isError ? (
        <ErrorState
          error={issuesQuery.error}
          onRetry={() => issuesQuery.refetch()}
        />
      ) : issues.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            icon={SearchX}
            title="No matching issues"
            description="No issues match the current filters. Try clearing some of them."
          />
        ) : (
          <EmptyState
            icon={ListTodo}
            title="No issues yet"
            description="Issues you create will show up here."
          />
        )
      ) : (
        <IssuesList
          issues={issues}
          statusMap={statusMap}
          memberMap={memberMap}
          grouped={grouped}
          hasNextPage={issuesQuery.hasNextPage}
          isFetchingNextPage={issuesQuery.isFetchingNextPage}
          fetchNextPage={issuesQuery.fetchNextPage}
        />
      )}
    </div>
  )
}

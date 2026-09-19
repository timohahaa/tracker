import { useQueryClient } from '@tanstack/react-query'
import { ListTodo, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { InfiniteScroll } from '@/components/common/infinite-scroll'
import { ListSkeleton } from '@/components/common/skeletons'
import { Button } from '@/components/ui/button'
import { IssueRow } from '@/features/issue/issue-row'
import { useUnlinkIssueProject } from '@/features/issue/use-issue'
import { useIssues } from '@/features/issue/use-issues'
import { useWorkspaceStatuses } from '@/features/issue/use-workspace-statuses'
import { useTeams } from '@/features/team/use-teams'
import { type Member, useMembers } from '@/features/workspace/use-members'

import { LinkIssueDialog } from './link-issue-dialog'
import { projectQueryKey } from './use-projects'

function errorMessage(err: unknown): string {
  return err instanceof AppError ? err.message : 'Something went wrong'
}

interface ProjectIssuesProps {
  workspaceID: string
  projectID: string
}

// Issues belonging to a project: reuses the shared issues list hook (filtered by
// `project`) and the IssueRow component, with per-row unlink and a link dialog.
// Cursor-paginated via the shared InfiniteScroll trigger.
export function ProjectIssues({ workspaceID, projectID }: ProjectIssuesProps) {
  const queryClient = useQueryClient()
  const issuesQuery = useIssues(workspaceID, { project: projectID })
  const unlinkProject = useUnlinkIssueProject()
  const [pendingId, setPendingId] = useState<string | null>(null)

  const teamsQuery = useTeams(workspaceID)
  const membersQuery = useMembers(workspaceID)

  const teamIDs = useMemo(
    () =>
      (teamsQuery.data?.data ?? [])
        .map((team) => team.id)
        .filter((id): id is string => !!id),
    [teamsQuery.data],
  )
  const { statusMap } = useWorkspaceStatuses(workspaceID, teamIDs)

  const memberMap = useMemo(() => {
    const map = new Map<string, Member>()
    for (const member of membersQuery.data?.data ?? []) {
      if (member.user_id) map.set(member.user_id, member)
    }
    return map
  }, [membersQuery.data])

  const issues = useMemo(
    () => issuesQuery.data?.pages.flatMap((page) => page.data ?? []) ?? [],
    [issuesQuery.data],
  )

  const handleUnlink = (issueID: string) => {
    setPendingId(issueID)
    unlinkProject.mutate(
      { params: { path: { workspaceID, issueID } } },
      {
        onSuccess: () => {
          toast.success('Issue unlinked')
          queryClient.invalidateQueries({
            queryKey: projectQueryKey(workspaceID, projectID),
          })
        },
        onError: (err) => toast.error(errorMessage(err)),
        onSettled: () => setPendingId(null),
      },
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <p className="text-sm text-muted-foreground">
          Issues linked to this project.
        </p>
        <LinkIssueDialog workspaceID={workspaceID} projectID={projectID} />
      </div>

      {issuesQuery.isLoading ? (
        <ListSkeleton rows={6} />
      ) : issuesQuery.isError ? (
        <ErrorState
          error={issuesQuery.error}
          onRetry={() => issuesQuery.refetch()}
        />
      ) : issues.length === 0 ? (
        <EmptyState
          icon={ListTodo}
          title="No issues yet"
          description="Link existing issues to this project to see them here."
          action={
            <LinkIssueDialog workspaceID={workspaceID} projectID={projectID} />
          }
        />
      ) : (
        <>
          <div className="overflow-hidden rounded-lg border">
            {issues.map((issue) => {
              const issueID = issue.id ?? ''
              return (
                <IssueRow
                  key={issueID}
                  issue={issue}
                  statusMap={statusMap}
                  memberMap={memberMap}
                  action={
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-destructive"
                      title="Unlink from project"
                      disabled={pendingId === issueID || !issueID}
                      onClick={() => handleUnlink(issueID)}
                    >
                      <X className="h-4 w-4" />
                      <span className="sr-only">Unlink issue</span>
                    </Button>
                  }
                />
              )
            })}
          </div>
          <InfiniteScroll
            hasNextPage={issuesQuery.hasNextPage}
            isFetchingNextPage={issuesQuery.isFetchingNextPage}
            fetchNextPage={issuesQuery.fetchNextPage}
          />
        </>
      )}
    </div>
  )
}

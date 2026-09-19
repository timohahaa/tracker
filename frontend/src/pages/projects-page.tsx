import { FolderKanban } from 'lucide-react'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'

import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { InfiniteScroll } from '@/components/common/infinite-scroll'
import { ListSkeleton } from '@/components/common/skeletons'
import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { CreateProjectDialog } from '@/features/project/create-project-dialog'
import {
  memberInitials,
  memberName,
} from '@/features/project/member-display'
import { toProjectStatus } from '@/features/project/project-status'
import { useProjects } from '@/features/project/use-projects'
import type { Project } from '@/features/project/use-projects-list'
import { type Team, useTeams } from '@/features/team/use-teams'
import { type Member, useMembers } from '@/features/workspace/use-members'

function ProjectRow({
  project,
  slug,
  teamMap,
  memberMap,
}: {
  project: Project
  slug: string
  teamMap: Map<string, Team>
  memberMap: Map<string, Member>
}) {
  const lead = project.lead_id ? memberMap.get(project.lead_id) : undefined
  const teamKeys = (project.team_ids ?? [])
    .map((id) => teamMap.get(id)?.key)
    .filter((key): key is string => !!key)

  return (
    <Link
      to={`/${slug}/projects/${project.id}`}
      className="flex items-center gap-3 border-b px-4 py-3 transition-colors last:border-b-0 hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
    >
      <StatusBadge
        status={toProjectStatus(project.status)}
        iconOnly
        className="shrink-0"
      />
      <span className="min-w-0 flex-1 truncate text-sm font-medium">
        {project.name}
      </span>

      {teamKeys.length > 0 ? (
        <div className="hidden shrink-0 items-center gap-1 sm:flex">
          {teamKeys.slice(0, 3).map((key) => (
            <Badge key={key} variant="secondary" className="font-mono">
              {key}
            </Badge>
          ))}
          {teamKeys.length > 3 ? (
            <span className="text-xs text-muted-foreground">
              +{teamKeys.length - 3}
            </span>
          ) : null}
        </div>
      ) : null}

      {lead ? (
        <Avatar className="h-6 w-6 shrink-0" title={memberName(lead)}>
          <AvatarFallback className="text-[10px]">
            {memberInitials(lead)}
          </AvatarFallback>
        </Avatar>
      ) : null}
    </Link>
  )
}

// Projects list. Any workspace member can view and create projects. Rows link to
// the project detail page. Cursor-paginated via infinite scroll.
export default function ProjectsPage() {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const projectsQuery = useProjects(workspaceID)
  const teamsQuery = useTeams(workspaceID)
  const membersQuery = useMembers(workspaceID)

  const projects = useMemo(
    () => projectsQuery.data?.pages.flatMap((page) => page.data ?? []) ?? [],
    [projectsQuery.data],
  )

  const teamMap = useMemo(() => {
    const map = new Map<string, Team>()
    for (const team of teamsQuery.data?.data ?? []) {
      if (team.id) map.set(team.id, team)
    }
    return map
  }, [teamsQuery.data])

  const memberMap = useMemo(() => {
    const map = new Map<string, Member>()
    for (const member of membersQuery.data?.data ?? []) {
      if (member.user_id) map.set(member.user_id, member)
    }
    return map
  }, [membersQuery.data])

  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-8">
      <header className="mb-6 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Projects</h1>
          <p className="text-sm text-muted-foreground">
            Projects span teams and group related issues.
          </p>
        </div>
        <CreateProjectDialog />
      </header>

      {projectsQuery.isLoading ? (
        <ListSkeleton rows={5} />
      ) : projectsQuery.isError ? (
        <ErrorState
          error={projectsQuery.error}
          onRetry={() => projectsQuery.refetch()}
        />
      ) : projects.length === 0 ? (
        <EmptyState
          icon={FolderKanban}
          title="No projects yet"
          description="Create your first project to group related issues."
          action={<CreateProjectDialog />}
        />
      ) : (
        <>
          <div className="rounded-lg border">
            {projects.map((project) => (
              <ProjectRow
                key={project.id}
                project={project}
                slug={slug}
                teamMap={teamMap}
                memberMap={memberMap}
              />
            ))}
          </div>
          <InfiniteScroll
            hasNextPage={projectsQuery.hasNextPage}
            isFetchingNextPage={projectsQuery.isFetchingNextPage}
            fetchNextPage={projectsQuery.fetchNextPage}
          />
        </>
      )}
    </div>
  )
}

import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'

import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { StatusBadge } from '@/components/status-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { ProjectIssues } from '@/features/project/project-issues'
import { ProjectSettingsForm } from '@/features/project/project-settings-form'
import { toProjectStatus } from '@/features/project/project-status'
import { useProject } from '@/features/project/use-projects'

// Project detail: keyed by the project UUID in the URL. Tabbed shell (Overview /
// Issues). Any workspace member can edit; deletion is gated on the workspace
// owner/admin role.
export default function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const projectID = projectId ?? ''
  const { id: workspaceID, slug, role } = useCurrentWorkspace()

  const projectQuery = useProject(workspaceID, projectID)
  const canDelete = role === 'owner' || role === 'admin'

  if (projectQuery.isLoading) {
    return (
      <div className="mx-auto w-full max-w-3xl px-6 py-8">
        <Skeleton className="mb-6 h-8 w-48" />
        <ListSkeleton rows={4} />
      </div>
    )
  }

  if (projectQuery.isError || !projectQuery.data) {
    return (
      <div className="mx-auto w-full max-w-3xl px-6 py-8">
        <Link
          to={`/${slug}/projects`}
          className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to projects
        </Link>
        <ErrorState
          error={projectQuery.error}
          onRetry={() => projectQuery.refetch()}
        />
      </div>
    )
  }

  const project = projectQuery.data

  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-8">
      <Link
        to={`/${slug}/projects`}
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to projects
      </Link>

      <header className="mb-6 flex items-center gap-3">
        <StatusBadge
          status={toProjectStatus(project.status)}
          iconOnly
          className="shrink-0"
        />
        <h1 className="text-2xl font-semibold tracking-tight">
          {project.name}
        </h1>
      </header>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="issues">Issues</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="pt-4">
          <ProjectSettingsForm project={project} canDelete={canDelete} />
        </TabsContent>

        <TabsContent value="issues" className="pt-4">
          <ProjectIssues workspaceID={workspaceID} projectID={projectID} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

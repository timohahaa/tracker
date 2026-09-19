import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'

import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCurrentUser } from '@/features/auth/session-provider'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { LabelsManager } from '@/features/label/labels-manager'
import { TeamMembers } from '@/features/team/team-members'
import { TeamSettingsForm } from '@/features/team/team-settings-form'
import { TeamStatuses } from '@/features/team/team-statuses'
import { useTeamMembers } from '@/features/team/use-team-members'
import { useTeamStatuses } from '@/features/team/use-team-statuses'
import { useTeam } from '@/features/team/use-teams'

function StatusesTab({
  teamID,
  canManage,
}: {
  teamID: string
  canManage: boolean
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const query = useTeamStatuses(workspaceID, teamID)

  if (query.isLoading) return <ListSkeleton rows={5} />
  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  return (
    <TeamStatuses
      teamID={teamID}
      statuses={query.data?.data ?? []}
      canManage={canManage}
    />
  )
}

// Team detail: keyed by the team UUID in the URL. Tabbed shell (Settings /
// Statuses / Members). Write affordances are gated on team membership, except
// team deletion which is gated on the workspace owner/admin role.
export default function TeamDetailPage() {
  const { teamId } = useParams<{ teamId: string }>()
  const teamID = teamId ?? ''
  const { id: workspaceID, slug, role } = useCurrentWorkspace()
  const currentUserId = useCurrentUser()?.id

  const teamQuery = useTeam(workspaceID, teamID)
  const membersQuery = useTeamMembers(workspaceID, teamID)

  const isTeamMember =
    !!currentUserId &&
    (membersQuery.data?.data ?? []).some((m) => m.user_id === currentUserId)
  const canManage = isTeamMember
  const canDelete = role === 'owner' || role === 'admin'

  if (teamQuery.isLoading) {
    return (
      <div className="mx-auto w-full max-w-3xl px-6 py-8">
        <Skeleton className="mb-6 h-8 w-48" />
        <ListSkeleton rows={4} />
      </div>
    )
  }

  if (teamQuery.isError || !teamQuery.data) {
    return (
      <div className="mx-auto w-full max-w-3xl px-6 py-8">
        <Link
          to={`/${slug}/teams`}
          className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to teams
        </Link>
        <ErrorState
          error={teamQuery.error}
          onRetry={() => teamQuery.refetch()}
        />
      </div>
    )
  }

  const team = teamQuery.data

  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-8">
      <Link
        to={`/${slug}/teams`}
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to teams
      </Link>

      <header className="mb-6 flex items-center gap-3">
        <Badge variant="secondary" className="shrink-0 font-mono">
          {team.key}
        </Badge>
        <h1 className="text-2xl font-semibold tracking-tight">{team.name}</h1>
      </header>

      <Tabs defaultValue="settings">
        <TabsList>
          <TabsTrigger value="settings">Settings</TabsTrigger>
          <TabsTrigger value="statuses">Statuses</TabsTrigger>
          <TabsTrigger value="members">Members</TabsTrigger>
          <TabsTrigger value="labels">Labels</TabsTrigger>
        </TabsList>

        <TabsContent value="settings" className="pt-4">
          <TeamSettingsForm
            team={team}
            canManage={canManage}
            canDelete={canDelete}
          />
        </TabsContent>

        <TabsContent value="statuses" className="pt-4">
          <StatusesTab teamID={teamID} canManage={canManage} />
        </TabsContent>

        <TabsContent value="members" className="pt-4">
          <TeamMembers teamID={teamID} canManage={canManage} />
        </TabsContent>

        <TabsContent value="labels" className="pt-4">
          <LabelsManager teamID={teamID} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

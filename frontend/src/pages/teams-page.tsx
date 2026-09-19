import { Users } from 'lucide-react'
import { Link } from 'react-router-dom'

import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { Badge } from '@/components/ui/badge'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { CreateTeamDialog } from '@/features/team/create-team-dialog'
import { type Team, useTeams } from '@/features/team/use-teams'

function TeamRow({ team, slug }: { team: Team; slug: string }) {
  return (
    <Link
      to={`/${slug}/teams/${team.id}`}
      className="flex items-start gap-3 border-b p-4 transition-colors last:border-b-0 hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
    >
      <Badge variant="secondary" className="mt-0.5 shrink-0 font-mono">
        {team.key}
      </Badge>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{team.name}</p>
        {team.description ? (
          <p className="truncate text-xs text-muted-foreground">
            {team.description}
          </p>
        ) : null}
      </div>
    </Link>
  )
}

// Teams list. Any workspace member can view and create teams. Rows link to the
// team detail page.
export default function TeamsPage() {
  const { id, slug } = useCurrentWorkspace()
  const query = useTeams(id)

  const teams = query.data?.data ?? []

  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-8">
      <header className="mb-6 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Teams</h1>
          <p className="text-sm text-muted-foreground">
            Teams own their issues, statuses and members.
          </p>
        </div>
        <CreateTeamDialog />
      </header>

      {query.isLoading ? (
        <ListSkeleton rows={4} />
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : teams.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No teams yet"
          description="Create your first team to start tracking issues."
          action={<CreateTeamDialog />}
        />
      ) : (
        <div className="rounded-lg border">
          {teams.map((team) => (
            <TeamRow key={team.id} team={team} slug={slug} />
          ))}
        </div>
      )}
    </div>
  )
}

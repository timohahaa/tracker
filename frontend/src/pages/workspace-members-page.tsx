import { Users } from 'lucide-react'

import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { MembersList } from '@/features/workspace/members-list'
import { useMembers } from '@/features/workspace/use-members'

// Workspace members management. Everyone sees the roster; owner/admin get inline
// role changes and member removal (gated inside MembersList). Rendered inside
// SettingsLayout as the "members" tab.
export default function WorkspaceMembersPage() {
  const { id } = useCurrentWorkspace()
  const query = useMembers(id)

  if (query.isLoading) {
    return <ListSkeleton rows={4} />
  }

  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  const members = query.data?.data ?? []

  if (members.length === 0) {
    return (
      <EmptyState
        icon={Users}
        title="No members yet"
        description="Members you invite will appear here once the invite flow is available."
      />
    )
  }

  return (
    <section className="space-y-3">
      <p className="text-sm text-muted-foreground">
        {members.length} {members.length === 1 ? 'member' : 'members'}
      </p>
      <MembersList members={members} />
    </section>
  )
}

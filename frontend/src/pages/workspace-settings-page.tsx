import { ErrorState } from '@/components/common/error-state'
import { Skeleton } from '@/components/ui/skeleton'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { useWorkspace } from '@/features/workspace/use-workspace'
import { WorkspaceSettingsForm } from '@/features/workspace/workspace-settings-form'

// General workspace settings (name/slug). Editable by owner/admin; read-only for
// members. Rendered inside SettingsLayout as the index tab.
export default function WorkspaceSettingsPage() {
  const { id, role } = useCurrentWorkspace()
  const query = useWorkspace(id)

  const canEdit = role === 'owner' || role === 'admin'

  if (query.isLoading) {
    return (
      <div className="max-w-md space-y-4">
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-32" />
      </div>
    )
  }

  if (query.isError || !query.data) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  return (
    <div className="max-w-md">
      <WorkspaceSettingsForm workspace={query.data} canEdit={canEdit} />
    </div>
  )
}

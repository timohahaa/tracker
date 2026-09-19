import { CheckCircle2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { ErrorState } from '@/components/common/error-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'

import { GitlabSetupForm } from './gitlab-setup-form'
import { useDeleteGitlab, useGitlabIntegration } from './use-gitlab'

function formatDate(iso: string | undefined): string | null {
  if (!iso) return null
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return null
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

function DisconnectDialog() {
  const { id: workspaceID } = useCurrentWorkspace()
  const deleteGitlab = useDeleteGitlab()
  const [open, setOpen] = useState(false)

  const error =
    deleteGitlab.error instanceof AppError ? deleteGitlab.error : null

  const handleDelete = () => {
    deleteGitlab.mutate(
      { params: { path: { workspaceID } } },
      {
        onSuccess: () => {
          toast.success('GitLab integration disconnected')
          setOpen(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">Disconnect</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Disconnect GitLab</DialogTitle>
          <DialogDescription>
            Remove the GitLab integration for this workspace? Incoming webhook
            events will stop being processed. You can reconnect later.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <p className="text-sm text-destructive">{error.message}</p>
        ) : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteGitlab.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteGitlab.isPending}
          >
            {deleteGitlab.isPending ? 'Disconnecting…' : 'Disconnect'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// GitLab integration section (Settings > Integrations). Any member can view the
// connection status; only owner/admin can configure or disconnect. Stored
// credentials are never fetched or displayed.
export function GitlabIntegration() {
  const { id: workspaceID, role } = useCurrentWorkspace()
  const query = useGitlabIntegration(workspaceID)
  const [reconnecting, setReconnecting] = useState(false)

  const canManage = role === 'owner' || role === 'admin'

  if (query.isLoading) {
    return (
      <div className="max-w-md space-y-4">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-9 w-32" />
      </div>
    )
  }

  if (query.isError || !query.data) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  const { connected, settings, created_at } = query.data
  const baseUrl = settings?.base_url
  const since = formatDate(created_at)

  return (
    <div className="max-w-md space-y-6">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">GitLab</h2>
        <p className="text-sm text-muted-foreground">
          Link merge requests and branches to issues via their identifier (e.g.
          ENG-123). Credentials are stored encrypted and never displayed.
        </p>
      </div>

      {connected ? (
        <div className="space-y-4">
          <div className="rounded-lg border p-4">
            <div className="mb-3 flex items-center gap-2">
              <CheckCircle2
                className="h-4 w-4 text-emerald-500"
                aria-hidden="true"
              />
              <Badge variant="secondary">Connected</Badge>
            </div>
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">GitLab URL</dt>
                <dd className="truncate font-mono">{baseUrl || '—'}</dd>
              </div>
              {since ? (
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground">Connected since</dt>
                  <dd>{since}</dd>
                </div>
              ) : null}
            </dl>
          </div>

          {canManage ? (
            reconnecting ? (
              <div className="rounded-lg border p-4">
                <h3 className="mb-3 text-sm font-medium">Reconnect</h3>
                <GitlabSetupForm
                  submitLabel="Reconnect"
                  onSuccess={() => setReconnecting(false)}
                  onCancel={() => setReconnecting(false)}
                />
              </div>
            ) : (
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => setReconnecting(true)}
                >
                  Reconnect
                </Button>
                <DisconnectDialog />
              </div>
            )
          ) : null}
        </div>
      ) : canManage ? (
        <div className="rounded-lg border p-4">
          <GitlabSetupForm submitLabel="Connect" onSuccess={() => {}} />
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          Not connected. Only workspace owners and admins can configure the
          GitLab integration.
        </p>
      )}
    </div>
  )
}

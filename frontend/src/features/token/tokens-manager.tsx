import { KeyRound, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
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
import { useCurrentWorkspace } from '@/features/auth/workspace-context'

import { CreateTokenDialog } from './create-token-dialog'
import { type ApiToken, useDeleteToken, useTokens } from './use-tokens'

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

function RevokeTokenDialog({ token }: { token: ApiToken }) {
  const { id: workspaceID } = useCurrentWorkspace()
  const deleteToken = useDeleteToken()
  const [open, setOpen] = useState(false)

  const error = deleteToken.error instanceof AppError ? deleteToken.error : null

  const handleDelete = () => {
    deleteToken.mutate(
      { params: { path: { workspaceID, tokenID: token.id ?? '' } } },
      {
        onSuccess: () => {
          toast.success('Token revoked')
          setOpen(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-destructive"
          aria-label={`Revoke ${token.name}`}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke token</DialogTitle>
          <DialogDescription>
            Revoke <span className="font-medium">{token.name}</span>? Any client
            using it will stop working immediately. This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <p className="text-sm text-destructive">{error.message}</p>
        ) : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteToken.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteToken.isPending}
          >
            {deleteToken.isPending ? 'Revoking…' : 'Revoke'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function TokenRow({ token }: { token: ApiToken }) {
  const created = formatDate(token.created_at)
  const lastUsed = formatDate(token.last_used_at)

  return (
    <div className="flex items-center gap-3 border-b py-3 pl-2 last:border-b-0">
      <KeyRound
        className="h-4 w-4 shrink-0 text-muted-foreground"
        aria-hidden="true"
      />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{token.name}</p>
        <p className="text-xs text-muted-foreground">
          {created ? `Created ${created}` : 'Created —'}
          {' · '}
          {lastUsed ? `Last used ${lastUsed}` : 'Never used'}
        </p>
      </div>
      <div className="shrink-0">
        <RevokeTokenDialog token={token} />
      </div>
    </div>
  )
}

// API tokens section (Settings > API tokens). Per-user: lists the current user's
// tokens, creates new ones (raw token revealed once), and revokes them.
export function TokensManager() {
  const { id: workspaceID } = useCurrentWorkspace()
  const query = useTokens(workspaceID)

  if (query.isLoading) return <ListSkeleton rows={3} />
  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  const tokens = query.data?.data ?? []

  const createTrigger = (
    <Button variant="outline" size="sm">
      New token
    </Button>
  )

  return (
    <div className="max-w-2xl space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">API tokens</h2>
        <p className="text-sm text-muted-foreground">
          Personal tokens for the REST API and MCP. Pass one as a{' '}
          <code className="font-mono text-xs">Bearer</code> token in the
          Authorization header.
        </p>
      </div>

      <div className="flex justify-end">
        <CreateTokenDialog trigger={createTrigger} />
      </div>

      {tokens.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No API tokens yet"
          description="Create a token to access the API or connect an MCP client."
          action={<CreateTokenDialog trigger={createTrigger} />}
        />
      ) : (
        <div className="rounded-lg border px-2">
          {tokens.map((token) => (
            <TokenRow key={token.id} token={token} />
          ))}
        </div>
      )}
    </div>
  )
}

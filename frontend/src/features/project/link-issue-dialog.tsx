import { useQueryClient } from '@tanstack/react-query'
import { Link2, Plus } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { ErrorState } from '@/components/common/error-state'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useUpdateIssue } from '@/features/issue/use-issue'
import { useIssues } from '@/features/issue/use-issues'

import { projectQueryKey } from './use-projects'
import { PROJECTS_QUERY_KEY } from './use-projects-list'

function errorMessage(err: unknown): string {
  return err instanceof AppError ? err.message : 'Something went wrong'
}

// Debounce a fast-changing value (search input) so we don't refetch per keystroke.
function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}

interface LinkIssuePickerProps {
  workspaceID: string
  projectID: string
}

// The searchable candidate list. Split out so the issues query only runs while
// the dialog is open (mounted).
function LinkIssuePicker({ workspaceID, projectID }: LinkIssuePickerProps) {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebouncedValue(search)
  const updateIssue = useUpdateIssue()
  const [pendingId, setPendingId] = useState<string | null>(null)

  const issuesQuery = useIssues(workspaceID, { q: debouncedSearch || undefined })

  // Flatten pages and drop issues already linked to this project.
  const candidates = useMemo(() => {
    const all =
      issuesQuery.data?.pages.flatMap((page) => page.data ?? []) ?? []
    return all.filter((issue) => issue.project_id !== projectID)
  }, [issuesQuery.data, projectID])

  const handleLink = (issueID: string) => {
    setPendingId(issueID)
    updateIssue.mutate(
      {
        params: { path: { workspaceID, issueID } },
        body: { project_id: projectID },
      },
      {
        onSuccess: () => {
          toast.success('Issue linked')
          // Linking auto-adds the issue's team to the project, so refresh it.
          queryClient.invalidateQueries({
            queryKey: projectQueryKey(workspaceID, projectID),
          })
          queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
        },
        onError: (err) => toast.error(errorMessage(err)),
        onSettled: () => setPendingId(null),
      },
    )
  }

  return (
    <div className="space-y-3">
      <Input
        placeholder="Search issues…"
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        autoFocus
      />

      {issuesQuery.isError ? (
        <ErrorState
          error={issuesQuery.error}
          onRetry={() => issuesQuery.refetch()}
        />
      ) : (
        <ScrollArea className="h-72 rounded-md border">
          {issuesQuery.isLoading ? (
            <p className="p-4 text-sm text-muted-foreground">Loading…</p>
          ) : candidates.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">
              No issues to link.
            </p>
          ) : (
            <ul className="divide-y">
              {candidates.map((issue) => {
                const issueID = issue.id ?? ''
                const busy = pendingId === issueID
                return (
                  <li
                    key={issueID}
                    className="flex items-center gap-3 px-3 py-2"
                  >
                    <span className="w-20 shrink-0 truncate font-mono text-xs text-muted-foreground">
                      {issue.identifier}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-sm">
                      {issue.title}
                    </span>
                    {issue.project_id ? (
                      <span className="shrink-0 text-xs text-muted-foreground">
                        in another project
                      </span>
                    ) : null}
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="shrink-0"
                      disabled={busy || !issueID}
                      onClick={() => handleLink(issueID)}
                    >
                      <Plus className="h-3.5 w-3.5" />
                      {busy ? 'Linking…' : 'Link'}
                    </Button>
                  </li>
                )
              })}
            </ul>
          )}
        </ScrollArea>
      )}
    </div>
  )
}

interface LinkIssueDialogProps {
  workspaceID: string
  projectID: string
}

// "Link issue" dialog for a project. Attaches an existing issue by setting its
// `project_id` (PATCH). The picker query only runs while the dialog is open.
export function LinkIssueDialog({
  workspaceID,
  projectID,
}: LinkIssueDialogProps) {
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Link2 className="h-4 w-4" />
          Link issue
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Link an issue</DialogTitle>
          <DialogDescription>
            Search for an existing issue to add to this project. Its team is added
            to the project automatically.
          </DialogDescription>
        </DialogHeader>
        {open ? (
          <LinkIssuePicker workspaceID={workspaceID} projectID={projectID} />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

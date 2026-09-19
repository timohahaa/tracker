import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'

import { ErrorState } from '@/components/common/error-state'
import { Skeleton } from '@/components/ui/skeleton'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { IssueDetail } from '@/features/issue/issue-detail'
import { useIssue } from '@/features/issue/use-issue'

// Issue detail page, keyed by the identifier (e.g. "ENG-123") or UUID in the
// URL. Renders the full editable view; create/edit semantics live in
// <IssueDetail>. Not-found (404) surfaces as a friendly ErrorState.
export default function IssueDetailPage() {
  const { identifier } = useParams<{ identifier: string }>()
  const issueID = identifier ?? ''
  const { id: workspaceID, slug } = useCurrentWorkspace()

  const issueQuery = useIssue(workspaceID, issueID)

  return (
    <div className="mx-auto w-full max-w-5xl px-6 py-8">
      <Link
        to={`/${slug}/issues`}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to issues
      </Link>

      {issueQuery.isLoading ? (
        <div className="grid gap-8 lg:grid-cols-[1fr_260px]">
          <div className="space-y-6">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-8 w-2/3" />
            <Skeleton className="h-40 w-full" />
          </div>
          <div className="space-y-4">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
          </div>
        </div>
      ) : issueQuery.isError || !issueQuery.data ? (
        <ErrorState
          error={issueQuery.error}
          onRetry={() => issueQuery.refetch()}
        />
      ) : (
        <IssueDetail issueID={issueID} issue={issueQuery.data} />
      )}
    </div>
  )
}

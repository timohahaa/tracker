import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

import { type Issue,ISSUES_QUERY_KEY } from './use-issues'

export type CreateIssueBody =
  components['schemas']['internal_handler_issue.createIssueRequest']
export type UpdateIssueBody =
  components['schemas']['internal_handler_issue.updateIssueRequest']

// Shared query-key prefix for the single-issue endpoint. openapi-react-query keys
// are `['get', path, init]`, so this `['get', path]` prefix matches every
// issueID/params variant for invalidation.
export const ISSUE_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/issues/{issueID}',
] as const

// Exact query key for one issue, matching what `useIssue` builds (init is the
// params object). Needed for `setQueryData` (optimistic updates / cache writes)
// where a prefix won't do.
export function issueQueryKey(workspaceID: string, issueID: string) {
  return [
    'get',
    '/api/workspaces/{workspaceID}/issues/{issueID}',
    { params: { path: { workspaceID, issueID } } },
  ] as const
}

// GET one issue by UUID or identifier (e.g. "ENG-123"). The detail route keys
// off the identifier, so `issueID` here is usually the identifier. Access: any
// workspace member.
export function useIssue(workspaceID: string, issueID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/issues/{issueID}',
    { params: { path: { workspaceID, issueID } } },
    { enabled: !!workspaceID && !!issueID },
  )
}

// POST create an issue. `status_id` defaults to the team's backlog status when
// omitted. Access: team member (403 otherwise). Errors: 400 validation, 403.
export function useCreateIssue() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/workspaces/{workspaceID}/issues', {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY }),
  })
}

// PATCH update an issue (partial). Writes the returned issue straight into the
// single-issue cache so the detail view reflects the change without a refetch,
// then invalidates the list. Access: team member. Errors: 400/403/404.
export function useUpdateIssue() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/issues/{issueID}',
    {
      onSuccess: (data, variables) => {
        const { workspaceID, issueID } = variables.params.path
        queryClient.setQueryData(issueQueryKey(workspaceID, issueID), data)
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
      },
    },
  )
}

// DELETE (soft) an issue. Drops it from the single-issue cache and refreshes the
// list. Access: team member. Errors: 403/404.
export function useDeleteIssue() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/issues/{issueID}',
    {
      onSuccess: (_data, variables) => {
        const { workspaceID, issueID } = variables.params.path
        queryClient.removeQueries({
          queryKey: issueQueryKey(workspaceID, issueID),
        })
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
      },
    },
  )
}

// DELETE the issue's project link (project_id -> null). Idempotent, returns the
// updated issue. PATCH can't clear a nullable FK (empty string is rejected,
// null/omitted means "don't touch"), so clearing the project goes through here.
export function useUnlinkIssueProject() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/issues/{issueID}/project',
    {
      onSuccess: (data, variables) => {
        const { workspaceID, issueID } = variables.params.path
        queryClient.setQueryData(
          issueQueryKey(workspaceID, issueID),
          data as Issue,
        )
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
      },
    },
  )
}

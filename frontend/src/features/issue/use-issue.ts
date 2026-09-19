import { type QueryClient,useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

import { type Issue,ISSUES_QUERY_KEY } from './use-issues'

export type CreateIssueBody =
  components['schemas']['internal_handler_issue.createIssueRequest']
export type UpdateIssueBody =
  components['schemas']['internal_handler_issue.updateIssueRequest']
export type AddAssigneeBody =
  components['schemas']['internal_handler_issue.addAssigneeRequest']

// One entry in an issue's assignee set (`{ user_id, is_primary }`).
export type Assignee = NonNullable<Issue['assignees']>[number]

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

// --- Assignee point operations -------------------------------------------
//
// Single assignee edits go through the dedicated sub-resource endpoints (per
// api.md): POST adds, DELETE removes, PUT .../primary sets the responsible.
// Bulk replace of the whole set (e.g. clearing the responsible while keeping
// everyone assigned) is done via `useUpdateIssue` with `assignees`. All three
// hooks below share the canonical optimistic pattern: cancel → snapshot →
// optimistic write → rollback onError → settle/invalidate.

// Snapshot handed from onMutate to onError so a failed mutation can restore the
// pre-mutation cache.
interface AssigneeOptimisticContext {
  key: ReturnType<typeof issueQueryKey>
  previous: Issue | undefined
}

// Cancel in-flight refetches, snapshot the cached issue, then apply `transform`
// to its assignee set. Returns the snapshot for rollback.
async function optimisticAssignees(
  queryClient: QueryClient,
  workspaceID: string,
  issueID: string,
  transform: (assignees: Assignee[]) => Assignee[],
): Promise<AssigneeOptimisticContext> {
  const key = issueQueryKey(workspaceID, issueID)
  await queryClient.cancelQueries({ queryKey: key })
  const previous = queryClient.getQueryData<Issue>(key)
  queryClient.setQueryData<Issue>(key, (old) =>
    old ? { ...old, assignees: transform(old.assignees ?? []) } : old,
  )
  return { key, previous }
}

// POST an assignee. `is_primary=true` also makes them responsible (backend
// clears any prior primary). Returns the updated issue. Access: team member.
// Errors: 400/403/404.
export function useAddAssignee() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'post',
    '/api/workspaces/{workspaceID}/issues/{issueID}/assignees',
    {
      onMutate: (variables) => {
        const { workspaceID, issueID } = variables.params.path
        const body = variables.body as AddAssigneeBody
        return optimisticAssignees(
          queryClient,
          workspaceID,
          issueID,
          (assignees) => {
            const others = assignees.filter((a) => a.user_id !== body.user_id)
            const rest = body.is_primary
              ? others.map((a) => ({ ...a, is_primary: false }))
              : others
            return [
              ...rest,
              { user_id: body.user_id, is_primary: body.is_primary ?? false },
            ]
          },
        )
      },
      onError: (_err, _variables, context) => {
        if (context) queryClient.setQueryData(context.key, context.previous)
      },
      onSuccess: (data, variables) => {
        const { workspaceID, issueID } = variables.params.path
        queryClient.setQueryData(issueQueryKey(workspaceID, issueID), data)
      },
      onSettled: () =>
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY }),
    },
  )
}

// DELETE an assignee (204). Removing the responsible leaves the issue with none.
// Access: team member. Errors: 403/404.
export function useRemoveAssignee() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/issues/{issueID}/assignees/{userID}',
    {
      onMutate: (variables) => {
        const { workspaceID, issueID, userID } = variables.params.path
        return optimisticAssignees(
          queryClient,
          workspaceID,
          issueID,
          (assignees) => assignees.filter((a) => a.user_id !== userID),
        )
      },
      onError: (_err, _variables, context) => {
        if (context) queryClient.setQueryData(context.key, context.previous)
      },
      onSettled: (_data, _err, variables) => {
        const { workspaceID, issueID } = variables.params.path
        // 204 carries no body — refetch the single issue to reconcile.
        queryClient.invalidateQueries({
          queryKey: issueQueryKey(workspaceID, issueID),
        })
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
      },
    },
  )
}

// PUT the responsible assignee (only among existing assignees). Returns the
// updated issue. Access: team member. Errors: 400/403/404.
export function useSetPrimaryAssignee() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'put',
    '/api/workspaces/{workspaceID}/issues/{issueID}/assignees/{userID}/primary',
    {
      onMutate: (variables) => {
        const { workspaceID, issueID, userID } = variables.params.path
        return optimisticAssignees(
          queryClient,
          workspaceID,
          issueID,
          (assignees) =>
            assignees.map((a) => ({ ...a, is_primary: a.user_id === userID })),
        )
      },
      onError: (_err, _variables, context) => {
        if (context) queryClient.setQueryData(context.key, context.previous)
      },
      onSuccess: (data, variables) => {
        const { workspaceID, issueID } = variables.params.path
        queryClient.setQueryData(issueQueryKey(workspaceID, issueID), data)
      },
      onSettled: () =>
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY }),
    },
  )
}

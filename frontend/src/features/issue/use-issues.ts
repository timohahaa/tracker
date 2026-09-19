import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Issue = components['schemas']['internal_handler_issue.issueResponse']

// Client-side filter model. Mirrors the GET /issues query params (see api.md).
// All fields optional; empty ones are dropped before hitting the wire and the
// query key.
export interface IssueFilters {
  q?: string
  team?: string
  status?: string
  status_group?: string
  assignee?: string
  responsible?: string
  priority?: string
  project?: string
  label?: string[]
  due_date_before?: string
  due_date_after?: string
  sort?: string
  order?: string
}

// Shared query-key prefix. openapi-react-query keys are `['get', path, init]`,
// so this prefix matches every filter/cursor variant for invalidation.
export const ISSUES_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/issues',
] as const

const PAGE_SIZE = 50

// Build the query object. Undefined entries are ignored by both openapi-fetch
// (never serialized) and React Query's key hashing (JSON.stringify drops them),
// so changing filters yields a fresh query that restarts at the first page.
function toQuery(filters: IssueFilters) {
  return {
    limit: PAGE_SIZE,
    q: filters.q || undefined,
    team: filters.team || undefined,
    status: filters.status || undefined,
    status_group: filters.status_group || undefined,
    assignee: filters.assignee || undefined,
    responsible: filters.responsible || undefined,
    priority: filters.priority || undefined,
    project: filters.project || undefined,
    label: filters.label && filters.label.length > 0 ? filters.label : undefined,
    due_date_before: filters.due_date_before || undefined,
    due_date_after: filters.due_date_after || undefined,
    sort: filters.sort || undefined,
    order: filters.order || undefined,
  }
}

// Cursor-paginated issues list. First page sends no cursor: `initialPageParam`
// is `null` (not `undefined`, which openapi-react-query would coerce to `0`),
// and openapi-fetch omits null query params. Subsequent pages use
// `pagination.next_cursor` while `has_more` is true. Access: any workspace
// member.
export function useIssues(workspaceID: string, filters: IssueFilters) {
  return $api.useInfiniteQuery(
    'get',
    '/api/workspaces/{workspaceID}/issues',
    { params: { path: { workspaceID }, query: toQuery(filters) } },
    {
      getNextPageParam: (lastPage) =>
        lastPage.pagination?.has_more && lastPage.pagination.next_cursor
          ? lastPage.pagination.next_cursor
          : undefined,
      initialPageParam: null,
      pageParamName: 'cursor',
    },
  )
}

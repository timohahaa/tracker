import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

import type { IssueFilters } from './use-issues'

// Single-valued filter keys stored 1:1 in the URL query string. `label` is
// handled separately (repeatable).
const SINGLE_KEYS = [
  'q',
  'team',
  'status',
  'status_group',
  'assignee',
  'responsible',
  'priority',
  'project',
  'due_date_before',
  'due_date_after',
  'sort',
  'order',
] as const

export type SingleFilterKey = (typeof SINGLE_KEYS)[number]

// Reflects the issue filters in the URL search params so a filtered view is
// shareable and survives back/forward. Changing any filter replaces the current
// history entry (no stack spam) and, because the filters feed the query key,
// resets cursor pagination to the first page.
export function useIssueFilters() {
  const [searchParams, setSearchParams] = useSearchParams()

  const filters = useMemo<IssueFilters>(() => {
    const next: IssueFilters = {}
    for (const key of SINGLE_KEYS) {
      const value = searchParams.get(key)
      if (value) next[key] = value
    }
    const labels = searchParams.getAll('label')
    if (labels.length > 0) next.label = labels
    return next
  }, [searchParams])

  const setFilter = useCallback(
    (key: SingleFilterKey, value: string | undefined) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (!value) next.delete(key)
          else next.set(key, value)
          return next
        },
        { replace: true },
      )
    },
    [setSearchParams],
  )

  const setLabels = useCallback(
    (labels: string[]) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          next.delete('label')
          for (const label of labels) next.append('label', label)
          return next
        },
        { replace: true },
      )
    },
    [setSearchParams],
  )

  const clearAll = useCallback(() => {
    setSearchParams(new URLSearchParams(), { replace: true })
  }, [setSearchParams])

  const hasActiveFilters = useMemo(
    () =>
      SINGLE_KEYS.some((key) => !!searchParams.get(key)) ||
      searchParams.getAll('label').length > 0,
    [searchParams],
  )

  return { filters, setFilter, setLabels, clearAll, hasActiveFilters }
}

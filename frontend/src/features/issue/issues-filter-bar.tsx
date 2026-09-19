import { ArrowDownWideNarrow, ArrowUpNarrowWide, Check, Rows3, Tag, X } from 'lucide-react'
import { useEffect, useState } from 'react'

import { LabelDot } from '@/components/label-dot'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Label } from '@/features/label/use-labels'
import type { Project } from '@/features/project/use-projects-list'
import { useTeamStatuses } from '@/features/team/use-team-statuses'
import type { Team } from '@/features/team/use-teams'
import type { Member } from '@/features/workspace/use-members'
import { PRIORITIES, PRIORITY_ORDER } from '@/lib/priority'
import { STATUS_GROUP_ORDER, STATUS_GROUPS } from '@/lib/status'
import { cn } from '@/lib/utils'

import type { SingleFilterKey } from './use-issue-filters'
import type { IssueFilters } from './use-issues'

// Sentinel for the "any / all" option, since Radix Select forbids empty-string
// item values and real ids never collide with it.
const ALL = '__all__'

interface Option {
  value: string
  label: string
}

function FilterSelect({
  value,
  placeholder,
  options,
  onChange,
  className,
}: {
  value: string | undefined
  placeholder: string
  options: Option[]
  onChange: (value: string | undefined) => void
  className?: string
}) {
  return (
    <Select
      value={value ?? ALL}
      onValueChange={(next) => onChange(next === ALL ? undefined : next)}
    >
      <SelectTrigger className={cn('h-8 w-auto min-w-[8rem]', className)}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>{placeholder}</SelectItem>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

// Debounced free-text search bound to `q`.
function SearchInput({
  value,
  onChange,
}: {
  value: string | undefined
  onChange: (value: string | undefined) => void
}) {
  const [text, setText] = useState(value ?? '')

  // Sync the input when `q` changes elsewhere (e.g. Clear filters).
  useEffect(() => {
    setText(value ?? '')
  }, [value])

  // Push the debounced value into the URL. The guard stops the sync above from
  // re-triggering a redundant write.
  useEffect(() => {
    if ((value ?? '') === text) return
    const handle = setTimeout(() => onChange(text.trim() || undefined), 350)
    return () => clearTimeout(handle)
    // `onChange` is stable (useCallback); intentionally not resyncing on `value`.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text])

  return (
    <Input
      type="search"
      placeholder="Search issues…"
      value={text}
      onChange={(event) => setText(event.target.value)}
      className="h-8 w-full sm:w-56"
    />
  )
}

function LabelFilter({
  labels,
  selected,
  onChange,
}: {
  labels: Label[]
  selected: string[]
  onChange: (labels: string[]) => void
}) {
  const toggle = (id: string) => {
    onChange(
      selected.includes(id)
        ? selected.filter((value) => value !== id)
        : [...selected, id],
    )
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8">
          <Tag className="h-3.5 w-3.5" />
          Labels
          {selected.length > 0 ? (
            <span className="ml-1 rounded bg-primary/15 px-1 text-xs text-primary">
              {selected.length}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-0" align="start">
        <Command>
          <CommandInput placeholder="Filter labels…" />
          <CommandList>
            <CommandEmpty>No labels.</CommandEmpty>
            <CommandGroup>
              {labels.map((label) => {
                const id = label.id ?? ''
                const active = selected.includes(id)
                return (
                  <CommandItem
                    key={id}
                    value={label.name ?? id}
                    onSelect={() => toggle(id)}
                  >
                    <Check
                      className={cn(
                        'h-4 w-4',
                        active ? 'opacity-100' : 'opacity-0',
                      )}
                    />
                    <LabelDot color={label.color ?? '#8a8f98'} />
                    <span className="truncate">{label.name}</span>
                  </CommandItem>
                )
              })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

interface IssuesFilterBarProps {
  workspaceID: string
  filters: IssueFilters
  setFilter: (key: SingleFilterKey, value: string | undefined) => void
  setLabels: (labels: string[]) => void
  clearAll: () => void
  hasActiveFilters: boolean
  teams: Team[]
  members: Member[]
  labels: Label[]
  projects: Project[]
  grouped: boolean
  onToggleGrouped: (grouped: boolean) => void
}

export function IssuesFilterBar({
  workspaceID,
  filters,
  setFilter,
  setLabels,
  clearAll,
  hasActiveFilters,
  teams,
  members,
  labels,
  projects,
  grouped,
  onToggleGrouped,
}: IssuesFilterBarProps) {
  // The per-status filter is only meaningful within a single team, so it uses
  // the selected team's statuses (falls back to none until a team is chosen).
  const statusesQuery = useTeamStatuses(workspaceID, filters.team ?? '')
  const statuses = filters.team ? (statusesQuery.data?.data ?? []) : []

  const memberOptions: Option[] = members.map((member) => ({
    value: member.user_id ?? '',
    label: member.display_name?.trim() || member.email || 'Member',
  }))

  const order = filters.order === 'asc' ? 'asc' : 'desc'

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput
          value={filters.q}
          onChange={(value) => setFilter('q', value)}
        />

        <FilterSelect
          value={filters.team}
          placeholder="Team"
          onChange={(value) => {
            setFilter('team', value)
            // Team-scoped status no longer applies once the team changes.
            if (filters.status) setFilter('status', undefined)
          }}
          options={teams.map((team) => ({
            value: team.id ?? '',
            label: team.name ?? team.key ?? 'Team',
          }))}
        />

        <FilterSelect
          value={filters.status_group}
          placeholder="Status group"
          onChange={(value) => setFilter('status_group', value)}
          options={STATUS_GROUP_ORDER.map((group) => ({
            value: group,
            label: STATUS_GROUPS[group].label,
          }))}
        />

        {filters.team ? (
          <FilterSelect
            value={filters.status}
            placeholder="Status"
            onChange={(value) => setFilter('status', value)}
            options={statuses.map((status) => ({
              value: status.id ?? '',
              label: status.name ?? '',
            }))}
          />
        ) : null}

        <FilterSelect
          value={filters.priority}
          placeholder="Priority"
          onChange={(value) => setFilter('priority', value)}
          options={PRIORITY_ORDER.map((priority) => ({
            value: priority,
            label: PRIORITIES[priority].label,
          }))}
        />

        <FilterSelect
          value={filters.assignee}
          placeholder="Assignee"
          onChange={(value) => setFilter('assignee', value)}
          options={memberOptions}
        />

        <FilterSelect
          value={filters.responsible}
          placeholder="Responsible"
          onChange={(value) => setFilter('responsible', value)}
          options={memberOptions}
        />

        <FilterSelect
          value={filters.project}
          placeholder="Project"
          onChange={(value) => setFilter('project', value)}
          options={projects.map((project) => ({
            value: project.id ?? '',
            label: project.name ?? 'Project',
          }))}
        />

        <LabelFilter
          labels={labels}
          selected={filters.label ?? []}
          onChange={setLabels}
        />

        {hasActiveFilters ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-muted-foreground"
            onClick={clearAll}
          >
            <X className="h-3.5 w-3.5" />
            Clear
          </Button>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-muted-foreground">Due</span>
        <Input
          type="date"
          aria-label="Due after"
          value={filters.due_date_after ?? ''}
          onChange={(event) =>
            setFilter('due_date_after', event.target.value || undefined)
          }
          className="h-8 w-auto"
        />
        <span className="text-xs text-muted-foreground">→</span>
        <Input
          type="date"
          aria-label="Due before"
          value={filters.due_date_before ?? ''}
          onChange={(event) =>
            setFilter('due_date_before', event.target.value || undefined)
          }
          className="h-8 w-auto"
        />

        <div className="ml-auto flex items-center gap-2">
          <FilterSelect
            value={filters.sort}
            placeholder="Updated"
            onChange={(value) => setFilter('sort', value)}
            options={[
              { value: 'updated_at', label: 'Updated' },
              { value: 'created_at', label: 'Created' },
              { value: 'priority', label: 'Priority' },
            ]}
            className="min-w-[6.5rem]"
          />
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            aria-label={order === 'asc' ? 'Ascending' : 'Descending'}
            onClick={() => setFilter('order', order === 'asc' ? 'desc' : 'asc')}
          >
            {order === 'asc' ? (
              <ArrowUpNarrowWide className="h-4 w-4" />
            ) : (
              <ArrowDownWideNarrow className="h-4 w-4" />
            )}
          </Button>
          <Button
            variant={grouped ? 'secondary' : 'outline'}
            size="sm"
            className="h-8"
            aria-pressed={grouped}
            onClick={() => onToggleGrouped(!grouped)}
          >
            <Rows3 className="h-3.5 w-3.5" />
            Group
          </Button>
        </div>
      </div>
    </div>
  )
}

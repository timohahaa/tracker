import { Check, ChevronsUpDown } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import type { Team } from '@/features/team/use-teams'
import { cn } from '@/lib/utils'

interface TeamsMultiSelectProps {
  teams: Team[]
  // Selected team IDs (replace-semantics on the wire).
  value: string[]
  onChange: (next: string[]) => void
  disabled?: boolean
}

// Multi-select for a project's teams (Popover + Command with checkmarks). The
// trigger shows the selected teams' keys, or a placeholder. `team_ids` is sent as
// a full replace on POST/PATCH.
export function TeamsMultiSelect({
  teams,
  value,
  onChange,
  disabled,
}: TeamsMultiSelectProps) {
  const selected = new Set(value)
  const selectedTeams = teams.filter((t) => t.id && selected.has(t.id))

  const toggle = (id: string) => {
    if (selected.has(id)) onChange(value.filter((v) => v !== id))
    else onChange([...value, id])
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          disabled={disabled}
          className="w-full justify-between font-normal"
        >
          {selectedTeams.length > 0 ? (
            <span className="flex flex-wrap gap-1">
              {selectedTeams.map((team) => (
                <Badge key={team.id} variant="secondary" className="font-mono">
                  {team.key}
                </Badge>
              ))}
            </span>
          ) : (
            <span className="text-muted-foreground">Select teams</span>
          )}
          <ChevronsUpDown className="h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search teams…" />
          <CommandList>
            <CommandEmpty>No teams found.</CommandEmpty>
            <CommandGroup>
              {teams.map((team) => {
                const id = team.id ?? ''
                if (!id) return null
                const isSelected = selected.has(id)
                return (
                  <CommandItem
                    key={id}
                    value={`${team.name ?? ''} ${team.key ?? ''} ${id}`}
                    onSelect={() => toggle(id)}
                    className="gap-2"
                  >
                    <Badge variant="secondary" className="font-mono">
                      {team.key}
                    </Badge>
                    <span className="min-w-0 flex-1 truncate">{team.name}</span>
                    <Check
                      className={cn(
                        'h-4 w-4 shrink-0',
                        isSelected ? 'opacity-100' : 'opacity-0',
                      )}
                    />
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

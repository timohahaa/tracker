import { Check, ChevronsUpDown } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  useSetCurrentWorkspace,
  useWorkspaces,
} from '@/features/auth/session-provider'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { cn } from '@/lib/utils'

interface WorkspaceSwitcherProps {
  // Invoked after a workspace is picked, so the mobile drawer can close.
  onSelect?: () => void
}

// Workspace picker (Phase 1.4). Lists the user's workspaces, marks the active
// one, and on selection persists the choice server-side (PATCH /api/auth/me)
// and navigates to the new slug. The list always has >=1 entry post-register.
export function WorkspaceSwitcher({ onSelect }: WorkspaceSwitcherProps) {
  const workspaces = useWorkspaces()
  const current = useCurrentWorkspace()
  const navigate = useNavigate()
  const setCurrentWorkspace = useSetCurrentWorkspace()

  const currentWorkspace = workspaces.find((ws) => ws.id === current.id)
  const currentName = currentWorkspace?.name ?? current.slug

  function handleSelect(id: string | undefined, slug: string | undefined) {
    onSelect?.()
    if (!id || !slug || id === current.id) return
    setCurrentWorkspace.mutate({ body: { current_workspace_id: id } })
    navigate(`/${slug}`)
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="w-full justify-between gap-2 px-2 font-medium"
          aria-label="Switch workspace"
        >
          <span className="truncate">{currentName}</span>
          <ChevronsUpDown className="h-4 w-4 shrink-0 opacity-60" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {workspaces.map((ws) => {
          const isActive = ws.id === current.id
          return (
            <DropdownMenuItem
              key={ws.id}
              onSelect={() => handleSelect(ws.id, ws.slug)}
              className="gap-2"
            >
              <Check
                className={cn(
                  'h-4 w-4 shrink-0',
                  isActive ? 'opacity-100' : 'opacity-0',
                )}
              />
              <span className="truncate">{ws.name ?? ws.slug}</span>
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

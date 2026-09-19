import { LogOut, Menu, Search } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { ThemeToggle } from '@/components/theme-toggle'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useCurrentUser } from '@/features/auth/session-provider'
import { useLogout } from '@/features/auth/use-logout'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { useCommands } from '@/features/command/command-provider'

// Derives up-to-two-letter initials from a display name, falling back to the
// email local part, then a generic glyph.
function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

interface TopbarProps {
  // Opens the mobile navigation drawer (only rendered on narrow screens).
  onOpenMobileNav: () => void
}

export function Topbar({ onOpenMobileNav }: TopbarProps) {
  const { slug } = useCurrentWorkspace()
  const user = useCurrentUser()
  const navigate = useNavigate()
  const logout = useLogout()
  const { openPalette } = useCommands()

  const displayName = user?.display_name || user?.email || 'Account'
  const email = user?.email ?? ''

  const handleLogout = () => {
    logout.mutate(
      {},
      { onSuccess: () => navigate('/login', { replace: true }) },
    )
  }

  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b bg-background px-4">
      <Button
        variant="ghost"
        size="icon"
        className="md:hidden"
        onClick={onOpenMobileNav}
        aria-label="Open navigation menu"
      >
        <Menu className="h-5 w-5" />
      </Button>

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{slug}</p>
      </div>

      <button
        type="button"
        onClick={openPalette}
        className="hidden items-center gap-2 rounded-md border bg-background px-2.5 py-1.5 text-sm text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-1 focus-visible:ring-ring sm:flex"
        aria-label="Open command palette"
      >
        <Search className="h-4 w-4" />
        <span>Search…</span>
        <kbd className="ml-2 rounded border bg-muted px-1.5 py-0.5 font-mono text-xs">
          ⌘K
        </kbd>
      </button>

      <Button
        variant="ghost"
        size="icon"
        className="sm:hidden"
        onClick={openPalette}
        aria-label="Open command palette"
      >
        <Search className="h-5 w-5" />
      </Button>

      <ThemeToggle />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="flex items-center gap-2 rounded-md px-1 py-1 outline-none transition-colors hover:bg-accent focus-visible:ring-1 focus-visible:ring-ring"
            aria-label="Open account menu"
          >
            <div className="hidden text-right leading-tight sm:block">
              <p className="truncate text-sm font-medium">{displayName}</p>
              {email && (
                <p className="truncate text-xs text-muted-foreground">{email}</p>
              )}
            </div>
            <Avatar className="h-8 w-8">
              <AvatarFallback className="text-xs">
                {initials(displayName)}
              </AvatarFallback>
            </Avatar>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuLabel className="truncate font-normal">
            {email || displayName}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onSelect={handleLogout}
            disabled={logout.isPending}
          >
            <LogOut className="h-4 w-4" />
            <span>{logout.isPending ? 'Logging out…' : 'Log out'}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  )
}

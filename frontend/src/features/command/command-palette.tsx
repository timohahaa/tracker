import {
  FolderKanban,
  Home,
  KeyRound,
  ListTodo,
  LogOut,
  Moon,
  Plug,
  Plus,
  Search,
  Settings,
  Sun,
  Tag,
  Users,
} from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { $api } from '@/api/client'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import {
  useSetCurrentWorkspace,
  useWorkspaces,
} from '@/features/auth/session-provider'
import { useLogout } from '@/features/auth/use-logout'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { useProjectsList } from '@/features/project/use-projects-list'
import { useTeams } from '@/features/team/use-teams'
import { useDebouncedValue } from '@/hooks/use-debounced-value'
import { useTheme } from '@/hooks/use-theme'
import { toastError } from '@/lib/toast'

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  // Opens the "New issue" dialog (owned by the CommandProvider).
  onCreateIssue: () => void
}

// Global command palette (Cmd/Ctrl+K). Reuses the shadcn Command (cmdk)
// primitive. Server-driven issue search is combined with client-side matching of
// static navigation/actions, so `shouldFilter` is disabled and matching is done
// here. Mounted only while open (the provider conditionally renders it), so its
// data hooks stay idle until invoked.
export function CommandPalette({
  open,
  onOpenChange,
  onCreateIssue,
}: CommandPaletteProps) {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const { resolvedTheme, toggleTheme } = useTheme()
  const logout = useLogout()
  const workspaces = useWorkspaces()
  const setCurrentWorkspace = useSetCurrentWorkspace()

  const [search, setSearch] = useState('')
  const debouncedSearch = useDebouncedValue(search.trim(), 250)

  const teamsQuery = useTeams(workspaceID)
  const teams = teamsQuery.data?.data ?? []
  const projectsQuery = useProjectsList(workspaceID)
  const projects = projectsQuery.data?.data ?? []

  // Server-side issue search, gated on a non-empty query.
  const issuesQuery = $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/issues',
    {
      params: {
        path: { workspaceID },
        query: { q: debouncedSearch, limit: 8 },
      },
    },
    { enabled: open && debouncedSearch.length > 0 },
  )
  const issues = issuesQuery.data?.data ?? []

  const q = search.trim().toLowerCase()
  const matches = (...text: (string | undefined)[]) =>
    !q || text.some((t) => (t ?? '').toLowerCase().includes(q))

  // Closes the palette first, then runs the action on the next tick so the
  // dialog teardown (focus restore) doesn't race navigation / opening another
  // dialog.
  const runAction = (fn: () => void) => {
    onOpenChange(false)
    setTimeout(fn, 0)
  }

  const go = (to: string) => runAction(() => navigate(to))

  const navItems = [
    { label: 'Home', icon: Home, to: `/${slug}` },
    { label: 'Issues', icon: ListTodo, to: `/${slug}/issues` },
    { label: 'Projects', icon: FolderKanban, to: `/${slug}/projects` },
    { label: 'Teams', icon: Users, to: `/${slug}/teams` },
    { label: 'Settings', icon: Settings, to: `/${slug}/settings` },
    { label: 'Members', icon: Users, to: `/${slug}/settings/members` },
    { label: 'Labels', icon: Tag, to: `/${slug}/settings/labels` },
    {
      label: 'Integrations',
      icon: Plug,
      to: `/${slug}/settings/integrations`,
    },
    { label: 'API tokens', icon: KeyRound, to: `/${slug}/settings/tokens` },
  ]

  const showNewIssue = matches('New issue', 'create')
  const showThemeToggle = matches('Toggle theme', 'dark', 'light')
  const showLogout = matches('Log out', 'sign out')

  const visibleNav = navItems.filter((item) => matches(item.label))
  const visibleTeams = teams.filter((team) => matches(team.name, team.key))
  const visibleProjects = projects.filter((project) => matches(project.name))
  const otherWorkspaces = workspaces.filter((ws) => ws.id !== workspaceID)
  const visibleWorkspaces = otherWorkspaces.filter((ws) =>
    matches(ws.name, ws.slug),
  )

  const handleLogout = () =>
    runAction(() =>
      logout.mutate(
        {},
        {
          onSuccess: () => navigate('/login', { replace: true }),
          onError: (err) => toastError(err),
        },
      ),
    )

  const handleSwitchWorkspace = (id: string, wsSlug: string) =>
    runAction(() => {
      setCurrentWorkspace.mutate(
        { body: { current_workspace_id: id } },
        { onError: (err) => toastError(err) },
      )
      navigate(`/${wsSlug}`)
    })

  return (
    // shouldFilter off: matching is done here (see `matches`) so server issue
    // results are never hidden by cmdk's own fuzzy filter.
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      commandProps={{ shouldFilter: false }}
    >
      <CommandInput
        placeholder="Search or jump to…"
        value={search}
        onValueChange={setSearch}
      />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        {showNewIssue || showThemeToggle || showLogout ? (
          <CommandGroup heading="Actions">
            {showNewIssue ? (
              <CommandItem
                value="action-new-issue"
                onSelect={() => runAction(onCreateIssue)}
              >
                <Plus />
                New issue
              </CommandItem>
            ) : null}
            {showThemeToggle ? (
              <CommandItem
                value="action-toggle-theme"
                onSelect={() => runAction(toggleTheme)}
              >
                {resolvedTheme === 'dark' ? <Sun /> : <Moon />}
                Toggle theme
              </CommandItem>
            ) : null}
            {showLogout ? (
              <CommandItem value="action-logout" onSelect={handleLogout}>
                <LogOut />
                Log out
              </CommandItem>
            ) : null}
          </CommandGroup>
        ) : null}

        {visibleNav.length > 0 ? (
          <>
            <CommandSeparator />
            <CommandGroup heading="Go to">
              {visibleNav.map((item) => (
                <CommandItem
                  key={item.to}
                  value={`nav-${item.to}`}
                  onSelect={() => go(item.to)}
                >
                  <item.icon />
                  {item.label}
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        ) : null}

        {visibleWorkspaces.length > 0 ? (
          <>
            <CommandSeparator />
            <CommandGroup heading="Switch workspace">
              {visibleWorkspaces.map((ws) => (
                <CommandItem
                  key={ws.id}
                  value={`ws-${ws.id}`}
                  onSelect={() =>
                    ws.id && ws.slug
                      ? handleSwitchWorkspace(ws.id, ws.slug)
                      : undefined
                  }
                >
                  <Home />
                  {ws.name ?? ws.slug}
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        ) : null}

        {visibleTeams.length > 0 ? (
          <>
            <CommandSeparator />
            <CommandGroup heading="Teams">
              {visibleTeams.map((team) => (
                <CommandItem
                  key={team.id}
                  value={`team-${team.id}`}
                  onSelect={() => go(`/${slug}/teams/${team.id}`)}
                >
                  <Users />
                  <span className="truncate">{team.name || team.key}</span>
                  {team.key ? (
                    <span className="ml-auto font-mono text-xs text-muted-foreground">
                      {team.key}
                    </span>
                  ) : null}
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        ) : null}

        {visibleProjects.length > 0 ? (
          <>
            <CommandSeparator />
            <CommandGroup heading="Projects">
              {visibleProjects.map((project) => (
                <CommandItem
                  key={project.id}
                  value={`project-${project.id}`}
                  onSelect={() => go(`/${slug}/projects/${project.id}`)}
                >
                  <FolderKanban />
                  <span className="truncate">{project.name || 'Project'}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        ) : null}

        {debouncedSearch.length > 0 ? (
          <>
            <CommandSeparator />
            <CommandGroup heading="Issues">
              {issues.map((issue) => (
                <CommandItem
                  key={issue.id}
                  value={`issue-${issue.id}`}
                  onSelect={() =>
                    go(`/${slug}/issues/${issue.identifier || issue.id}`)
                  }
                >
                  <ListTodo />
                  <span className="truncate">{issue.title}</span>
                  {issue.identifier ? (
                    <span className="ml-auto font-mono text-xs text-muted-foreground">
                      {issue.identifier}
                    </span>
                  ) : null}
                </CommandItem>
              ))}
              <CommandItem
                value="issues-search-all"
                onSelect={() =>
                  go(
                    `/${slug}/issues?q=${encodeURIComponent(debouncedSearch)}`,
                  )
                }
              >
                <Search />
                Search all issues for “{debouncedSearch}”
              </CommandItem>
            </CommandGroup>
          </>
        ) : null}
      </CommandList>
    </CommandDialog>
  )
}

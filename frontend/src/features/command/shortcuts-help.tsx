import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface ShortcutsHelpProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// A single keycap.
function Kbd({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="inline-flex min-w-[1.5rem] items-center justify-center rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">
      {children}
    </kbd>
  )
}

// One row: the shortcut keys on the left, its description on the right.
function Shortcut({
  keys,
  label,
}: {
  keys: React.ReactNode[]
  label: string
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-1.5">
      <span className="text-sm">{label}</span>
      <span className="flex shrink-0 items-center gap-1">
        {keys.map((key, i) => (
          <span key={i} className="flex items-center gap-1">
            {i > 0 ? (
              <span className="text-xs text-muted-foreground">then</span>
            ) : null}
            {key}
          </span>
        ))}
      </span>
    </div>
  )
}

// Discoverable keyboard-shortcut reference, opened with `?`.
export function ShortcutsHelp({ open, onOpenChange }: ShortcutsHelpProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Keyboard shortcuts</DialogTitle>
          <DialogDescription>
            Shortcuts work anywhere except while typing in a field.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <section>
            <h3 className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              General
            </h3>
            <Shortcut keys={[<Kbd key="k">⌘ K</Kbd>]} label="Command palette" />
            <Shortcut keys={[<Kbd key="slash">/</Kbd>]} label="Search" />
            <Shortcut keys={[<Kbd key="c">C</Kbd>]} label="New issue" />
            <Shortcut
              keys={[<Kbd key="help">?</Kbd>]}
              label="Keyboard shortcuts"
            />
          </section>

          <section>
            <h3 className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Go to
            </h3>
            <Shortcut
              keys={[<Kbd key="g">G</Kbd>, <Kbd key="h">H</Kbd>]}
              label="Home"
            />
            <Shortcut
              keys={[<Kbd key="g">G</Kbd>, <Kbd key="i">I</Kbd>]}
              label="Issues"
            />
            <Shortcut
              keys={[<Kbd key="g">G</Kbd>, <Kbd key="p">P</Kbd>]}
              label="Projects"
            />
            <Shortcut
              keys={[<Kbd key="g">G</Kbd>, <Kbd key="t">T</Kbd>]}
              label="Teams"
            />
            <Shortcut
              keys={[<Kbd key="g">G</Kbd>, <Kbd key="s">S</Kbd>]}
              label="Settings"
            />
          </section>
        </div>
      </DialogContent>
    </Dialog>
  )
}

import { AlertTriangle } from 'lucide-react'

// Compact, form-wide error banner for auth pages (e.g. 401 bad credentials on
// login, 409 duplicate email/slug on register). Field-level errors are shown
// under the inputs via <FieldError /> instead.
export function AuthErrorBanner({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <p>{message}</p>
    </div>
  )
}

interface AuthShellProps {
  title: string
  // Short line under the title (e.g. "Sign in to your workspace").
  subtitle?: string
  // Optional top-level (form-wide) error banner, rendered above the form.
  banner?: React.ReactNode
  // The form.
  children: React.ReactNode
  // Footer line (e.g. the link to the other auth page).
  footer?: React.ReactNode
}

// Shared, centered auth layout in the dark/mono terminal theme. Keeps the login
// and register pages visually consistent (uses theme tokens only).
export function AuthShell({
  title,
  subtitle,
  banner,
  children,
  footer,
}: AuthShellProps) {
  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="rounded-lg border bg-card p-6 shadow-sm">
          <div className="mb-6 space-y-1.5">
            <p className="text-xs uppercase tracking-widest text-muted-foreground">
              linear-clone
            </p>
            <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
            {subtitle ? (
              <p className="text-sm text-muted-foreground">{subtitle}</p>
            ) : null}
          </div>

          {banner ? <div className="mb-4">{banner}</div> : null}

          {children}
        </div>

        {footer ? (
          <p className="mt-4 text-center text-sm text-muted-foreground">
            {footer}
          </p>
        ) : null}
      </div>
    </div>
  )
}

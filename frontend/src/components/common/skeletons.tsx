import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

// Suspense fallback for lazy routes. Kept intentionally minimal — the real
// per-page skeletons live with their pages once those arrive in Phase 2.
export function RouteFallback() {
  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6">
      <div className="w-full max-w-md space-y-4">
        <Skeleton className="h-8 w-1/2" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
      </div>
    </div>
  )
}

interface ListSkeletonProps {
  /** Number of placeholder rows to render. */
  rows?: number
  className?: string
}

// Composite skeleton for list-shaped content (issues, teams, members, ...).
export function ListSkeleton({ rows = 5, className }: ListSkeletonProps) {
  return (
    <div className={cn('space-y-2', className)}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 rounded-md border p-3">
          <Skeleton className="h-8 w-8 shrink-0 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  )
}

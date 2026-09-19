import { Loader2 } from 'lucide-react'
import { useEffect, useRef } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface InfiniteScrollProps {
  /** True when another page can be fetched (`useInfiniteQuery().hasNextPage`). */
  hasNextPage: boolean
  /** True while the next page is loading (`useInfiniteQuery().isFetchingNextPage`). */
  isFetchingNextPage: boolean
  /** `useInfiniteQuery().fetchNextPage`. */
  fetchNextPage: () => void
  /** IntersectionObserver root margin — fetch a bit before the sentinel is fully
   *  in view so scrolling feels seamless. */
  rootMargin?: string
  className?: string
}

// Shared cursor-pagination trigger for any `useInfiniteQuery`-backed list
// (issues today; projects and other paginated lists next). Renders an
// IntersectionObserver sentinel that auto-fetches the next page as it scrolls
// into view, with a "Load more" button fallback for non-scrolling containers
// and keyboard users. Renders nothing once the list is exhausted.
export function InfiniteScroll({
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
  rootMargin = '200px',
  className,
}: InfiniteScrollProps) {
  const sentinelRef = useRef<HTMLDivElement | null>(null)

  // Re-subscribing whenever hasNextPage/isFetchingNextPage change also re-fires
  // the callback if the sentinel is still on screen after a page loads (short
  // pages that never leave the viewport), so auto-scroll doesn't stall.
  useEffect(() => {
    const node = sentinelRef.current
    if (!node || !hasNextPage) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { rootMargin },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, rootMargin])

  if (!hasNextPage && !isFetchingNextPage) return null

  return (
    <div
      ref={sentinelRef}
      className={cn('flex justify-center py-4', className)}
    >
      {isFetchingNextPage ? (
        <span className="inline-flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          Loading more…
        </span>
      ) : hasNextPage ? (
        <Button variant="outline" size="sm" onClick={() => fetchNextPage()}>
          Load more
        </Button>
      ) : null}
    </div>
  )
}

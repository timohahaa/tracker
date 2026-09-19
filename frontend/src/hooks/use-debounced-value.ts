import { useEffect, useState } from 'react'

// Returns a copy of `value` that only updates after it has stayed unchanged for
// `delay` ms. Used to throttle server-side search (e.g. the command palette's
// issue lookup) so keystrokes don't fire a request each.
export function useDebouncedValue<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(timer)
  }, [value, delay])

  return debounced
}

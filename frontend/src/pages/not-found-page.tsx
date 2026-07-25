import { Link } from 'react-router-dom'

import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export default function NotFoundPage() {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-3 p-6 text-center">
      <h1 className="text-3xl font-semibold">404</h1>
      <p className="text-sm text-muted-foreground">
        This page could not be found.
      </p>
      <Link to="/login" className={cn(buttonVariants({ variant: 'outline' }))}>
        Back to login
      </Link>
    </div>
  )
}

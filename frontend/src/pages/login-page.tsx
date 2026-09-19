import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { RouteFallback } from '@/components/common/skeletons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useSession } from '@/features/auth/session-provider'
import { useLogin } from '@/features/auth/use-login'

import { AuthErrorBanner, AuthShell } from './auth-shell'

const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
})

type LoginValues = z.infer<typeof loginSchema>

// Only accept relative in-app paths as the post-login destination; reject
// absolute/protocol-relative URLs to avoid open-redirects.
function safeNext(raw: string | null): string {
  if (raw && raw.startsWith('/') && !raw.startsWith('//')) return raw
  return '/'
}

export default function LoginPage() {
  const { status, defaultWorkspace } = useSession()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const next = safeNext(searchParams.get('next'))

  const login = useLogin()
  const serverError = login.error instanceof AppError ? login.error : null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  })

  // Don't flash the form while the session is still resolving; redirect an
  // already-authenticated user away from /login.
  if (status === 'loading') return <RouteFallback />
  if (status === 'authenticated') {
    return (
      <Navigate to={defaultWorkspace?.slug ? `/${defaultWorkspace.slug}` : '/'} replace />
    )
  }

  const onSubmit = (values: LoginValues) => {
    login.mutate(
      { body: values },
      { onSuccess: () => navigate(next, { replace: true }) },
    )
  }

  // Field-level errors (400) render under inputs; anything else (401, network,
  // internal) surfaces as the top-level banner.
  const bannerMessage =
    serverError && (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  return (
    <AuthShell
      title="Log in"
      subtitle="Sign in to your workspace"
      banner={bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}
      footer={
        <>
          Don&apos;t have an account?{' '}
          <Link to="/register" className="text-foreground underline underline-offset-4">
            Create one
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <div className="space-y-1.5">
          <label htmlFor="email" className="text-sm font-medium">
            Email
          </label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            autoFocus
            aria-invalid={!!errors.email}
            {...register('email')}
          />
          {errors.email ? (
            <p className="text-sm text-destructive">{errors.email.message}</p>
          ) : (
            <FieldError error={login.error} name="email" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="password" className="text-sm font-medium">
            Password
          </label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            aria-invalid={!!errors.password}
            {...register('password')}
          />
          {errors.password ? (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          ) : (
            <FieldError error={login.error} name="password" />
          )}
        </div>

        <Button type="submit" className="w-full" disabled={login.isPending}>
          {login.isPending ? 'Signing in…' : 'Log in'}
        </Button>
      </form>
    </AuthShell>
  )
}

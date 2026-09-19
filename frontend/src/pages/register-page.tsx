import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { RouteFallback } from '@/components/common/skeletons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useSession } from '@/features/auth/session-provider'
import { useRegister } from '@/features/auth/use-register'

import { AuthErrorBanner, AuthShell } from './auth-shell'

const registerSchema = z.object({
  display_name: z.string().min(1, 'Name is required'),
  email: z.string().min(1, 'Email is required').email('Enter a valid email'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  workspace_name: z.string().min(1, 'Workspace name is required'),
})

type RegisterValues = z.infer<typeof registerSchema>

export default function RegisterPage() {
  const { status, defaultWorkspace } = useSession()
  const navigate = useNavigate()

  const registerMutation = useRegister()
  const serverError =
    registerMutation.error instanceof AppError ? registerMutation.error : null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      display_name: '',
      email: '',
      password: '',
      workspace_name: '',
    },
  })

  if (status === 'loading') return <RouteFallback />
  if (status === 'authenticated') {
    return (
      <Navigate to={defaultWorkspace?.slug ? `/${defaultWorkspace.slug}` : '/'} replace />
    )
  }

  const onSubmit = (values: RegisterValues) => {
    registerMutation.mutate(
      { body: values },
      {
        onSuccess: (data) => {
          const slug = data.workspace?.slug
          navigate(slug ? `/${slug}` : '/', { replace: true })
        },
      },
    )
  }

  // 400 field errors render under their inputs; 409 (already_exists) and other
  // non-field errors surface as the top-level banner.
  const bannerMessage =
    serverError && (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  return (
    <AuthShell
      title="Create account"
      subtitle="Set up your account and first workspace"
      banner={bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="text-foreground underline underline-offset-4">
            Log in
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <div className="space-y-1.5">
          <label htmlFor="display_name" className="text-sm font-medium">
            Name
          </label>
          <Input
            id="display_name"
            autoComplete="name"
            autoFocus
            aria-invalid={!!errors.display_name}
            {...register('display_name')}
          />
          {errors.display_name ? (
            <p className="text-sm text-destructive">
              {errors.display_name.message}
            </p>
          ) : (
            <FieldError error={registerMutation.error} name="display_name" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="email" className="text-sm font-medium">
            Email
          </label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            aria-invalid={!!errors.email}
            {...register('email')}
          />
          {errors.email ? (
            <p className="text-sm text-destructive">{errors.email.message}</p>
          ) : (
            <FieldError error={registerMutation.error} name="email" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="password" className="text-sm font-medium">
            Password
          </label>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            aria-invalid={!!errors.password}
            {...register('password')}
          />
          {errors.password ? (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          ) : (
            <FieldError error={registerMutation.error} name="password" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="workspace_name" className="text-sm font-medium">
            Workspace name
          </label>
          <Input
            id="workspace_name"
            autoComplete="organization"
            aria-invalid={!!errors.workspace_name}
            {...register('workspace_name')}
          />
          {errors.workspace_name ? (
            <p className="text-sm text-destructive">
              {errors.workspace_name.message}
            </p>
          ) : (
            <FieldError error={registerMutation.error} name="workspace_name" />
          )}
        </div>

        <Button
          type="submit"
          className="w-full"
          disabled={registerMutation.isPending}
        >
          {registerMutation.isPending ? 'Creating account…' : 'Create account'}
        </Button>
      </form>
    </AuthShell>
  )
}

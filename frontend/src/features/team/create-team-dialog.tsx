import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { useCreateTeam } from './use-teams'

// Client-side shape validation only. The backend is the source of truth for the
// key format (it uppercases/normalizes) and uniqueness, so we keep this lenient
// and surface server `fields[]` via <FieldError> and the 409 banner.
const createTeamSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  key: z.string().min(1, 'Key is required'),
  description: z.string(),
})

type CreateTeamValues = z.infer<typeof createTeamSchema>

// "New team" dialog. Available to any workspace member. On success navigates to
// the new team's detail page (its default statuses are created by the backend).
export function CreateTeamDialog() {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const createTeam = useCreateTeam()
  const [open, setOpen] = useState(false)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateTeamValues>({
    resolver: zodResolver(createTeamSchema),
    defaultValues: { name: '', key: '', description: '' },
  })

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) {
      reset()
      createTeam.reset()
    }
  }

  const onSubmit = (values: CreateTeamValues) => {
    createTeam.mutate(
      {
        params: { path: { workspaceID } },
        body: {
          name: values.name,
          key: values.key,
          description: values.description || undefined,
        },
      },
      {
        onSuccess: (data) => {
          toast.success('Team created')
          handleOpenChange(false)
          if (data.id) navigate(`/${slug}/teams/${data.id}`)
        },
      },
    )
  }

  const serverError =
    createTeam.error instanceof AppError ? createTeam.error : null
  // Non-field errors (e.g. 409 key taken) surface as a banner; 400 validation
  // with fields[] renders under the inputs.
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>New team</Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <DialogHeader>
            <DialogTitle>Create team</DialogTitle>
            <DialogDescription>
              Teams own their issues, statuses and members. A short key prefixes
              issue identifiers (e.g. <span className="font-mono">ENG-123</span>).
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

            <div className="space-y-1.5">
              <label htmlFor="team-name" className="text-sm font-medium">
                Name
              </label>
              <Input
                id="team-name"
                placeholder="Engineering"
                aria-invalid={!!errors.name}
                {...register('name')}
              />
              {errors.name ? (
                <p className="text-sm text-destructive">
                  {errors.name.message}
                </p>
              ) : (
                <FieldError error={createTeam.error} name="name" />
              )}
            </div>

            <div className="space-y-1.5">
              <label htmlFor="team-key" className="text-sm font-medium">
                Key
              </label>
              <Input
                id="team-key"
                placeholder="ENG"
                className="font-mono uppercase"
                aria-invalid={!!errors.key}
                {...register('key')}
              />
              <p className="text-xs text-muted-foreground">
                A short identifier used to prefix issues. Normalized to uppercase.
              </p>
              {errors.key ? (
                <p className="text-sm text-destructive">{errors.key.message}</p>
              ) : (
                <FieldError error={createTeam.error} name="key" />
              )}
            </div>

            <div className="space-y-1.5">
              <label htmlFor="team-description" className="text-sm font-medium">
                Description
                <span className="ml-1 font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="team-description"
                placeholder="What this team works on"
                {...register('description')}
              />
              <FieldError error={createTeam.error} name="description" />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={createTeam.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createTeam.isPending}>
              {createTeam.isPending ? 'Creating…' : 'Create team'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

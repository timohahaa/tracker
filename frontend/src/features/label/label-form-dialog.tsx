import { zodResolver } from '@hookform/resolvers/zod'
import { type ReactNode, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { LabelChip } from '@/components/label-dot'
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
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { isValidHex, LABEL_COLORS } from '@/lib/label-color'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { ColorPicker } from './color-picker'
import { type Label, useCreateLabel, useUpdateLabel } from './use-labels'

const FALLBACK_COLOR = '#8a8f98'

const labelSchema = z.object({
  name: z.string().trim().min(1, 'Name is required'),
  color: z.string().refine(isValidHex, 'Enter a valid hex color (e.g. #5e6ad2)'),
})

type LabelValues = z.infer<typeof labelSchema>

interface LabelFormDialogProps {
  trigger: ReactNode
  // Present → edit that label; absent → create a new one.
  label?: Label
  // Scope for new labels: a team id → team-scoped; omitted → workspace-level.
  // Ignored in edit mode (the API can't move a label between scopes).
  teamID?: string
}

// Create or edit a label (name + color). Reused for both workspace-level and
// team-scoped labels; scope is decided by `teamID` on create.
export function LabelFormDialog({ trigger, label, teamID }: LabelFormDialogProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const isEdit = !!label
  const createLabel = useCreateLabel()
  const updateLabel = useUpdateLabel()
  const [open, setOpen] = useState(false)

  const mutation = isEdit ? updateLabel : createLabel

  const {
    register,
    handleSubmit,
    control,
    watch,
    reset,
    formState: { errors },
  } = useForm<LabelValues>({
    resolver: zodResolver(labelSchema),
    defaultValues: {
      name: label?.name ?? '',
      color: label?.color ?? LABEL_COLORS[0],
    },
  })

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) {
      reset({
        name: label?.name ?? '',
        color: label?.color ?? LABEL_COLORS[0],
      })
      mutation.reset()
    }
  }

  const serverError = mutation.error instanceof AppError ? mutation.error : null
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  const onSubmit = (values: LabelValues) => {
    const name = values.name.trim()
    if (isEdit) {
      updateLabel.mutate(
        {
          params: { path: { workspaceID, labelID: label.id ?? '' } },
          body: { name, color: values.color },
        },
        {
          onSuccess: () => {
            toast.success('Label updated')
            setOpen(false)
          },
        },
      )
    } else {
      createLabel.mutate(
        {
          params: { path: { workspaceID } },
          body: {
            name,
            color: values.color,
            ...(teamID ? { team_id: teamID } : {}),
          },
        },
        {
          onSuccess: () => {
            toast.success('Label created')
            setOpen(false)
          },
        },
      )
    }
  }

  const previewName = watch('name')?.trim() || 'Label'
  const previewColor = watch('color')

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit label' : 'Create label'}</DialogTitle>
          <DialogDescription>
            {teamID
              ? 'A team-scoped label, available on that team’s issues.'
              : isEdit
                ? 'Update the label name and color.'
                : 'A workspace label, available across all teams.'}
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="space-y-4"
          noValidate
        >
          {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

          <div className="space-y-1.5">
            <label htmlFor="label-name" className="text-sm font-medium">
              Name
            </label>
            <Input
              id="label-name"
              aria-invalid={!!errors.name}
              {...register('name')}
            />
            {errors.name ? (
              <p className="text-sm text-destructive">{errors.name.message}</p>
            ) : (
              <FieldError error={mutation.error} name="name" />
            )}
          </div>

          <div className="space-y-1.5">
            <span className="text-sm font-medium">Color</span>
            <Controller
              control={control}
              name="color"
              render={({ field }) => (
                <ColorPicker value={field.value} onChange={field.onChange} />
              )}
            />
            {errors.color ? (
              <p className="text-sm text-destructive">{errors.color.message}</p>
            ) : (
              <FieldError error={mutation.error} name="color" />
            )}
          </div>

          <div className="space-y-1.5">
            <span className="text-sm font-medium">Preview</span>
            <div>
              <LabelChip
                name={previewName}
                color={isValidHex(previewColor) ? previewColor : FALLBACK_COLOR}
                filled
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending
                ? isEdit
                  ? 'Saving…'
                  : 'Creating…'
                : isEdit
                  ? 'Save'
                  : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

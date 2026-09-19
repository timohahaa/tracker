import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Member } from '@/features/workspace/use-members'

import { memberName } from './member-display'

// Radix Select forbids empty-string values, so the "no lead" choice uses this
// sentinel and maps to '' outside.
export const NO_LEAD = '__none__'

interface LeadSelectProps {
  members: Member[]
  // The selected lead's user id, or '' for none.
  value: string
  onChange: (next: string) => void
  disabled?: boolean
  id?: string
}

// Single-select of workspace members for a project lead, with a "No lead" option.
export function LeadSelect({
  members,
  value,
  onChange,
  disabled,
  id,
}: LeadSelectProps) {
  return (
    <Select
      value={value || NO_LEAD}
      onValueChange={(next) => onChange(next === NO_LEAD ? '' : next)}
      disabled={disabled}
    >
      <SelectTrigger id={id}>
        <SelectValue placeholder="No lead" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={NO_LEAD}>No lead</SelectItem>
        {members.map((member) => (
          <SelectItem key={member.user_id} value={member.user_id ?? ''}>
            {memberName(member)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

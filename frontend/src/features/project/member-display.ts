import type { Member } from '@/features/workspace/use-members'

// Two-letter initials from a member's display name, falling back to email.
export function memberInitials(member: Member | undefined): string {
  const source = member?.display_name?.trim() || member?.email || ''
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

// Human-readable member name, falling back to email then a placeholder.
export function memberName(
  member: Member | undefined,
  fallback = 'Unknown',
): string {
  return member?.display_name?.trim() || member?.email || fallback
}

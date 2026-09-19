import { LabelsManager } from '@/features/label/labels-manager'

// Workspace-level labels (team_id null) management. Rendered inside
// SettingsLayout as the "labels" tab. Team-scoped labels live on the team
// detail page instead.
export default function WorkspaceLabelsPage() {
  return <LabelsManager />
}

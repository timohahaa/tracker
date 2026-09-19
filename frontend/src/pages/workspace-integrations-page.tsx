import { GitlabIntegration } from '@/features/integration/gitlab-integration'

// Workspace integrations (GitLab). Rendered inside SettingsLayout as the
// "integrations" tab. Any member can view status; owner/admin can configure.
export default function WorkspaceIntegrationsPage() {
  return <GitlabIntegration />
}

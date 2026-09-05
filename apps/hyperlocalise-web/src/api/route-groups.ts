/*
 * Copyright (c) 2026 Hyperlocalise Pty Ltd
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in this application's LICENSE file.
 *
 * Change Date: Four years after publication of the applicable version.
 *
 * On the Change Date, in accordance with the Business Source License, use
 * of this software will be governed by the GNU General Public License
 * Version 2.0 or later.
 */
import { Hono } from "hono";

import type { FileStorageAdapter } from "@/lib/file-storage/types";
import type {
  JobQueue,
  ProviderAgentCommentQueue,
  ProviderAgentQaQueue,
  ProviderAgentTranslationQueue,
  ProviderAgentWritebackQueue,
  TranslationFileImportQueue,
  TranslationJobEventData,
} from "@/lib/workflow/types";

import { createAgentEmailRoutes } from "./routes/agent-email/agent-email.route";
import { createAgentSlackRoutes } from "./routes/agent-slack/agent-slack.route";
import { createSlackConnectRoutes } from "./routes/slack-connect/slack-connect.route";
import { createApiKeyRoutes } from "./routes/api-key/api-key.route";
import { authRoutes } from "./routes/auth/auth.route";
import { createNativeAuthRoutes } from "./routes/auth/native-auth.route";
import { createConversationRoutes } from "./routes/conversation/conversation.route";
import { createCanvaConnectionRoutes } from "./routes/canva-connection/canva-connection.route";
import { createContentfulConnectionRoutes } from "./routes/contentful-connection/contentful-connection.route";
import { createMcpServerConnectionRoutes } from "./routes/mcp-server-connection/mcp-server-connection.route";
import { createLinkedDomainRoutes } from "./routes/linked-domain/linked-domain.route";
import { createAhrefsConnectionRoutes } from "./routes/ahrefs-connection/ahrefs-connection.route";
import { createSemrushConnectionRoutes } from "./routes/semrush-connection/semrush-connection.route";
import { createIntercomConnectionRoutes } from "./routes/intercom-connection/intercom-connection.route";
import { createGlossaryRoutes } from "./routes/glossary/glossary.route";
import { createKnowledgeMemoryRoutes } from "./routes/knowledge-memory/knowledge-memory.route";
import { createMemoryRoutes } from "./routes/memory/memory.route";
import { createOrganizationIssueSheetRoutes } from "./routes/issues/organization-issue-sheet.route";
import { createOrganizationIssuesRoutes } from "./routes/issues/issues.route";
import { createMentionSuggestionsRoutes } from "./routes/mentions/mention-suggestions.route";
import { createIssueNotificationsRoutes } from "./routes/notifications/notifications.route";
import { createNotificationPreferencesRoutes } from "./routes/notification-preferences/notification-preferences.route";
import { createGithubInstallationRoutes } from "./routes/github-installation/github-installation.route";
import { createWorkspaceJobRoutes } from "./routes/project/job.route";
import { createProjectRoutes } from "./routes/project/project.route";
import { createProviderCredentialRoutes } from "./routes/provider-credential/provider-credential.route";
import { createPublicFileRoutes } from "./routes/public-files/public-files.route";
import { createPublicImageRoutes } from "./routes/public-images/public-images.route";
import { createPublicJobRoutes } from "./routes/public-jobs/public-jobs.route";
import { createPublicTranslationRoutes } from "./routes/public-translations/public-translations.route";
import { createSlackOAuthRoutes } from "./routes/slack-oauth/slack-oauth.route";
import { createFileRoutes } from "./routes/file/file.route";
import { createWorkspaceFilesRoutes } from "./routes/workspace-files/workspace-files.route";
import { createWorkspaceAutomationRoutes } from "./routes/workspace-automation/workspace-automation.route";
import { createVisualWorkflowRoutes } from "./routes/visual-workflow/visual-workflow.route";
import { createExternalTmsProviderCredentialRoutes } from "./routes/external-tms-provider-credential/external-tms-provider-credential.route";
import { createTmsProviderRoutes } from "./routes/tms-provider/tms-provider.route";
import { createTmsAgentAutomationRoutes } from "./routes/tms-agent-automation/tms-agent-automation.route";
import { createTmsDashboardSummaryRoutes } from "./routes/tms-dashboard-summary/tms-dashboard-summary.route";
import { createMemberRoutes } from "./routes/member/member.route";
import { createTeamRoutes } from "./routes/team/team.route";
import { createWorkspaceRoutes } from "./routes/workspace/workspace.route";
import { createBillingRoutes } from "./routes/billing/billing.route";
import { createHyperlabRoutes } from "./routes/hyperlab/hyperlab.route";
import { createReportsRoutes } from "./routes/reports/reports.route";
import { createActivityLogRoutes } from "./routes/activity-log/activity-log.route";
import { createOverviewRoutes } from "./routes/overview/overview.route";

export type OrgScopedRouteOptions = {
  jobQueue: JobQueue<TranslationJobEventData>;
  providerAgentTranslationQueue: ProviderAgentTranslationQueue;
  providerAgentQaQueue: ProviderAgentQaQueue;
  providerAgentCommentQueue: ProviderAgentCommentQueue;
  providerAgentWritebackQueue: ProviderAgentWritebackQueue;
  fileStorageAdapter?: FileStorageAdapter;
  translationFileImportQueue?: TranslationFileImportQueue;
};

export type PublicApiRouteOptions = {
  jobQueue: JobQueue<TranslationJobEventData>;
  fileStorageAdapter?: FileStorageAdapter;
};

export function createAuthRoutes() {
  return new Hono()
    .route("/native", createNativeAuthRoutes())
    .route("/", authRoutes)
    .route("/slack", createSlackOAuthRoutes());
}

export function createPublicApiRoutes(options: PublicApiRouteOptions) {
  return new Hono()
    .route("/files", createPublicFileRoutes({ fileStorageAdapter: options.fileStorageAdapter }))
    .route("/jobs", createPublicJobRoutes(options))
    .route("/projects", createPublicTranslationRoutes())
    .route(
      "/projects",
      createPublicImageRoutes({ fileStorageAdapter: options.fileStorageAdapter }),
    );
}

export function createOrgInboxRoutes(options: OrgScopedRouteOptions) {
  return new Hono()
    .route("/issues", createOrganizationIssuesRoutes())
    .route("/issue-sheet", createOrganizationIssueSheetRoutes())
    .route("/notifications", createIssueNotificationsRoutes())
    .route("/notification-preferences", createNotificationPreferencesRoutes())
    .route("/mentions", createMentionSuggestionsRoutes())
    .route(
      "/conversations",
      createConversationRoutes({ fileStorageAdapter: options.fileStorageAdapter }),
    );
}

export function createOrgKnowledgeRoutes(
  options: Pick<OrgScopedRouteOptions, "fileStorageAdapter"> = {},
) {
  return new Hono()
    .route("/glossaries", createGlossaryRoutes({ fileStorageAdapter: options.fileStorageAdapter }))
    .route("/knowledge-memory", createKnowledgeMemoryRoutes())
    .route("/translation-memories", createMemoryRoutes());
}

export function createOrgProjectsRoutes(options: OrgScopedRouteOptions) {
  return new Hono()
    .route("/projects", createProjectRoutes(options))
    .route(
      "/jobs",
      createWorkspaceJobRoutes({
        jobQueue: options.jobQueue,
        providerAgentTranslationQueue: options.providerAgentTranslationQueue,
        providerAgentQaQueue: options.providerAgentQaQueue,
        providerAgentCommentQueue: options.providerAgentCommentQueue,
        providerAgentWritebackQueue: options.providerAgentWritebackQueue,
      }),
    )
    .route("/files", createFileRoutes({ fileStorageAdapter: options.fileStorageAdapter }))
    .route("/workspace-files", createWorkspaceFilesRoutes())
    .route(
      "/automations",
      createWorkspaceAutomationRoutes({ fileStorageAdapter: options.fileStorageAdapter }),
    )
    .route("/visual-workflows", createVisualWorkflowRoutes());
}

export function createOrgTmsRoutes(options: OrgScopedRouteOptions) {
  return new Hono()
    .route("/external-tms-provider-credential", createExternalTmsProviderCredentialRoutes())
    .route(
      "/tms-provider",
      createTmsProviderRoutes({
        providerAgentTranslationQueue: options.providerAgentTranslationQueue,
      }),
    )
    .route("/tms-agent-automation", createTmsAgentAutomationRoutes())
    .route("/tms-dashboard-summary", createTmsDashboardSummaryRoutes())
    .route("/provider-credential", createProviderCredentialRoutes());
}

export function createOrgIntegrationsRoutes() {
  return new Hono()
    .route("/contentful-connections", createContentfulConnectionRoutes())
    .route("/mcp-server-connections", createMcpServerConnectionRoutes())
    .route("/linked-domains", createLinkedDomainRoutes())
    .route("/semrush-connections", createSemrushConnectionRoutes())
    .route("/ahrefs-connections", createAhrefsConnectionRoutes())
    .route("/intercom-connections", createIntercomConnectionRoutes())
    .route("/canva-connections", createCanvaConnectionRoutes());
}

export function createOrgAgentsRoutes() {
  return new Hono()
    .route("/agent-email", createAgentEmailRoutes())
    .route("/agent-slack", createAgentSlackRoutes())
    .route("/slack-connect", createSlackConnectRoutes())
    .route("/github-installation", createGithubInstallationRoutes());
}

export function createOrgWorkspaceRoutes() {
  return new Hono()
    .route("/teams", createTeamRoutes())
    .route("/members", createMemberRoutes())
    .route("/workspace", createWorkspaceRoutes())
    .route("/billing", createBillingRoutes())
    .route("/api-keys", createApiKeyRoutes())
    .route("/activity-logs", createActivityLogRoutes())
    .route("/reports", createReportsRoutes())
    .route("/hyperlab", createHyperlabRoutes())
    .route("/overview", createOverviewRoutes());
}

export function createOrgScopedAppRoutes(options: OrgScopedRouteOptions) {
  return new Hono()
    .route("/", createOrgInboxRoutes(options))
    .route("/", createOrgKnowledgeRoutes(options))
    .route("/", createOrgProjectsRoutes(options))
    .route("/", createOrgTmsRoutes(options))
    .route("/", createOrgIntegrationsRoutes())
    .route("/", createOrgAgentsRoutes())
    .route("/", createOrgWorkspaceRoutes());
}

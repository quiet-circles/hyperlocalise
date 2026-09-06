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
import { hasCapability, isWorkspaceOperatorRole } from "@/api/auth/policy";
import { getWorkspaceFeatureFlagEnabled, workspaceReportsFlag } from "@/lib/flags/workspace-flags";
import { normalizeProjectId } from "@/lib/projects/identity/project-id";
import { requireAppAuthContext } from "@/lib/workos/app-auth";

import { JobDetailPageContent } from "./_components/job-detail-page-content";
import { OrgPageSuspense } from "../../../../_components/org-page-suspense";

export default function ProjectJobDetailPage({
  params,
}: {
  params: Promise<{ organizationSlug: string; projectId: string; jobId: string }>;
}) {
  return (
    <OrgPageSuspense>
      <ProjectJobDetailPageLoader params={params} />
    </OrgPageSuspense>
  );
}

async function ProjectJobDetailPageLoader({
  params,
}: {
  params: Promise<{ organizationSlug: string; projectId: string; jobId: string }>;
}) {
  const { organizationSlug, projectId: rawProjectId, jobId } = await params;
  const projectId = normalizeProjectId(rawProjectId);
  const auth = await requireAppAuthContext({ organizationSlug });
  const canEditJobFields = hasCapability(auth.membership.role, "jobs:write");
  const canEditSharedCredentialProviderJobFields = isWorkspaceOperatorRole(auth.membership.role);
  const reportsEnabled = await getWorkspaceFeatureFlagEnabled(workspaceReportsFlag, auth);

  return (
    <JobDetailPageContent
      jobId={jobId}
      organizationSlug={organizationSlug}
      projectId={projectId}
      canEditJobFields={canEditJobFields}
      canEditSharedCredentialProviderJobFields={canEditSharedCredentialProviderJobFields}
      reportsEnabled={reportsEnabled}
    />
  );
}

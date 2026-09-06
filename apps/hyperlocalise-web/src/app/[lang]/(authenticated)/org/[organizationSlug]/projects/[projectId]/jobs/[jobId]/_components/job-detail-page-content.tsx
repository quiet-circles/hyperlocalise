"use client";

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
import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client-instance";
import {
  parseProviderJobId,
  resolveEncodedProviderJobId,
} from "@/lib/providers/jobs/tms-provider-resource-id";

import { useActiveTmsProvider } from "../../../../../_hooks/use-active-tms-provider";

import { JobDetailSkeleton } from "./job-detail-skeleton";

import type { JobDetailRecord } from "./job-detail-types";
import { NativeJobDetailContent } from "./native-job-detail-content";
import { ProviderLiveJobDetailContent } from "./provider-live-job-detail-content";

export function JobDetailPageContent({
  jobId,
  organizationSlug,
  projectId,
  canEditJobFields,
  canEditSharedCredentialProviderJobFields = false,
  reportsEnabled = false,
}: {
  jobId: string;
  organizationSlug: string;
  projectId: string;
  canEditJobFields: boolean;
  canEditSharedCredentialProviderJobFields?: boolean;
  reportsEnabled?: boolean;
}) {
  const activeTmsProviderQuery = useActiveTmsProvider(organizationSlug);
  const encodedProviderJobFromRoute = parseProviderJobId(jobId);

  const routingJobQuery = useQuery({
    queryKey: ["job-routing", organizationSlug, projectId, jobId],
    enabled: !encodedProviderJobFromRoute,
    queryFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"].$get({
        param: { organizationSlug, jobId },
      });

      if (!response.ok) {
        return null;
      }

      const body = (await response.json()) as { job: JobDetailRecord };
      return body.job.projectId === projectId ? body.job : null;
    },
  });

  const encodedProviderJobId =
    encodedProviderJobFromRoute !== null
      ? jobId
      : routingJobQuery.data
        ? resolveEncodedProviderJobId({
            jobId,
            projectId,
            externalProviderKind: routingJobQuery.data.externalProviderKind,
            externalJobId: routingJobQuery.data.externalJobId,
            externalTaskId: routingJobQuery.data.externalTaskId,
          })
        : null;

  const useLiveProviderJob =
    Boolean(encodedProviderJobId) &&
    Boolean(activeTmsProviderQuery.data) &&
    parseProviderJobId(encodedProviderJobId)?.providerKind ===
      activeTmsProviderQuery.data?.providerKind;

  if (
    activeTmsProviderQuery.isLoading ||
    (!encodedProviderJobFromRoute && routingJobQuery.isLoading)
  ) {
    return <JobDetailSkeleton />;
  }

  if (useLiveProviderJob && encodedProviderJobId) {
    const providerKind = parseProviderJobId(encodedProviderJobId)?.providerKind;
    const canEditProviderJobFields =
      providerKind === "smartling" ? canEditSharedCredentialProviderJobFields : canEditJobFields;

    return (
      <ProviderLiveJobDetailContent
        jobId={encodedProviderJobId}
        organizationSlug={organizationSlug}
        projectId={projectId}
        canEditJobFields={canEditProviderJobFields}
      />
    );
  }

  return (
    <NativeJobDetailContent
      jobId={jobId}
      organizationSlug={organizationSlug}
      projectId={projectId}
      canEditJobFields={canEditJobFields}
      reportsEnabled={reportsEnabled}
    />
  );
}

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
import { ReportsWorkspace } from "@/components/reports/reports-workspace";
import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  LeftToRightListBulletIcon,
  LinkSquare02Icon,
  RefreshIcon,
  StopCircleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { FormattedMessage, useIntl } from "react-intl";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { MarkdownPreview } from "@/components/markdown-editor/markdown-editor";
import { useAppShellBreadcrumbAppend } from "@/components/app-shell/store/use-app-shell-breadcrumb";
import { apiClient } from "@/lib/api-client-instance";
import {
  buildJobContentEditorHref,
  canOpenJobContentEditor,
} from "@/lib/projects/job-content-editor-routing";

import { getProviderPayloadString } from "../../../../../jobs/_components/provider-crowdin-job-display";

import { NativeJobOwnerField } from "./job-detail-assignee-field";
import { JobDetailEditableTitle } from "./job-detail-editable-title";
import {
  getInputPayloadMetadataDescription,
  jobDetailTaskLayoutFromRecord,
} from "./job-detail-layout-helpers";
import { JobDetailTaskView } from "./job-detail-task-view";
import type { JobDetailRecord } from "./job-detail-types";
import { JobProviderDetailSection } from "./job-provider-detail-section";
import { NativeJobDescriptionField } from "./native-job-description-field";
import {
  isNativeFileTranslationJob,
  NativeJobSourceFilesSection,
} from "./native-job-detail-helpers";
import { nativeJobDetailContentMessages as messages } from "./native-job-detail-content.messages";
import {
  canCancelJob,
  canMarkJobFailed,
  canRetryJob,
  isProviderBackedJob,
} from "./job-detail-types";

async function parseActionError(response: Response, fallback: string) {
  let error: string | undefined;

  try {
    const body = (await response.json()) as { error?: string };
    error = body.error;
  } catch {
    error = undefined;
  }

  return error ? `${fallback}: ${error}` : `${fallback} (${response.status})`;
}

export function NativeJobDetailContent({
  jobId,
  organizationSlug,
  projectId,
  canEditJobFields = false,
}: {
  jobId: string;
  organizationSlug: string;
  projectId: string;
  canEditJobFields?: boolean;
}) {
  const [markFailedDialogOpen, setMarkFailedDialogOpen] = useState(false);
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false);
  const intl = useIntl();
  const queryClient = useQueryClient();
  const jobQueryKey = ["job", organizationSlug, projectId, jobId] as const;

  const jobQuery = useQuery({
    queryKey: jobQueryKey,
    queryFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"].$get({
        param: { organizationSlug, jobId },
      });

      if (!response.ok) {
        throw new Error(intl.formatMessage(messages.failedToLoadJob, { status: response.status }));
      }

      const body = (await response.json()) as { job: JobDetailRecord };
      if (body.job.projectId !== projectId) {
        throw new Error(intl.formatMessage(messages.jobWrongProject));
      }
      return body.job;
    },
  });

  const retryJob = useMutation({
    mutationFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"].retry.$post({
        param: { organizationSlug, jobId },
      });

      if (!response.ok) {
        throw new Error(
          await parseActionError(response, intl.formatMessage(messages.failedToRetryJob)),
        );
      }

      const body = (await response.json()) as { job: JobDetailRecord };
      return body.job;
    },
    onSuccess: async (updatedJob) => {
      queryClient.setQueryData(jobQueryKey, updatedJob);
      await queryClient.invalidateQueries({ queryKey: ["jobs", organizationSlug] });
      toast.success(intl.formatMessage(messages.jobQueuedForRetry));
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : intl.formatMessage(messages.failedToRetryJob),
      );
    },
  });

  const markJobFailed = useMutation({
    mutationFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"][
        "mark-failed"
      ].$post({
        param: { organizationSlug, jobId },
      });

      if (!response.ok) {
        throw new Error(
          await parseActionError(response, intl.formatMessage(messages.failedToMarkJobFailed)),
        );
      }

      const body = (await response.json()) as { job: JobDetailRecord };
      return body.job;
    },
    onSuccess: async (updatedJob) => {
      queryClient.setQueryData(jobQueryKey, updatedJob);
      await queryClient.invalidateQueries({ queryKey: ["jobs", organizationSlug] });
      setMarkFailedDialogOpen(false);
      toast.success(intl.formatMessage(messages.jobMarkedAsFailed));
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : intl.formatMessage(messages.failedToMarkJobFailed),
      );
    },
  });

  const cancelJob = useMutation({
    mutationFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"].cancel.$post({
        param: { organizationSlug, jobId },
      });

      if (!response.ok) {
        throw new Error(
          await parseActionError(response, intl.formatMessage(messages.failedToCancelJob)),
        );
      }

      const body = (await response.json()) as { job: JobDetailRecord };
      return body.job;
    },
    onSuccess: async (updatedJob) => {
      queryClient.setQueryData(jobQueryKey, updatedJob);
      await queryClient.invalidateQueries({ queryKey: ["jobs", organizationSlug] });
      setCancelDialogOpen(false);
      toast.success(intl.formatMessage(messages.jobCancelled));
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : intl.formatMessage(messages.failedToCancelJob),
      );
    },
  });

  const job = jobQuery.data;
  const layout = job ? jobDetailTaskLayoutFromRecord(job, intl) : null;
  const contentEditorHref = job
    ? buildJobContentEditorHref(organizationSlug, projectId, job)
    : null;
  const showCatAction = job ? canOpenJobContentEditor(job) : false;
  const isProviderBacked = Boolean(job && isProviderBackedJob(job));
  const isNativeEditable = Boolean(job && canEditJobFields && !isProviderBacked);
  const description = job
    ? isProviderBacked
      ? (getProviderPayloadString(job.externalProviderPayload, "description") ?? "")
      : getInputPayloadMetadataDescription(job)
    : "";

  const saveTitle = useMutation({
    mutationFn: async (nextTitle: string) => {
      const response = await apiClient.api.orgs[":organizationSlug"].jobs[":jobId"].$patch({
        param: { organizationSlug, jobId },
        json: { title: nextTitle },
      });
      if (!response.ok) {
        throw new Error(
          await parseActionError(response, intl.formatMessage(messages.failedToUpdateJob)),
        );
      }
      const body = (await response.json()) as { job: JobDetailRecord };
      return body.job;
    },
    onSuccess: async (updatedJob) => {
      queryClient.setQueryData(jobQueryKey, updatedJob);
      await queryClient.invalidateQueries({ queryKey: ["jobs", organizationSlug] });
    },
  });

  const properties = (layout?.properties ?? []).map((property) => {
    if (property.id !== "assignees" || !isNativeEditable || !job) {
      return property;
    }
    return {
      ...property,
      value: (
        <NativeJobOwnerField
          organizationSlug={organizationSlug}
          projectId={projectId}
          jobId={jobId}
          ownerUserId={job.ownerUserId}
          assigneeType={job.assigneeType}
          queryKey={jobQueryKey}
          disabled={
            retryJob.isPending ||
            markJobFailed.isPending ||
            cancelJob.isPending ||
            saveTitle.isPending
          }
        />
      ),
    };
  });

  useAppShellBreadcrumbAppend({
    id: "job-detail",
    label: layout?.title,
  });

  const headerActions = job ? (
    <>
      {job.externalUrl ? (
        <Button
          nativeButton={false}
          render={
            <a href={job.externalUrl} target="_blank" rel="noreferrer noopener">
              <HugeiconsIcon icon={LinkSquare02Icon} strokeWidth={1.8} />
              <FormattedMessage
                {...messages.openInProvider}
                values={{ providerKind: job.externalProviderKind }}
              />
            </a>
          }
          size="sm"
          variant="outline"
        />
      ) : null}
      {canRetryJob(job) ? (
        <Button
          size="sm"
          variant="outline"
          disabled={retryJob.isPending || markJobFailed.isPending || cancelJob.isPending}
          onClick={() => retryJob.mutate()}
        >
          <HugeiconsIcon icon={RefreshIcon} strokeWidth={1.8} />
          {retryJob.isPending ? (
            <FormattedMessage {...messages.retrying} />
          ) : (
            <FormattedMessage {...messages.retryJob} />
          )}
        </Button>
      ) : null}
      {canCancelJob(job) ? (
        <Button
          size="sm"
          variant="outline"
          disabled={retryJob.isPending || markJobFailed.isPending || cancelJob.isPending}
          onClick={() => setCancelDialogOpen(true)}
        >
          <FormattedMessage {...messages.cancelJob} />
        </Button>
      ) : null}
      {canMarkJobFailed(job) ? (
        <Button
          size="sm"
          variant="destructive"
          disabled={retryJob.isPending || markJobFailed.isPending || cancelJob.isPending}
          onClick={() => setMarkFailedDialogOpen(true)}
        >
          <HugeiconsIcon icon={StopCircleIcon} strokeWidth={1.8} />
          <FormattedMessage {...messages.markAsFailed} />
        </Button>
      ) : null}
      {showCatAction && contentEditorHref ? (
        <Button size="sm" render={<Link href={contentEditorHref} />}>
          <HugeiconsIcon icon={LeftToRightListBulletIcon} />
          <FormattedMessage {...messages.openEditor} />
        </Button>
      ) : null}
    </>
  ) : null;

  return (
    <>
      <JobDetailTaskView
        jobId={jobId}
        organizationSlug={organizationSlug}
        projectId={projectId}
        title={
          <JobDetailEditableTitle
            title={layout?.title ?? jobId}
            editable={isNativeEditable}
            disabled={
              retryJob.isPending ||
              markJobFailed.isPending ||
              cancelJob.isPending ||
              saveTitle.isPending
            }
            onSave={async (nextTitle) => {
              await saveTitle.mutateAsync(nextTitle);
            }}
          />
        }
        metrics={layout?.metrics ?? []}
        properties={properties}
        secondaryProperties={layout?.secondaryProperties ?? []}
        headerActions={headerActions}
        isLoading={jobQuery.isLoading}
        error={jobQuery.isError ? jobQuery.error : undefined}
        description={description}
        canEditDescription={isNativeEditable}
        renderDescriptionField={
          isNativeEditable
            ? ({ description: fieldDescription, editable }) => (
                <NativeJobDescriptionField
                  organizationSlug={organizationSlug}
                  jobId={jobId}
                  description={fieldDescription}
                  editable={editable}
                  queryKey={jobQueryKey}
                />
              )
            : description.trim().length > 0
              ? ({ description: fieldDescription }) => (
                  <MarkdownPreview value={fieldDescription} className="border-border bg-card" />
                )
              : undefined
        }
        renderFilesSection={
          job && isNativeFileTranslationJob(job)
            ? () => (
                <NativeJobSourceFilesSection
                  organizationSlug={organizationSlug}
                  projectId={projectId}
                  job={job}
                />
              )
            : undefined
        }
        renderExtraMain={
          job && isProviderBackedJob(job)
            ? () => (
                <JobProviderDetailSection
                  job={job}
                  jobId={jobId}
                  organizationSlug={organizationSlug}
                  projectId={projectId}
                  showProviderMetadata={false}
                  showAgentActions={false}
                />
              )
            : () => (
                <ReportsWorkspace
                  organizationSlug={organizationSlug}
                  projectId={projectId}
                  jobId={jobId}
                />
              )
        }
      />

      <AlertDialog
        open={markFailedDialogOpen}
        onOpenChange={(open) => {
          if (!markJobFailed.isPending) {
            setMarkFailedDialogOpen(open);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              <FormattedMessage {...messages.markFailedTitle} />
            </AlertDialogTitle>
            <AlertDialogDescription>
              <FormattedMessage {...messages.markFailedDescription} />
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={markJobFailed.isPending}>
              <FormattedMessage {...messages.cancel} />
            </AlertDialogCancel>
            <Button
              variant="destructive"
              disabled={markJobFailed.isPending}
              onClick={() => markJobFailed.mutate()}
            >
              <HugeiconsIcon icon={StopCircleIcon} strokeWidth={1.8} />
              {markJobFailed.isPending ? (
                <FormattedMessage {...messages.marking} />
              ) : (
                <FormattedMessage {...messages.markFailedConfirm} />
              )}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={cancelDialogOpen}
        onOpenChange={(open) => {
          if (!cancelJob.isPending) {
            setCancelDialogOpen(open);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              <FormattedMessage {...messages.cancelJobTitle} />
            </AlertDialogTitle>
            <AlertDialogDescription>
              <FormattedMessage {...messages.cancelJobDescription} />
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={cancelJob.isPending}>
              <FormattedMessage {...messages.keepJob} />
            </AlertDialogCancel>
            <Button
              variant="destructive"
              disabled={cancelJob.isPending}
              onClick={() => cancelJob.mutate()}
            >
              {cancelJob.isPending ? (
                <FormattedMessage {...messages.cancelling} />
              ) : (
                <FormattedMessage {...messages.cancelJob} />
              )}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

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
import { captureAnalysis, captureCompletions, captureJobStatus } from "@/lib/reporting/capture";
import { and, eq, inArray, isNull, or } from "drizzle-orm";

import { db, schema } from "@/lib/database/client";
import type { ReviewJobEventData } from "@/lib/workflow/types";

type ReviewJobConfig = {
  sourcePath?: string;
  targetLocale?: string;
  translationKeyIds?: string[];
  translationJobId?: string;
};

type ClaimReviewJobInput = {
  event: ReviewJobEventData;
  runId: string;
};

export type ClaimedReviewJob = {
  id: string;
  projectId: string;
  criteria: string;
  targetLocale: string | null;
  config: ReviewJobConfig;
  workflowRunId: string;
};

async function getStoredReviewJob(jobId: string, projectId: string) {
  const [job] = await db
    .select({
      id: schema.jobs.id,
      projectId: schema.jobs.projectId,
      status: schema.jobs.status,
      criteria: schema.reviewJobDetails.criteria,
      targetLocale: schema.reviewJobDetails.targetLocale,
      config: schema.reviewJobDetails.config,
      workflowRunId: schema.jobs.workflowRunId,
      outcomePayload: schema.jobs.outcomePayload,
      lastError: schema.jobs.lastError,
      completedAt: schema.jobs.completedAt,
    })
    .from(schema.jobs)
    .innerJoin(schema.reviewJobDetails, eq(schema.reviewJobDetails.jobId, schema.jobs.id))
    .where(
      and(
        eq(schema.jobs.kind, "review"),
        eq(schema.jobs.id, jobId),
        eq(schema.jobs.projectId, projectId),
      ),
    )
    .limit(1);

  return job ? { ...job, projectId: job.projectId ?? projectId } : undefined;
}

export async function claimReviewJob(input: ClaimReviewJobInput) {
  const attachedJob = await db
    .update(schema.jobs)
    .set({ workflowRunId: input.runId })
    .where(
      and(
        eq(schema.jobs.kind, "review"),
        eq(schema.jobs.id, input.event.jobId),
        eq(schema.jobs.projectId, input.event.projectId),
        isNull(schema.jobs.workflowRunId),
      ),
    )
    .returning({ id: schema.jobs.id, projectId: schema.jobs.projectId })
    .then(async ([job]) => {
      if (job) {
        return {
          id: job.id,
          projectId: job.projectId ?? input.event.projectId,
          runId: input.runId,
          ownedByCurrentRun: true,
        };
      }

      const existingJob = await getStoredReviewJob(input.event.jobId, input.event.projectId);
      if (!existingJob) {
        throw new Error(
          `review job ${input.event.jobId} was not found in project ${input.event.projectId}`,
        );
      }

      return {
        id: existingJob.id,
        projectId: existingJob.projectId,
        runId: existingJob.workflowRunId,
        ownedByCurrentRun: existingJob.workflowRunId === input.runId,
      };
    });

  if (!attachedJob.runId) {
    throw new Error(`review job ${input.event.jobId} does not have an associated workflow run id`);
  }

  if (!attachedJob.ownedByCurrentRun) {
    const existingJob = await getStoredReviewJob(input.event.jobId, input.event.projectId);
    if (!existingJob) {
      throw new Error(
        `review job ${input.event.jobId} was not found in project ${input.event.projectId}`,
      );
    }

    return { kind: "skipped" as const, job: existingJob };
  }

  const [claimedJob] = await db
    .update(schema.jobs)
    .set({
      status: "running",
      lastError: null,
      outcomePayload: null,
      completedAt: null,
    })
    .where(
      and(
        eq(schema.jobs.kind, "review"),
        eq(schema.jobs.id, input.event.jobId),
        eq(schema.jobs.projectId, input.event.projectId),
        or(eq(schema.jobs.status, "queued"), eq(schema.jobs.status, "running")),
        eq(schema.jobs.workflowRunId, attachedJob.runId),
      ),
    )
    .returning({ id: schema.jobs.id, projectId: schema.jobs.projectId });

  if (!claimedJob) {
    const existingJob = await getStoredReviewJob(input.event.jobId, input.event.projectId);
    if (!existingJob) {
      throw new Error(
        `review job ${input.event.jobId} was not found in project ${input.event.projectId}`,
      );
    }

    return { kind: "skipped" as const, job: existingJob };
  }

  const details = await getStoredReviewJob(claimedJob.id, input.event.projectId);
  if (!details) {
    throw new Error(`review job ${claimedJob.id} details were not found`);
  }

  return {
    kind: "claimed" as const,
    job: {
      id: details.id,
      projectId: details.projectId,
      criteria: details.criteria,
      targetLocale: details.targetLocale,
      config: (details.config ?? {}) as ReviewJobConfig,
      workflowRunId: attachedJob.runId,
    } satisfies ClaimedReviewJob,
  };
}

export async function executeClaimedReviewJob(job: ClaimedReviewJob) {
  const result = await executeNativeReviewJob({
    jobId: job.id,
    projectId: job.projectId,
    criteria: job.criteria,
    targetLocale: job.targetLocale,
    config: job.config,
  });

  if (!result.ok) {
    return {
      ok: false as const,
      code: result.code,
      message: result.message,
    };
  }

  return {
    ok: true as const,
    outcome: result.outcome,
    status: result.status,
  };
}

export async function completeReviewJob(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  outcome: unknown;
  status: "succeeded" | "waiting_for_review";
}) {
  const [updatedJob] = await db
    .update(schema.jobs)
    .set({
      status: input.status,
      outcomePayload: input.outcome,
      lastError: input.status === "waiting_for_review" ? "Review found blocking issues" : null,
      completedAt: input.status === "succeeded" ? new Date() : null,
    })
    .where(
      and(
        eq(schema.jobs.kind, "review"),
        eq(schema.jobs.id, input.jobId),
        eq(schema.jobs.projectId, input.projectId),
        eq(schema.jobs.workflowRunId, input.workflowRunId),
      ),
    )
    .returning({ id: schema.jobs.id });

  if (!updatedJob) {
    throw new Error(
      `review job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
    );
  }

  await captureJobStatus({
    jobId: input.jobId,
    status: input.status,
    operationKey: `review:${input.jobId}:${input.status === "succeeded" ? "succeeded" : "waiting"}`,
  });
  return getStoredReviewJob(input.jobId, input.projectId);
}

export async function failReviewJob(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  code: string;
  message: string;
}) {
  const [updatedJob] = await db
    .update(schema.jobs)
    .set({
      status: "failed",
      outcomePayload: { code: input.code, message: input.message },
      lastError: input.message,
      completedAt: new Date(),
    })
    .where(
      and(
        eq(schema.jobs.kind, "review"),
        eq(schema.jobs.id, input.jobId),
        eq(schema.jobs.projectId, input.projectId),
        eq(schema.jobs.workflowRunId, input.workflowRunId),
      ),
    )
    .returning({ id: schema.jobs.id });

  if (!updatedJob) {
    throw new Error(
      `review job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
    );
  }

  return getStoredReviewJob(input.jobId, input.projectId);
}

export async function executeNativeReviewJob(input: {
  jobId: string;
  projectId: string;
  criteria: string;
  targetLocale: string | null;
  config: ReviewJobConfig;
}) {
  const [project] = await db
    .select({
      organizationId: schema.projects.organizationId,
      sourceLocale: schema.projects.sourceLocale,
    })
    .from(schema.projects)
    .where(eq(schema.projects.id, input.projectId))
    .limit(1);

  if (!project?.organizationId) {
    return {
      ok: false as const,
      code: "review_project_not_found",
      message: "Review project was not found",
    };
  }

  const targetLocale = input.targetLocale ?? input.config.targetLocale ?? null;
  if (!targetLocale) {
    return {
      ok: false as const,
      code: "review_target_locale_required",
      message: "Review jobs require a target locale",
    };
  }

  const translationConditions = [
    eq(schema.projectTranslations.organizationId, project.organizationId),
    eq(schema.projectTranslations.projectId, input.projectId),
    eq(schema.projectTranslations.targetLocale, targetLocale),
    eq(schema.projectTranslations.status, "needs_review"),
  ];

  if (input.config.translationKeyIds?.length) {
    translationConditions.push(
      inArray(schema.projectTranslations.translationKeyId, input.config.translationKeyIds),
    );
  }

  const keyConditions = [
    eq(schema.projectTranslationKeys.organizationId, project.organizationId),
    eq(schema.projectTranslationKeys.projectId, input.projectId),
    eq(schema.projectTranslationKeys.isHidden, false),
  ];

  if (input.config.sourcePath) {
    const [sourceFile] = await db
      .select({ id: schema.repositorySourceFiles.id })
      .from(schema.repositorySourceFiles)
      .where(
        and(
          eq(schema.repositorySourceFiles.organizationId, project.organizationId),
          eq(schema.repositorySourceFiles.projectId, input.projectId),
          eq(schema.repositorySourceFiles.sourcePath, input.config.sourcePath),
        ),
      )
      .limit(1);

    if (!sourceFile) {
      return {
        ok: false as const,
        code: "review_source_file_not_found",
        message: "Review source file was not found",
      };
    }

    keyConditions.push(eq(schema.projectTranslationKeys.repositorySourceFileId, sourceFile.id));
  }

  const rows = await db
    .select({
      translationId: schema.projectTranslations.id,
      translationKeyId: schema.projectTranslations.translationKeyId,
      text: schema.projectTranslations.text,
      status: schema.projectTranslations.status,
      key: schema.projectTranslationKeys.key,
      sourceText: schema.projectTranslationKeys.sourceText,
      maxLength: schema.projectTranslationKeys.maxLength,
    })
    .from(schema.projectTranslations)
    .innerJoin(
      schema.projectTranslationKeys,
      eq(schema.projectTranslations.translationKeyId, schema.projectTranslationKeys.id),
    )
    .where(and(...translationConditions, ...keyConditions));

  await captureAnalysis({
    organizationId: project.organizationId,
    projectId: input.projectId,
    jobId: input.jobId,
    sourceLocale: project.sourceLocale ?? "en",
    targetLocale,
    sourceEntries: Object.fromEntries(rows.map((row) => [row.translationKeyId, row.sourceText])),
    step: "review",
  });
  await captureJobStatus({
    jobId: input.jobId,
    status: "running",
    operationKey: `review:${input.jobId}:running`,
  });

  const findings: Array<{
    translationKeyId: string;
    key: string;
    issue: string;
    severity: "warn" | "error";
  }> = [];

  for (const row of rows) {
    if (!row.text.trim()) {
      findings.push({
        translationKeyId: row.translationKeyId,
        key: row.key,
        issue: "Missing translation text",
        severity: "error",
      });
      continue;
    }

    if (row.maxLength && row.text.length > row.maxLength) {
      findings.push({
        translationKeyId: row.translationKeyId,
        key: row.key,
        issue: `Translation exceeds max length (${row.text.length}/${row.maxLength})`,
        severity: "error",
      });
    }

    if (row.sourceText.trim() === row.text.trim()) {
      findings.push({
        translationKeyId: row.translationKeyId,
        key: row.key,
        issue: "Translation matches source text",
        severity: "warn",
      });
    }
  }

  const blocking = findings.filter((finding) => finding.severity === "error");
  const reviewedIds = rows.map((row) => row.translationId);

  if (reviewedIds.length > 0) {
    await db
      .update(schema.projectTranslations)
      .set({
        status: blocking.length > 0 ? "needs_review" : "approved",
        reviewedAt: blocking.length > 0 ? null : new Date(),
      })
      .where(inArray(schema.projectTranslations.id, reviewedIds));
  }

  if (blocking.length === 0)
    await captureCompletions({
      organizationId: project.organizationId,
      jobId: input.jobId,
      targetLocale,
      step: "review",
      provenance: "automated",
      sourceEntries: Object.fromEntries(rows.map((row) => [row.translationKeyId, row.sourceText])),
    });
  const outcome = {
    criteria: input.criteria,
    targetLocale,
    reviewedCount: rows.length,
    issueCount: findings.length,
    blockingIssueCount: blocking.length,
    findings,
  };

  const jobStatus = blocking.length > 0 ? "waiting_for_review" : "succeeded";

  return {
    ok: true as const,
    outcome,
    status: jobStatus as "succeeded" | "waiting_for_review",
  };
}

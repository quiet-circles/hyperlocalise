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
import type { StringTranslationJobResult } from "@/lib/translation/domain";
import type { ClaimedTranslationJob } from "@/lib/translation/jobs";
import type { TranslationJobEventData } from "@/lib/workflow/types";

export async function claimTranslationJobStep(input: {
  event: TranslationJobEventData;
  runId: string;
}) {
  "use step";
  const { claimTranslationJob } = await import("@/lib/translation/jobs");
  return claimTranslationJob(input);
}

export async function ensureAiFeaturesAllowedStep(input: { organizationId: string }) {
  "use step";
  const { ensureAiFeaturesAllowed } = await import("@/lib/billing/ai-features");
  return ensureAiFeaturesAllowed(input);
}

export async function executeClaimedTranslationJobStep(job: ClaimedTranslationJob) {
  "use step";
  const { executeClaimedTranslationJob } = await import("@/lib/translation/jobs");
  return executeClaimedTranslationJob(job);
}

export async function completeTranslationJobStep(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  result: StringTranslationJobResult;
}) {
  "use step";
  const { completeTranslationJob } = await import("@/lib/translation/jobs");
  return completeTranslationJob(input);
}

export async function failTranslationJobStep(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  code: string;
  message: string;
}) {
  "use step";
  const { failTranslationJob } = await import("@/lib/translation/jobs");
  return failTranslationJob(input);
}

export async function markEmailTranslationJobRunning(input: {
  jobId: string;
  workflowRunId: string;
}) {
  "use step";
  const { and, eq, isNull, or } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  const [updatedJob] = await db
    .update(schema.jobs)
    .set({
      status: "running",
      workflowRunId: input.workflowRunId,
      lastError: null,
      outcomePayload: null,
      completedAt: null,
    })
    .where(
      and(
        eq(schema.jobs.id, input.jobId),
        eq(schema.jobs.kind, "translation"),
        or(isNull(schema.jobs.workflowRunId), eq(schema.jobs.workflowRunId, input.workflowRunId)),
        // Do not claim terminal jobs: legacy rows may have null workflowRunId, and replays must not
        // reset succeeded/failed jobs that already share this workflowRunId.
        or(eq(schema.jobs.status, "queued"), eq(schema.jobs.status, "running")),
      ),
    )
    .returning({ id: schema.jobs.id });

  if (!updatedJob) {
    throw new Error(
      `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
    );
  }
}

export async function markEmailTranslationJobSucceeded(input: {
  jobId: string;
  workflowRunId: string;
  sourceFilename: string;
  outputFilename: string;
  targetLocale: string;
}) {
  "use step";
  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  const succeededJob = await db.transaction(async (tx) => {
    const [updatedJob] = await tx
      .update(schema.jobs)
      .set({
        status: "succeeded",
        outcomePayload: {
          kind: "email_file_result",
          sourceFilename: input.sourceFilename,
          outputFilename: input.outputFilename,
          targetLocale: input.targetLocale,
        },
        lastError: null,
        completedAt: new Date(),
      })
      .where(
        and(
          eq(schema.jobs.id, input.jobId),
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.workflowRunId, input.workflowRunId),
        ),
      )
      .returning({ id: schema.jobs.id, organizationId: schema.jobs.organizationId });

    if (!updatedJob) {
      throw new Error(
        `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
      );
    }

    await tx
      .update(schema.translationJobDetails)
      .set({ outcomeKind: "file_result" })
      .where(eq(schema.translationJobDetails.jobId, input.jobId));

    return updatedJob;
  });

  const { completeAndTrackBillableUsage, formatUsageControlError } =
    await import("@/lib/billing/usage-control");
  const { isErr } = await import("@/lib/primitives/result/results");
  const operationKey = `job:${input.jobId}:translation_jobs`;
  const trackUsageResult = await completeAndTrackBillableUsage({
    organizationId: succeededJob.organizationId,
    operationKey,
    autumnEventName: "translation_job.completed",
    unit: "job",
    jobId: input.jobId,
    aiCreditSource: "email_translation_job_complete",
  });

  if (isErr(trackUsageResult)) {
    console.error("[email-translation-job] Autumn usage tracking failed after job succeeded", {
      jobId: input.jobId,
      operationKey,
      error: formatUsageControlError(trackUsageResult.error),
    });
  }
}

export async function markEmailTranslationJobFailed(input: {
  jobId: string;
  workflowRunId: string;
  reason: string;
}) {
  "use step";
  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  await db.transaction(async (tx) => {
    const [updatedJob] = await tx
      .update(schema.jobs)
      .set({
        status: "failed",
        outcomePayload: {
          kind: "email_file_error",
          message: input.reason,
        },
        lastError: input.reason,
        completedAt: new Date(),
      })
      .where(
        and(
          eq(schema.jobs.id, input.jobId),
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.workflowRunId, input.workflowRunId),
        ),
      )
      .returning({ id: schema.jobs.id });

    if (!updatedJob) {
      throw new Error(
        `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
      );
    }

    await tx
      .update(schema.translationJobDetails)
      .set({ outcomeKind: "error" })
      .where(eq(schema.translationJobDetails.jobId, input.jobId));
  });

  const { PRODUCT_USAGE_ANALYTICS_EVENTS } = await import("@/lib/analytics/events");
  const { serverAnalytics } = await import("@/lib/analytics/server");
  serverAnalytics.track(PRODUCT_USAGE_ANALYTICS_EVENTS.translationJobFailed, {
    status: "failed",
    source: "email_translation_job",
  });
}

export async function getProjectOrganizationStep(projectId: string): Promise<string> {
  "use step";
  const { eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  const [project] = await db
    .select({ organizationId: schema.projects.organizationId })
    .from(schema.projects)
    .where(eq(schema.projects.id, projectId))
    .limit(1);

  if (!project) {
    throw new Error(`project ${projectId} not found`);
  }

  return project.organizationId;
}

export async function getStoredFileStep(fileId: string, organizationId: string) {
  "use step";
  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  const [file] = await db
    .select()
    .from(schema.storedFiles)
    .where(
      and(eq(schema.storedFiles.id, fileId), eq(schema.storedFiles.organizationId, organizationId)),
    )
    .limit(1);

  if (!file) {
    throw new Error(`stored file ${fileId} not found`);
  }

  return file;
}

export async function getRepositorySourcePathForStoredFileStep(
  fileId: string,
  organizationId: string,
) {
  "use step";
  const { getRepositorySourceFileVersionForStoredFile } =
    await import("@/lib/file-storage/records");
  const version = await getRepositorySourceFileVersionForStoredFile({
    fileId,
    organizationId,
  });

  return version?.sourcePath ?? null;
}

export async function getStoredFileContentStep(fileId: string, organizationId: string) {
  "use step";
  const { get } = await import("@vercel/blob");
  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");
  const { env } = await import("@/lib/env");

  const [file] = await db
    .select({ storageKey: schema.storedFiles.storageKey })
    .from(schema.storedFiles)
    .where(
      and(eq(schema.storedFiles.id, fileId), eq(schema.storedFiles.organizationId, organizationId)),
    )
    .limit(1);

  if (!file) {
    throw new Error(`stored file ${fileId} not found`);
  }

  const storedObject = await get(file.storageKey, {
    access: env.FILE_STORAGE_ACCESS,
    token: env.BLOB_READ_WRITE_TOKEN,
  });

  if (!storedObject?.stream) {
    throw new Error(`stored file ${fileId} content not found`);
  }

  const arrayBuffer = await new Response(storedObject.stream).arrayBuffer();
  return Buffer.from(arrayBuffer);
}

export async function storeOutputFileStep(input: {
  organizationId: string;
  projectId: string;
  jobId: string;
  filename: string;
  contentType: string;
  content: Buffer;
}) {
  "use step";
  const { del, put } = await import("@vercel/blob");
  const { db, schema } = await import("@/lib/database/client");
  const { env } = await import("@/lib/env");
  const { createStoredFileId, sha256Hex, storageKey } = await import("@/lib/file-storage/records");

  const id = createStoredFileId();
  const key = storageKey({
    organizationId: input.organizationId,
    projectId: input.projectId,
    id,
    filename: input.filename,
  });
  const uploaded = await put(key, input.content, {
    access: env.FILE_STORAGE_ACCESS,
    addRandomSuffix: false,
    contentType: input.contentType,
    token: env.BLOB_READ_WRITE_TOKEN,
  });

  try {
    const [file] = await db
      .insert(schema.storedFiles)
      .values({
        id,
        organizationId: input.organizationId,
        projectId: input.projectId,
        createdByUserId: null,
        role: "output",
        sourceKind: "job_output",
        sourceInteractionId: null,
        sourceJobId: input.jobId,
        storageProvider: "vercel_blob",
        storageKey: uploaded.pathname,
        storageUrl: uploaded.url,
        downloadUrl: uploaded.downloadUrl ?? null,
        filename: input.filename,
        contentType: uploaded.contentType,
        byteSize: input.content.byteLength,
        sha256: await sha256Hex(input.content),
        etag: uploaded.etag ?? null,
        metadata: {},
      })
      .returning();

    if (!file) {
      throw new Error(`failed to create stored file record for ${input.filename}`);
    }

    return file;
  } catch (error) {
    await del(uploaded.pathname, { token: env.BLOB_READ_WRITE_TOKEN });
    throw error;
  }
}

export async function reuseFileTranslationMemoryEntriesStep(input: {
  projectId: string;
  sourceLocale: string;
  targetLocale: string;
  sourceEntries: Record<string, string>;
}) {
  "use step";
  const { reuseFileTranslationMemoryEntries } = await import("@/lib/translation/file-memory");
  return reuseFileTranslationMemoryEntries(input);
}

export async function loadProjectTranslationsAsPrefilledEntriesStep(input: {
  organizationId: string;
  projectId: string;
  sourcePath: string;
  targetLocale: string;
}) {
  "use step";
  const { loadProjectTranslationsAsPrefilledEntries } =
    await import("@/lib/projects/translations/project-translation-service");
  return loadProjectTranslationsAsPrefilledEntries(input);
}

export async function persistFileTranslationMemoryEntriesStep(input: {
  projectId: string;
  jobId: string;
  sourceLocale: string;
  targetLocale: string;
  sourcePath: string;
  sourceFileHash: string;
  sourceEntries: Record<string, string>;
  targetEntries: Record<string, string>;
}) {
  "use step";
  const { persistFileTranslationMemoryEntries } = await import("@/lib/translation/file-memory");
  return persistFileTranslationMemoryEntries(input);
}

export async function persistFileProjectTranslationsStep(input: {
  organizationId: string;
  projectId: string;
  jobId: string;
  sourcePath: string;
  sourceLocale: string;
  targetLocale: string;
  sourceEntries: Record<string, string>;
  targetEntries: Record<string, string>;
}) {
  "use step";
  const { persistFileJobTranslations } =
    await import("@/lib/projects/translations/project-translation-service");
  return persistFileJobTranslations(input);
}

export async function persistDocumentVariantBytesStep(input: {
  organizationId: string;
  projectId: string;
  sourcePath: string;
  targetLocale: string;
  content: Buffer;
  contentType: string;
  filename: string;
  repositorySourceFileId?: string | null;
  sourceJobId?: string | null;
}) {
  "use step";
  const { getImageVariant, replaceImageVariantBytes } =
    await import("@/lib/projects/files/image-variant-service");
  const result = await replaceImageVariantBytes({
    organizationId: input.organizationId,
    projectId: input.projectId,
    sourcePath: input.sourcePath,
    targetLocale: input.targetLocale,
    content: input.content,
    contentType: input.contentType,
    filename: input.filename,
    repositorySourceFileId: input.repositorySourceFileId,
    sourceJobId: input.sourceJobId,
    provenance: "translation_job",
  });
  if (!result.ok) {
    if (result.error.code === "approved_locked") {
      const existing = await getImageVariant({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
        targetLocale: input.targetLocale,
      });
      if (existing?.storedFileId) {
        return existing;
      }
    }
    throw new Error(`failed to persist document variant: ${result.error.code}`);
  }
  return result.value;
}

export async function completeFileTranslationJobStep(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  outputFiles: Array<{ fileId: string; locale: string; filename: string }>;
}) {
  "use step";
  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");

  const didSucceed = await db.transaction(async (tx) => {
    const [updatedJob] = await tx
      .update(schema.jobs)
      .set({
        status: "succeeded",
        outcomePayload: {
          outputFiles: input.outputFiles,
        },
        lastError: null,
        completedAt: new Date(),
      })
      .where(
        and(
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.id, input.jobId),
          eq(schema.jobs.projectId, input.projectId),
          eq(schema.jobs.workflowRunId, input.workflowRunId),
        ),
      )
      .returning({ id: schema.jobs.id });

    if (!updatedJob) {
      return false;
    }

    await tx
      .update(schema.translationJobDetails)
      .set({ outcomeKind: "file_result" })
      .where(eq(schema.translationJobDetails.jobId, input.jobId));

    return true;
  });

  if (!didSucceed) {
    throw new Error(
      `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
    );
  }

  const { captureJobStatus } = await import("@/lib/reporting/capture");
  await captureJobStatus({
    jobId: input.jobId,
    status: "succeeded",
    operationKey: `file:${input.workflowRunId}:succeeded`,
  });
  const { completeAndTrackBillableUsage, formatUsageControlError } =
    await import("@/lib/billing/usage-control");
  const { isErr } = await import("@/lib/primitives/result/results");
  const operationKey = `job:${input.jobId}:translation_jobs`;
  const [jobForUsage] = await db
    .select({ organizationId: schema.jobs.organizationId })
    .from(schema.jobs)
    .where(eq(schema.jobs.id, input.jobId))
    .limit(1);
  if (!jobForUsage?.organizationId) {
    throw new Error(`translation job ${input.jobId} has no organization for usage tracking`);
  }

  const trackUsageResult = await completeAndTrackBillableUsage({
    organizationId: jobForUsage.organizationId,
    operationKey,
    autumnEventName: "translation_job.completed",
    unit: "job",
    jobId: input.jobId,
    aiCreditSource: "translation_job_complete",
  });

  if (isErr(trackUsageResult)) {
    console.error("[file-translation-job] Autumn usage tracking failed after job succeeded", {
      jobId: input.jobId,
      operationKey,
      error: formatUsageControlError(trackUsageResult.error),
    });
  }
}

export async function localizeImageVariantForJobStep(input: {
  organizationId: string;
  projectId: string;
  sourcePath: string;
  targetLocale: string;
  sourceLocale?: string | null;
  sourceStoredFileId: string;
  sourceJobId: string;
  createdByUserId?: string | null;
}) {
  "use step";
  const { getRepositorySourceFileVersionForStoredFile } =
    await import("@/lib/file-storage/records");
  const { getImageVariant, localizeAndStoreImageVariant } =
    await import("@/lib/projects/files/image-variant-service");
  const { localizedImageOutputFilename } = await import("@/lib/agents/image-localization");

  const version = await getRepositorySourceFileVersionForStoredFile({
    fileId: input.sourceStoredFileId,
    organizationId: input.organizationId,
  });

  const filename = localizedImageOutputFilename(
    input.sourcePath.split("/").at(-1) ?? "image.png",
    input.targetLocale,
    "image/png",
  );

  const result = await localizeAndStoreImageVariant({
    organizationId: input.organizationId,
    projectId: input.projectId,
    sourcePath: input.sourcePath,
    targetLocale: input.targetLocale,
    sourceLocale: input.sourceLocale,
    sourceStoredFileId: input.sourceStoredFileId,
    repositorySourceFileId: version?.repositorySourceFileId ?? null,
    provenance: "translation_job",
    sourceJobId: input.sourceJobId,
    createdByUserId: input.createdByUserId,
  });

  if (!result.ok) {
    if (result.error.code === "approved_locked") {
      const existing = await getImageVariant({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
        targetLocale: input.targetLocale,
      });

      if (!existing?.storedFileId) {
        throw new Error("image localization failed: approved_locked");
      }

      return {
        fileId: existing.storedFileId,
        locale: input.targetLocale,
        filename,
      };
    }

    throw new Error(`image localization failed: ${result.error.code}`);
  }

  if (!result.value.storedFileId) {
    throw new Error("image localization produced no stored file");
  }

  return {
    fileId: result.value.storedFileId,
    locale: input.targetLocale,
    filename,
  };
}

export async function localizeVideoVariantForJobStep(input: {
  organizationId: string;
  projectId: string;
  sourcePath: string;
  targetLocale: string;
  sourceLocale?: string | null;
  sourceStoredFileId: string;
  sourceJobId: string;
  createdByUserId?: string | null;
}) {
  "use step";
  const { getRepositorySourceFileVersionForStoredFile } =
    await import("@/lib/file-storage/records");
  const { getVideoVariant, localizeAndStoreVideoVariant } =
    await import("@/lib/projects/files/video-variant-service");
  const { localizedVideoOutputFilename } = await import("@/lib/agents/video-localization");

  const version = await getRepositorySourceFileVersionForStoredFile({
    fileId: input.sourceStoredFileId,
    organizationId: input.organizationId,
  });

  const filename = localizedVideoOutputFilename(
    input.sourcePath.split("/").at(-1) ?? "video.mp4",
    input.targetLocale,
  );

  const result = await localizeAndStoreVideoVariant({
    organizationId: input.organizationId,
    projectId: input.projectId,
    sourcePath: input.sourcePath,
    targetLocale: input.targetLocale,
    sourceLocale: input.sourceLocale,
    sourceStoredFileId: input.sourceStoredFileId,
    repositorySourceFileId: version?.repositorySourceFileId ?? null,
    provenance: "translation_job",
    sourceJobId: input.sourceJobId,
    createdByUserId: input.createdByUserId,
  });

  if (!result.ok) {
    if (result.error.code === "approved_locked") {
      const existing = await getVideoVariant({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
        targetLocale: input.targetLocale,
      });

      if (!existing?.storedFileId) {
        throw new Error("video localization failed: approved_locked");
      }

      return {
        fileId: existing.storedFileId,
        locale: input.targetLocale,
        filename,
      };
    }

    throw new Error(`video localization failed: ${result.error.code}`);
  }

  if (!result.value.storedFileId) {
    throw new Error("video localization produced no stored file");
  }

  return {
    fileId: result.value.storedFileId,
    locale: input.targetLocale,
    filename,
  };
}

export async function captureFileAnalysisStep(input: {
  organizationId: string;
  projectId: string;
  jobId: string;
  sourceLocale: string;
  targetLocale: string;
  sourceEntries: Record<string, string>;
}) {
  "use step";
  try {
    const { captureAnalysis } = await import("@/lib/reporting/capture");
    await captureAnalysis(input);
  } catch (error) {
    console.warn("[file-translation-workflow] analysis capture failed", {
      jobId: input.jobId,
      error,
    });
  }
}

export async function captureFileCompletionsStep(input: {
  organizationId: string;
  jobId: string;
  targetLocale: string;
  sourceEntries: Record<string, string>;
  targetEntries: Record<string, string>;
}) {
  "use step";
  const { captureCompletions } = await import("@/lib/reporting/capture");
  await captureCompletions({
    ...input,
    provenance: "automated",
    sourceEntries: Object.fromEntries(
      Object.entries(input.sourceEntries).filter(([key]) => input.targetEntries[key]?.trim()),
    ),
  });
}

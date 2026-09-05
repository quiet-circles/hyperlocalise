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
import { and, eq, isNull, or } from "drizzle-orm";

import { stringTranslationJobInputSchema } from "@/api/routes/project/job.schema";
import { PRODUCT_USAGE_ANALYTICS_EVENTS } from "@/lib/analytics/events";
import { serverAnalytics } from "@/lib/analytics/server";
import { ensureAiFeaturesAllowed } from "@/lib/billing/ai-features";
import { db, schema } from "@/lib/database/client";
import type { TranslationJobEventData } from "@/lib/workflow/types";
import {
  isProjectTranslationKeyHidden,
  persistStringJobTranslations,
} from "@/lib/projects/translations/project-translation-service";
import {
  completeAndTrackBillableUsage,
  formatUsageControlError,
} from "@/lib/billing/usage-control";
import { isErr } from "@/lib/primitives/result/results";
import {
  defaultGlossaryMatchResolution,
  defaultTranslationMemoryMatchResolution,
} from "@/lib/providers/capabilities/match-resolution";
import { TranslationContextBuilder } from "@/lib/translation/context";
import { OrganizationModelResolver } from "@/lib/translation/generation";
import type {
  StringTranslationGenerator,
  StringTranslationJobResult,
} from "@/lib/translation/domain";

export type ClaimedTranslationJob = {
  id: string;
  projectId: string;
  type: "string" | "file";
  inputPayload: unknown;
  workflowRunId: string;
  knowledgeMemoryEnabled: boolean;
};

export type ClaimTranslationJobResult =
  | {
      kind: "claimed";
      job: ClaimedTranslationJob;
    }
  | {
      kind: "skipped";
      job: StoredTranslationJob;
    };

export type TranslationJobExecutionResult =
  | {
      ok: true;
      result: StringTranslationJobResult;
    }
  | {
      ok: false;
      code: string;
      message: string;
    };

type StoredTranslationJob = NonNullable<Awaited<ReturnType<TranslationJobRepository["getStored"]>>>;

class TranslationJobRepository {
  async getStored(jobId: string, projectId: string) {
    const [job] = await db
      .select({
        id: schema.jobs.id,
        projectId: schema.jobs.projectId,
        type: schema.translationJobDetails.type,
        status: schema.jobs.status,
        inputPayload: schema.jobs.inputPayload,
        outcomeKind: schema.translationJobDetails.outcomeKind,
        outcomePayload: schema.jobs.outcomePayload,
        lastError: schema.jobs.lastError,
        workflowRunId: schema.jobs.workflowRunId,
        completedAt: schema.jobs.completedAt,
      })
      .from(schema.jobs)
      .innerJoin(
        schema.translationJobDetails,
        eq(schema.translationJobDetails.jobId, schema.jobs.id),
      )
      .where(
        and(
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.id, jobId),
          eq(schema.jobs.projectId, projectId),
        ),
      )
      .limit(1);

    return job ? { ...job, projectId: job.projectId ?? projectId } : undefined;
  }

  async claim(input: {
    event: TranslationJobEventData;
    runId: string;
  }): Promise<ClaimTranslationJobResult> {
    const attachedJob = await db
      .update(schema.jobs)
      .set({ workflowRunId: input.runId })
      .where(
        and(
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.id, input.event.jobId),
          eq(schema.jobs.projectId, input.event.projectId),
          isNull(schema.jobs.workflowRunId),
        ),
      )
      .returning({
        id: schema.jobs.id,
        projectId: schema.jobs.projectId,
        runId: schema.jobs.workflowRunId,
      })
      .then(async ([job]) => {
        if (job) {
          const storedJob = await this.getStored(job.id, job.projectId ?? input.event.projectId);
          if (!storedJob) {
            throw new Error(
              `translation job ${input.event.jobId} was not found in project ${input.event.projectId}`,
            );
          }

          return {
            id: storedJob.id,
            projectId: storedJob.projectId,
            type: storedJob.type,
            runId: input.runId,
            ownedByCurrentRun: true,
          };
        }

        const existingJob = await this.getStored(input.event.jobId, input.event.projectId);
        if (!existingJob) {
          throw new Error(
            `translation job ${input.event.jobId} was not found in project ${input.event.projectId}`,
          );
        }

        return {
          id: existingJob.id,
          projectId: existingJob.projectId,
          type: existingJob.type,
          runId: existingJob.workflowRunId,
          ownedByCurrentRun: existingJob.workflowRunId === input.runId,
        };
      });

    if (!attachedJob.runId) {
      throw new Error(
        `translation job ${input.event.jobId} does not have an associated workflow run id`,
      );
    }

    if (!attachedJob.ownedByCurrentRun) {
      const existingJob = await this.getStored(input.event.jobId, input.event.projectId);
      if (!existingJob) {
        throw new Error(
          `translation job ${input.event.jobId} was not found in project ${input.event.projectId}`,
        );
      }

      return { kind: "skipped", job: existingJob };
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
          eq(schema.jobs.kind, "translation"),
          eq(schema.jobs.id, input.event.jobId),
          eq(schema.jobs.projectId, input.event.projectId),
          or(eq(schema.jobs.status, "queued"), eq(schema.jobs.status, "running")),
          eq(schema.jobs.workflowRunId, attachedJob.runId),
        ),
      )
      .returning({
        id: schema.jobs.id,
        projectId: schema.jobs.projectId,
        inputPayload: schema.jobs.inputPayload,
      });

    if (!claimedJob) {
      const existingJob = await this.getStored(input.event.jobId, input.event.projectId);
      if (!existingJob) {
        throw new Error(
          `translation job ${input.event.jobId} was not found in project ${input.event.projectId}`,
        );
      }

      return { kind: "skipped", job: existingJob };
    }

    await db
      .update(schema.translationJobDetails)
      .set({ outcomeKind: null })
      .where(eq(schema.translationJobDetails.jobId, input.event.jobId));

    return {
      kind: "claimed",
      job: {
        id: claimedJob.id,
        projectId: claimedJob.projectId ?? input.event.projectId,
        type: attachedJob.type,
        inputPayload: claimedJob.inputPayload,
        workflowRunId: attachedJob.runId,
        knowledgeMemoryEnabled: input.event.knowledgeMemoryEnabled === true,
      },
    };
  }
}

class TranslationJobExecutor {
  constructor(
    private readonly contextBuilder = new TranslationContextBuilder(),
    private readonly modelResolver = new OrganizationModelResolver(),
  ) {}

  async execute(
    claimedJob: ClaimedTranslationJob,
    translateStringJobOverride?: StringTranslationGenerator,
  ): Promise<TranslationJobExecutionResult> {
    if (claimedJob.type !== "string") {
      return {
        ok: false,
        code: "unsupported_job_type",
        message: `translation job type ${claimedJob.type} is not supported`,
      };
    }

    const [project] = await db
      .select({ organizationId: schema.projects.organizationId })
      .from(schema.projects)
      .where(eq(schema.projects.id, claimedJob.projectId))
      .limit(1);
    if (!project) {
      return {
        ok: false,
        code: "translation_project_not_found",
        message: "translation project not found",
      };
    }

    const aiFeatures = await ensureAiFeaturesAllowed({ organizationId: project.organizationId });
    if (!aiFeatures.ok) {
      return {
        ok: false,
        code: aiFeatures.error.code,
        message: aiFeatures.error.message,
      };
    }

    const parsedInput = stringTranslationJobInputSchema.safeParse(claimedJob.inputPayload);
    if (!parsedInput.success) {
      return {
        ok: false,
        code: "invalid_string_translation_job_input",
        message: "invalid stored string translation job input",
      };
    }

    if (parsedInput.data.translationKeyId) {
      const isHidden = await isProjectTranslationKeyHidden({
        projectId: claimedJob.projectId,
        translationKeyId: parsedInput.data.translationKeyId,
      });
      if (isHidden) {
        return {
          ok: false,
          code: "translation_key_hidden",
          message: "Hidden source strings are skipped by translation jobs",
        };
      }
    }

    const contextResult = await this.contextBuilder.build(
      claimedJob.projectId,
      parsedInput.data,
      undefined,
      {
        knowledgeMemoryEnabled: claimedJob.knowledgeMemoryEnabled,
        translationMemoryMatchResolution: defaultTranslationMemoryMatchResolution,
        glossaryMatchResolution: defaultGlossaryMatchResolution,
      },
    );

    if (!contextResult.ok) {
      return {
        ok: false,
        code: contextResult.code,
        message: contextResult.message,
      };
    }

    await db
      .update(schema.jobs)
      .set({ contextSnapshot: contextResult.context.toSnapshot() })
      .where(
        and(eq(schema.jobs.id, claimedJob.id), eq(schema.jobs.projectId, claimedJob.projectId)),
      );

    for (const targetLocale of parsedInput.data.targetLocales) {
      await captureAnalysis({
        organizationId: project.organizationId,
        projectId: claimedJob.projectId,
        jobId: claimedJob.id,
        sourceLocale: parsedInput.data.sourceLocale,
        targetLocale,
        sourceEntries: {
          [parsedInput.data.translationKeyId ?? "source"]: parsedInput.data.sourceText,
        },
      });
    }
    await captureJobStatus({
      jobId: claimedJob.id,
      status: "running",
      operationKey: `status:${claimedJob.workflowRunId}:running`,
    });

    if (translateStringJobOverride) {
      const result = await translateStringJobOverride(
        contextResult.context.toStringTranslationInput(
          contextResult.context.project.name,
          contextResult.context.project.translationContext,
        ),
      );

      return { ok: true, result };
    }

    const organizationGenerator = await this.modelResolver.resolve(claimedJob.projectId);
    if (!organizationGenerator.ok) {
      return {
        ok: false,
        code: organizationGenerator.code,
        message: organizationGenerator.message,
      };
    }

    const result = await organizationGenerator.translateStringJob({
      ...contextResult.context.toStringTranslationInput(
        organizationGenerator.project.name,
        organizationGenerator.project.translationContext,
      ),
      reporting: {
        organizationId: project.organizationId,
        projectId: claimedJob.projectId,
        jobId: claimedJob.id,
      },
    });

    return { ok: true, result };
  }
}

class TranslationJobCompletionService {
  constructor(private readonly repository = new TranslationJobRepository()) {}

  async complete(input: {
    jobId: string;
    projectId: string;
    workflowRunId: string;
    result: StringTranslationJobResult;
  }) {
    const didSucceed = await db.transaction(async (tx) => {
      const [updatedJob] = await tx
        .update(schema.jobs)
        .set({
          status: "succeeded",
          outcomePayload: input.result,
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
        .set({ outcomeKind: "string_result" })
        .where(eq(schema.translationJobDetails.jobId, input.jobId));

      return true;
    });

    if (!didSucceed) {
      throw new Error(
        `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
      );
    }

    const parsedInput = stringTranslationJobInputSchema.safeParse(
      (await this.repository.getStored(input.jobId, input.projectId))?.inputPayload,
    );

    if (parsedInput.success) {
      const [project] = await db
        .select({ organizationId: schema.projects.organizationId })
        .from(schema.projects)
        .where(eq(schema.projects.id, input.projectId))
        .limit(1);

      if (project?.organizationId) {
        try {
          await persistStringJobTranslations({
            organizationId: project.organizationId,
            projectId: input.projectId,
            jobId: input.jobId,
            sourceLocale: parsedInput.data.sourceLocale,
            translations: input.result.translations,
            translationKeyId: parsedInput.data.translationKeyId,
          });
        } catch (error) {
          console.warn("[translation-job] string translation persistence failed", {
            jobId: input.jobId,
            projectId: input.projectId,
            organizationId: project.organizationId,
            translationKeyId: parsedInput.data.translationKeyId,
            error,
          });
        }
      }
    }

    const operationKey = `job:${input.jobId}:translation_jobs`;
    const [reportingJob] = await db
      .select({ organizationId: schema.jobs.organizationId })
      .from(schema.jobs)
      .where(eq(schema.jobs.id, input.jobId));
    if (reportingJob && parsedInput.success)
      for (const translation of input.result.translations) {
        await captureCompletions({
          organizationId: reportingJob.organizationId,
          jobId: input.jobId,
          targetLocale: translation.locale,
          sourceEntries: {
            [parsedInput.data.translationKeyId ?? "source"]: parsedInput.data.sourceText,
          },
          provenance: "automated",
        });
      }
    await captureJobStatus({
      jobId: input.jobId,
      status: "succeeded",
      operationKey: `status:${input.workflowRunId}:succeeded`,
    });
    const tokenUsage = input.result.tokenUsage;
    const [projectForUsage] = await db
      .select({ organizationId: schema.projects.organizationId })
      .from(schema.projects)
      .where(eq(schema.projects.id, input.projectId))
      .limit(1);
    const organizationId = projectForUsage?.organizationId;
    if (!organizationId) {
      throw new Error(`translation job ${input.jobId} has no organization for usage tracking`);
    }

    const trackUsageResult = await completeAndTrackBillableUsage({
      organizationId,
      operationKey,
      autumnEventName: "translation_job.completed",
      unit: "job",
      tokenUsage: tokenUsage ?? null,
      jobId: input.jobId,
      aiCreditSource: "translation_job_complete",
    });
    if (isErr(trackUsageResult)) {
      console.error("[translation-job] Autumn usage tracking failed after job succeeded", {
        jobId: input.jobId,
        operationKey,
        error: formatUsageControlError(trackUsageResult.error),
      });
    }

    const succeededJob = await this.repository.getStored(input.jobId, input.projectId);
    if (!succeededJob) {
      throw new Error(`translation job ${input.jobId} was not found in project ${input.projectId}`);
    }

    return succeededJob;
  }

  async fail(input: {
    jobId: string;
    projectId: string;
    workflowRunId: string;
    code: string;
    message: string;
  }) {
    const didFail = await db.transaction(async (tx) => {
      const [updatedJob] = await tx
        .update(schema.jobs)
        .set({
          status: "failed",
          outcomePayload: { code: input.code, message: input.message },
          lastError: input.message,
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
        .set({ outcomeKind: "error" })
        .where(eq(schema.translationJobDetails.jobId, input.jobId));

      return true;
    });

    if (!didFail) {
      throw new Error(
        `translation job ${input.jobId} is not owned by workflow run ${input.workflowRunId}`,
      );
    }

    serverAnalytics.track(PRODUCT_USAGE_ANALYTICS_EVENTS.translationJobFailed, {
      status: "failed",
      source: "translation_job",
    });

    const failedJob = await this.repository.getStored(input.jobId, input.projectId);
    if (!failedJob) {
      throw new Error(`translation job ${input.jobId} was not found in project ${input.projectId}`);
    }

    return failedJob;
  }
}

export class TranslationJobService {
  private readonly repository = new TranslationJobRepository();
  private readonly executor = new TranslationJobExecutor();
  private readonly completion = new TranslationJobCompletionService();

  claim(input: { event: TranslationJobEventData; runId: string }) {
    return this.repository.claim(input);
  }

  execute(
    claimedJob: ClaimedTranslationJob,
    translateStringJobOverride?: StringTranslationGenerator,
  ) {
    return this.executor.execute(claimedJob, translateStringJobOverride);
  }

  complete(input: {
    jobId: string;
    projectId: string;
    workflowRunId: string;
    result: StringTranslationJobResult;
  }) {
    return this.completion.complete(input);
  }

  fail(input: {
    jobId: string;
    projectId: string;
    workflowRunId: string;
    code: string;
    message: string;
  }) {
    return this.completion.fail(input);
  }
}

const defaultJobService = new TranslationJobService();

export async function claimTranslationJob(input: {
  event: TranslationJobEventData;
  runId: string;
}) {
  return defaultJobService.claim(input);
}

export async function executeClaimedTranslationJob(
  claimedJob: ClaimedTranslationJob,
  translateStringJobOverride?: StringTranslationGenerator,
) {
  return defaultJobService.execute(claimedJob, translateStringJobOverride);
}

export async function completeTranslationJob(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  result: StringTranslationJobResult;
}) {
  return defaultJobService.complete(input);
}

export async function failTranslationJob(input: {
  jobId: string;
  projectId: string;
  workflowRunId: string;
  code: string;
  message: string;
}) {
  return defaultJobService.fail(input);
}

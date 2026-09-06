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
import "dotenv/config";

import { randomUUID } from "node:crypto";

import { and, eq } from "drizzle-orm";
import { afterEach, beforeAll, describe, expect, it } from "vite-plus/test";

import { createProjectTestFixture } from "@/api/routes/project/project.fixture";
import { db, schema } from "@/lib/database/client";
import { ensureRepositorySourceFile } from "@/lib/file-storage/records";
import { saveNativeProjectContentEditorTranslation } from "@/lib/projects/content-editor/native-content-editor-service";
import { upsertProjectTranslationKeysFromEntries } from "@/lib/projects/translations/project-translation-service";
import { normalizeTranslationMemorySourceText } from "@/lib/translation/normalizeTranslationMemorySourceText";

import { captureAiUsage } from "./ai-cost";
import { bestReportingMatchScore, captureAnalysis, sourceSimilarity } from "./capture";

const projectFixture = createProjectTestFixture();
const SOURCE_PATH = "locales/en.json";

beforeAll(async () => {
  await db.$client.query("select 1");
});

afterEach(async () => {
  await projectFixture.cleanup();
});

async function createProjectWithTranslationKey() {
  const { organization, project, user } = await projectFixture.createStoredProjectFixture();
  const sourceFile = await ensureRepositorySourceFile({
    organizationId: organization.id,
    projectId: project.id,
    sourcePath: SOURCE_PATH,
  });
  await upsertProjectTranslationKeysFromEntries({
    organizationId: organization.id,
    projectId: project.id,
    repositorySourceFileId: sourceFile.id,
    entries: [{ key: "greeting", text: "Hello world", context: null }],
  });
  const [translationKey] = await db
    .select({
      id: schema.projectTranslationKeys.id,
      sourceText: schema.projectTranslationKeys.sourceText,
    })
    .from(schema.projectTranslationKeys)
    .where(eq(schema.projectTranslationKeys.repositorySourceFileId, sourceFile.id))
    .limit(1);
  return { organization, project, user, translationKey: translationKey! };
}

describe("reporting capture", () => {
  it("scores source similarity without requiring an exact match", () => {
    expect(sourceSimilarity("Hello world", "Hello world")).toBe(100);
    expect(sourceSimilarity("Hello world", "Hello worlds")).toBeGreaterThan(80);
    expect(sourceSimilarity("Hello", "")).toBe(0);
  });

  it("preserves existing CAT job linkage and records billable work", async () => {
    const { organization, project, user, translationKey } = await createProjectWithTranslationKey();
    const [job] = await db
      .insert(schema.jobs)
      .values({
        id: `job_${randomUUID()}`,
        organizationId: organization.id,
        projectId: project.id,
        createdByUserId: user.id,
        kind: "translation",
        status: "running",
        workflowRunId: `run_${randomUUID()}`,
        inputPayload: {},
      })
      .returning();
    await db.insert(schema.projectTranslations).values({
      organizationId: organization.id,
      projectId: project.id,
      translationKeyId: translationKey.id,
      targetLocale: "fr-FR",
      text: "Bonjour",
      status: "draft",
      provenance: "translation_job",
      sourceJobId: job.id,
    });

    const saved = await saveNativeProjectContentEditorTranslation({
      organizationId: organization.id,
      projectId: project.id,
      sourcePath: SOURCE_PATH,
      targetLocale: "fr-FR",
      translationKeyId: translationKey.id,
      text: "Bonjour le monde",
      actorUserId: user.id,
    });
    expect(saved?.status).toBe("draft");

    const [translation] = await db
      .select({
        sourceJobId: schema.projectTranslations.sourceJobId,
        provenance: schema.projectTranslations.provenance,
      })
      .from(schema.projectTranslations)
      .where(
        and(
          eq(schema.projectTranslations.translationKeyId, translationKey.id),
          eq(schema.projectTranslations.targetLocale, "fr-FR"),
        ),
      );
    expect(translation).toMatchObject({
      sourceJobId: job.id,
      provenance: "manual",
    });

    const [analysis] = await db
      .select({
        jobId: schema.reportingAnalyses.jobId,
        billable: schema.reportingAnalyses.billable,
        isCurrent: schema.reportingAnalyses.isCurrent,
      })
      .from(schema.reportingAnalyses)
      .where(
        and(
          eq(schema.reportingAnalyses.organizationId, organization.id),
          eq(schema.reportingAnalyses.segmentId, translationKey.id),
          eq(schema.reportingAnalyses.jobId, job.id),
        ),
      );
    expect(analysis).toMatchObject({
      jobId: job.id,
      billable: true,
      isCurrent: true,
    });

    const completions = await db
      .select({ id: schema.reportingActivity.id })
      .from(schema.reportingActivity)
      .where(
        and(
          eq(schema.reportingActivity.organizationId, organization.id),
          eq(schema.reportingActivity.kind, "completion"),
          eq(schema.reportingActivity.jobId, job.id),
        ),
      );
    expect(completions).toHaveLength(1);
  });

  it("records manual CAT work without a job instead of skipping reporting", async () => {
    const { organization, project, user, translationKey } = await createProjectWithTranslationKey();

    await saveNativeProjectContentEditorTranslation({
      organizationId: organization.id,
      projectId: project.id,
      sourcePath: SOURCE_PATH,
      targetLocale: "fr-FR",
      translationKeyId: translationKey.id,
      text: "Bonjour",
      actorUserId: user.id,
    });

    const analyses = await db
      .select({
        jobId: schema.reportingAnalyses.jobId,
        billable: schema.reportingAnalyses.billable,
      })
      .from(schema.reportingAnalyses)
      .where(
        and(
          eq(schema.reportingAnalyses.organizationId, organization.id),
          eq(schema.reportingAnalyses.segmentId, translationKey.id),
        ),
      );
    expect(analyses).toEqual([{ jobId: null, billable: true }]);

    await saveNativeProjectContentEditorTranslation({
      organizationId: organization.id,
      projectId: project.id,
      sourcePath: SOURCE_PATH,
      targetLocale: "fr-FR",
      translationKeyId: translationKey.id,
      text: "Bonjour encore",
      actorUserId: user.id,
    });
    const afterRetry = await db
      .select({
        id: schema.reportingAnalyses.id,
        isCurrent: schema.reportingAnalyses.isCurrent,
      })
      .from(schema.reportingAnalyses)
      .where(
        and(
          eq(schema.reportingAnalyses.organizationId, organization.id),
          eq(schema.reportingAnalyses.segmentId, translationKey.id),
        ),
      );
    expect(afterRetry).toHaveLength(1);
    expect(afterRetry[0]?.isCurrent).toBe(true);

    const completions = await db
      .select({ id: schema.reportingActivity.id })
      .from(schema.reportingActivity)
      .where(
        and(
          eq(schema.reportingActivity.organizationId, organization.id),
          eq(schema.reportingActivity.kind, "completion"),
        ),
      );
    expect(completions).toHaveLength(1);
    const costs = await db
      .select({ id: schema.reportingCosts.id })
      .from(schema.reportingCosts)
      .where(eq(schema.reportingCosts.organizationId, organization.id));
    expect(costs).toHaveLength(1);
  });

  it("matches approved memory entries with indexed lookup instead of a full scan", async () => {
    const { organization, project, user } = await projectFixture.createStoredProjectFixture();
    const [memory] = await db
      .insert(schema.memories)
      .values({
        organizationId: organization.id,
        createdByUserId: user.id,
        name: "Reporting TM",
      })
      .returning();
    await db.insert(schema.projectMemories).values({
      organizationId: organization.id,
      projectId: project.id,
      memoryId: memory.id,
    });
    await db.insert(schema.memoryEntries).values({
      memoryId: memory.id,
      sourceLocale: "en-US",
      targetLocale: "fr-FR",
      sourceText: "Hello world",
      normalizedSourceText: normalizeTranslationMemorySourceText("Hello world"),
      targetText: "Bonjour le monde",
      reviewStatus: "approved",
    });
    await db.insert(schema.memoryEntries).values({
      memoryId: memory.id,
      sourceLocale: "en-US",
      targetLocale: "fr-FR",
      sourceText: "Hello worlds",
      normalizedSourceText: normalizeTranslationMemorySourceText("Hello worlds"),
      targetText: "Bonjour les mondes",
      reviewStatus: "approved",
    });

    expect(
      await bestReportingMatchScore({
        memoryIds: [memory.id],
        sourceLocale: "en-US",
        targetLocale: "fr-FR",
        sourceText: "Hello world",
      }),
    ).toBe(100);
    expect(
      await bestReportingMatchScore({
        memoryIds: [memory.id],
        sourceLocale: "en-US",
        targetLocale: "fr-FR",
        sourceText: "Hello worlds!",
      }),
    ).toBeGreaterThan(80);

    await captureAnalysis({
      organizationId: organization.id,
      projectId: project.id,
      sourceLocale: "en-US",
      targetLocale: "fr-FR",
      sourceEntries: { greeting: "Hello world", other: "Hello worlds!" },
      billable: true,
    });
    const rows = await db
      .select({
        segmentId: schema.reportingAnalyses.segmentId,
        matchScore: schema.reportingAnalyses.matchScore,
      })
      .from(schema.reportingAnalyses)
      .where(eq(schema.reportingAnalyses.organizationId, organization.id));
    expect(rows.find((row) => row.segmentId === "greeting")?.matchScore).toBe(100);
    expect(rows.find((row) => row.segmentId === "other")?.matchScore).toBeGreaterThan(80);
    expect(
      await bestReportingMatchScore({
        memoryIds: [memory.id],
        sourceLocale: "en-US",
        targetLocale: "fr-FR",
        sourceText: "Hello brave world",
      }),
    ).toBeGreaterThan(50);
  });

  it("keeps analysis capture from failing the caller", async () => {
    await expect(
      captureAnalysis({
        organizationId: randomUUID(),
        projectId: `project_${randomUUID()}`,
        sourceLocale: "en-US",
        targetLocale: "fr-FR",
        sourceEntries: { source: "Hello" },
      }),
    ).resolves.toBeUndefined();
  });

  it("keeps usage capture from failing the caller", async () => {
    await expect(
      captureAiUsage({
        organizationId: randomUUID(),
        invocationId: `usage_${randomUUID()}`,
        provider: "openai",
        model: "gpt-4o",
        inputTokens: 10,
        outputTokens: 4,
      }),
    ).resolves.toBeUndefined();
  });
});

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
import { createHash } from "node:crypto";
import { and, eq, desc, inArray, isNull, sql } from "drizzle-orm";
import { db, schema, type DatabaseClient } from "@/lib/database/client";
import { listAttachedProjectMemoryIds } from "@/lib/memory/ensure-default-native-project-memory";
import { normalizeTranslationMemorySourceText } from "@/lib/translation/normalizeTranslationMemorySourceText";
import { buildTranslationMemoryTsQuery } from "@/lib/translation/translation-memory-ts-query";
import { countSourceWords, matchBucket, WORD_COUNT_VERSION } from "./word-analysis";
import { wordCost } from "./money";

const MEMORY_MATCH_CANDIDATE_LIMIT = 32;

export async function reportingStart(database: DatabaseClient = db) {
  await database.insert(schema.reportingRollout).values({ id: 1 }).onConflictDoNothing();
  const [rollout] = await database.select().from(schema.reportingRollout).limit(1);
  return rollout.startedAt;
}

export async function resolveReportingRate(
  input: {
    organizationId: string;
    projectId: string;
    jobId?: string;
    sourceLocale: string;
    targetLocale: string;
    step: "translation" | "review";
  },
  database: DatabaseClient = db,
) {
  const [override] = input.jobId
    ? await database
        .select()
        .from(schema.reportingTaskRates)
        .where(
          and(
            eq(schema.reportingTaskRates.organizationId, input.organizationId),
            eq(schema.reportingTaskRates.jobId, input.jobId),
            eq(schema.reportingTaskRates.step, input.step),
          ),
        )
    : [];
  if (override?.rateId) {
    const [rate] = await database
      .select()
      .from(schema.reportingRates)
      .where(
        and(
          eq(schema.reportingRates.id, override.rateId),
          eq(schema.reportingRates.organizationId, input.organizationId),
        ),
      );
    return rate ?? null;
  }
  const [budget] = await database
    .select()
    .from(schema.reportingBudgets)
    .where(
      and(
        eq(schema.reportingBudgets.organizationId, input.organizationId),
        eq(schema.reportingBudgets.projectId, input.projectId),
      ),
    );
  if (!budget?.rateCardName) return null;
  const [rate] = await database
    .select()
    .from(schema.reportingRates)
    .where(
      and(
        eq(schema.reportingRates.organizationId, input.organizationId),
        eq(schema.reportingRates.name, budget.rateCardName),
        eq(schema.reportingRates.sourceLocale, input.sourceLocale),
        eq(schema.reportingRates.targetLocale, input.targetLocale),
        eq(schema.reportingRates.step, input.step),
      ),
    )
    .orderBy(desc(schema.reportingRates.createdAt), desc(schema.reportingRates.id))
    .limit(1);
  return rate ?? null;
}

/** Character edit similarity; the persisted algorithm version fixes its meaning. */
export function sourceSimilarity(left: string, right: string): number {
  const a = Array.from(left.normalize("NFC").trim()),
    b = Array.from(right.normalize("NFC").trim());
  if (a.join("") === b.join("")) return 100;
  if (!a.length || !b.length) return 0;
  let previous = Array.from({ length: b.length + 1 }, (_, i) => i);
  for (let i = 1; i <= a.length; i++) {
    const current = [i];
    for (let j = 1; j <= b.length; j++)
      current[j] = Math.min(
        current[j - 1] + 1,
        previous[j] + 1,
        previous[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1),
      );
    previous = current;
  }
  return Math.floor(100 * (1 - previous[b.length] / Math.max(a.length, b.length)));
}

export async function bestReportingMatchScore(input: {
  memoryIds: string[];
  sourceLocale: string;
  targetLocale: string;
  sourceText: string;
}): Promise<number> {
  if (!input.memoryIds.length) return 0;
  const normalized = normalizeTranslationMemorySourceText(input.sourceText);
  const [exact] = await db
    .select({ id: schema.memoryEntries.id })
    .from(schema.memoryEntries)
    .where(
      and(
        inArray(schema.memoryEntries.memoryId, input.memoryIds),
        eq(schema.memoryEntries.normalizedSourceText, normalized),
        eq(schema.memoryEntries.sourceLocale, input.sourceLocale),
        eq(schema.memoryEntries.targetLocale, input.targetLocale),
        eq(schema.memoryEntries.reviewStatus, "approved"),
      ),
    )
    .limit(1);
  if (exact) return 100;
  const tsQuery = buildTranslationMemoryTsQuery(input.sourceText);
  if (!tsQuery) return 0;
  const candidates = await db
    .select({ source: schema.memoryEntries.sourceText })
    .from(schema.memoryEntries)
    .where(
      and(
        inArray(schema.memoryEntries.memoryId, input.memoryIds),
        eq(schema.memoryEntries.sourceLocale, input.sourceLocale),
        eq(schema.memoryEntries.targetLocale, input.targetLocale),
        eq(schema.memoryEntries.reviewStatus, "approved"),
        sql`${schema.memoryEntries.searchVector} @@ to_tsquery('simple', ${tsQuery})`,
      ),
    )
    .orderBy(
      desc(
        sql`ts_rank(${schema.memoryEntries.searchVector}, to_tsquery('simple', ${tsQuery}))`,
      ),
    )
    .limit(MEMORY_MATCH_CANDIDATE_LIMIT);
  return candidates.reduce(
    (best, candidate) => Math.max(best, sourceSimilarity(input.sourceText, candidate.source)),
    0,
  );
}

export async function captureAnalysis(input: {
  organizationId: string;
  projectId: string;
  jobId?: string;
  sourceLocale: string;
  targetLocale: string;
  sourceEntries: Record<string, string>;
  step?: "translation" | "review";
  billable?: boolean;
}) {
  await reportingStart();
  if (input.jobId) {
    const [job] = await db
      .select({ id: schema.jobs.id })
      .from(schema.jobs)
      .leftJoin(schema.externalJobDetails, eq(schema.externalJobDetails.jobId, schema.jobs.id))
      .where(
        and(
          eq(schema.jobs.id, input.jobId),
          eq(schema.jobs.organizationId, input.organizationId),
          eq(schema.jobs.projectId, input.projectId),
          isNull(schema.externalJobDetails.jobId),
        ),
      );
    if (!job) return;
  }
  const step = input.step ?? "translation";
  const rate = await resolveReportingRate({ ...input, step });
  let memoryIds: string[] | null = null;
  try {
    memoryIds = await listAttachedProjectMemoryIds(input.projectId);
  } catch {
    console.warn("reporting_match_analysis_unavailable", { jobId: input.jobId });
  }
  const uniqueTexts = [...new Set(Object.values(input.sourceEntries))];
  const scores = new Map<string, number | null>();
  if (memoryIds)
    for (const sourceText of uniqueTexts)
      scores.set(
        sourceText,
        await bestReportingMatchScore({
          memoryIds,
          sourceLocale: input.sourceLocale,
          targetLocale: input.targetLocale,
          sourceText,
        }),
      );
  const seen = new Set<string>();
  await db.transaction(async (tx) => {
    if (input.jobId)
      await tx
        .select({ id: schema.jobs.id })
        .from(schema.jobs)
        .where(eq(schema.jobs.id, input.jobId))
        .for("update");
    for (const [segmentId, sourceText] of Object.entries(input.sourceEntries)) {
      const normalized = sourceText.normalize("NFC").trim();
      const repetition = seen.has(normalized);
      seen.add(normalized);
      const score = memoryIds ? (scores.get(sourceText) ?? 0) : null;
      const identity = and(
        input.jobId
          ? eq(schema.reportingAnalyses.jobId, input.jobId)
          : isNull(schema.reportingAnalyses.jobId),
        eq(schema.reportingAnalyses.organizationId, input.organizationId),
        eq(schema.reportingAnalyses.segmentId, segmentId),
        eq(schema.reportingAnalyses.targetLocale, input.targetLocale),
        eq(schema.reportingAnalyses.step, step),
      );
      await tx.update(schema.reportingAnalyses).set({ isCurrent: false }).where(identity);
      await tx
        .insert(schema.reportingAnalyses)
        .values({
          organizationId: input.organizationId,
          projectId: input.projectId,
          jobId: input.jobId ?? null,
          step,
          segmentId,
          sourceRevision: createHash("sha256").update(sourceText).digest("hex"),
          sourceLocale: input.sourceLocale,
          targetLocale: input.targetLocale,
          words: countSourceWords(sourceText, input.sourceLocale),
          billable: input.billable ?? false,
          matchScore: score,
          bucket: matchBucket(score, repetition),
          algorithmVersion: WORD_COUNT_VERSION,
          rateId: rate?.id,
        })
        .onConflictDoUpdate({
          target: [
            schema.reportingAnalyses.organizationId,
            schema.reportingAnalyses.jobId,
            schema.reportingAnalyses.segmentId,
            schema.reportingAnalyses.sourceRevision,
            schema.reportingAnalyses.targetLocale,
            schema.reportingAnalyses.step,
          ],
          set: { isCurrent: true, ...(input.billable ? { billable: true } : {}) },
        });
    }
  });
}

export async function captureCompletions(
  input: {
    organizationId: string;
    jobId?: string;
    targetLocale: string;
    sourceEntries: Record<string, string>;
    provenance: "human" | "automated";
    step?: "translation" | "review";
  },
  database: DatabaseClient = db,
) {
  const step = input.step ?? "translation";
  const analyses = await database
    .select()
    .from(schema.reportingAnalyses)
    .where(
      and(
        eq(schema.reportingAnalyses.organizationId, input.organizationId),
        input.jobId
          ? eq(schema.reportingAnalyses.jobId, input.jobId)
          : isNull(schema.reportingAnalyses.jobId),
        eq(schema.reportingAnalyses.targetLocale, input.targetLocale),
        eq(schema.reportingAnalyses.step, step),
        eq(schema.reportingAnalyses.isCurrent, true),
      ),
    );
  for (const analysis of analyses) {
    const source = input.sourceEntries[analysis.segmentId];
    if (
      source === undefined ||
      createHash("sha256").update(source).digest("hex") !== analysis.sourceRevision
    )
      continue;
    const operationKey = `completion:${analysis.id}:${step}`;
    await database
      .insert(schema.reportingActivity)
      .values({
        organizationId: input.organizationId,
        projectId: analysis.projectId,
        jobId: input.jobId ?? null,
        operationKey,
        kind: "completion",
        step,
        targetLocale: input.targetLocale,
        analysisId: analysis.id,
      })
      .onConflictDoNothing();
    if (input.provenance !== "human") continue;
    if (input.jobId) {
      const [taskRate] = await database
        .select()
        .from(schema.reportingTaskRates)
        .where(
          and(
            eq(schema.reportingTaskRates.jobId, input.jobId),
            eq(schema.reportingTaskRates.step, step),
          ),
        );
      if (taskRate?.overrideUsd !== null && taskRate?.overrideUsd !== undefined) continue;
    }
    const [rate] = analysis.rateId
      ? await database
          .select()
          .from(schema.reportingRates)
          .where(eq(schema.reportingRates.id, analysis.rateId))
      : [];
    if (rate?.basis === "hour") continue;
    await database
      .insert(schema.reportingCosts)
      .values({
        organizationId: input.organizationId,
        projectId: analysis.projectId,
        jobId: input.jobId ?? null,
        operationKey: `human:${analysis.id}:${step}`,
        kind: "human",
        step,
        targetLocale: input.targetLocale,
        rateId: rate?.id,
        amountUsd:
          rate && analysis.words !== null && analysis.bucket !== "unavailable"
            ? wordCost(rate.rate, analysis.words, rate.percentages[analysis.bucket] ?? 100)
            : null,
        basis:
          rate && analysis.words !== null && analysis.bucket !== "unavailable"
            ? "rate"
            : "unpriced",
      })
      .onConflictDoNothing();
  }
}

export async function captureJobStatus(input: {
  jobId: string;
  status: string;
  operationKey: string;
  durationMs?: number;
}) {
  await reportingStart();
  if (input.status === "succeeded") await captureTaskOverrides(input.jobId);
  const [job] = await db
    .select({
      organizationId: schema.jobs.organizationId,
      projectId: schema.jobs.projectId,
      kind: schema.jobs.kind,
    })
    .from(schema.jobs)
    .leftJoin(schema.externalJobDetails, eq(schema.externalJobDetails.jobId, schema.jobs.id))
    .where(and(eq(schema.jobs.id, input.jobId), isNull(schema.externalJobDetails.jobId)));
  if (!job) return;
  await db
    .insert(schema.reportingActivity)
    .values({
      ...job,
      jobId: input.jobId,
      kind: "status",
      step: job.kind,
      status: input.status,
      operationKey: input.operationKey,
      durationMs: input.durationMs,
    })
    .onConflictDoNothing();
}

/** Fixed task fees are recognized once, at job success, never at quote creation. */
export async function captureTaskOverrides(jobId: string, database: DatabaseClient = db) {
  await database.execute(sql`
    insert into reporting_costs (organization_id, project_id, job_id, operation_key, kind, step, amount_usd, basis, created_at)
    select r.organization_id, j.project_id, j.id, 'task-override:' || r.id, 'human', r.step, r.override_usd, 'manual', coalesce(j.completed_at, now())
    from reporting_task_rates r join jobs j on j.id=r.job_id
    where j.id=${jobId} and j.status='succeeded' and r.override_usd is not null
    on conflict (organization_id, operation_key) do nothing
  `);
}

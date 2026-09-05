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
import { getUsage } from "tokenlens";
import { eq, and } from "drizzle-orm";
import { db, schema } from "@/lib/database/client";
import { reportingStart } from "./capture";

export const AI_PRICING_VERSION = "tokenlens-1.3.1-v1";
export type AiReportingUsage = {
  invocationId: string;
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  tokenCategories?: Record<string, number>;
  reportedUsd?: string;
  usageUnavailable?: boolean;
};
export async function captureAiUsage(
  input: AiReportingUsage & {
    organizationId: string;
    projectId?: string | null;
    jobId?: string | null;
    step?: string;
    targetLocale?: string;
  },
) {
  await reportingStart();
  if (input.jobId) {
    const [external] = await db
      .select({ id: schema.externalJobDetails.jobId })
      .from(schema.externalJobDetails)
      .where(
        and(
          eq(schema.externalJobDetails.jobId, input.jobId),
          eq(schema.externalJobDetails.organizationId, input.organizationId),
        ),
      );
    if (external) return;
  }
  const estimate = getUsage({
    modelId: input.model,
    usage: { input: input.inputTokens, output: input.outputTokens },
  }).costUSD?.totalUSD;
  const amountUsd =
    input.reportedUsd ??
    (!input.usageUnavailable && estimate !== undefined && Number.isFinite(estimate)
      ? estimate.toFixed(8)
      : null);
  await db
    .insert(schema.reportingCosts)
    .values({
      organizationId: input.organizationId,
      projectId: input.projectId,
      jobId: input.jobId,
      operationKey: `ai:${input.invocationId}`,
      kind: "ai",
      step: input.step ?? "translation",
      targetLocale: input.targetLocale,
      amountUsd,
      basis:
        input.reportedUsd !== undefined
          ? "reported"
          : amountUsd !== null
            ? "estimated"
            : "unpriced",
      provider: input.provider,
      model: input.model,
      inputTokens: input.inputTokens,
      outputTokens: input.outputTokens,
      tokenCategories: input.tokenCategories,
      pricingVersion:
        amountUsd !== null && input.reportedUsd === undefined ? AI_PRICING_VERSION : null,
    })
    .onConflictDoNothing();
}

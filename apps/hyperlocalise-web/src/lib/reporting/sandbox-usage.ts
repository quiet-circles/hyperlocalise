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
import { z } from "zod";
import { safeJsonParse } from "@/lib/primitives/safeJsonParse/safeJsonParse";
import { isErr } from "@/lib/primitives/result/results";
import { captureAiUsage } from "./ai-cost";

const usageSchema = z.object({
  inputTokens: z.number().int().nonnegative(),
  outputTokens: z.number().int().nonnegative(),
  cachedInputTokens: z.number().int().nonnegative().optional(),
  reasoningTokens: z.number().int().nonnegative().optional(),
});
const reportSchema = usageSchema.extend({
  localeUsage: z.record(z.string(), usageSchema).optional(),
});

export function sandboxUsageRows(report: string | null) {
  const json = safeJsonParse(report ?? "");
  const parsed = isErr(json) ? null : reportSchema.safeParse(json.value);
  if (!parsed?.success)
    return [{ inputTokens: 0, outputTokens: 0, usageUnavailable: true, targetLocale: undefined }];
  const total = parsed.data;
  const locales = Object.entries(total.localeUsage ?? {});
  // Locale subtotals exclude shared context generation. Keep that remainder visible.
  const input = locales.reduce((sum, [, value]) => sum + value.inputTokens, 0);
  const output = locales.reduce((sum, [, value]) => sum + value.outputTokens, 0);
  if (input > total.inputTokens || output > total.outputTokens)
    return [{ ...total, targetLocale: undefined }];
  const rows = locales.map(([targetLocale, value]) => ({
    ...value,
    targetLocale: targetLocale as string | undefined,
  }));
  if (!locales.length || input < total.inputTokens || output < total.outputTokens)
    rows.push({
      inputTokens: total.inputTokens - input,
      outputTokens: total.outputTokens - output,
      targetLocale: undefined,
    });
  return rows;
}

export async function captureSandboxUsage(input: {
  organizationId: string;
  projectId: string;
  jobId: string;
  invocationId: string;
  provider: string;
  model: string;
  report: string | null;
}) {
  for (const row of sandboxUsageRows(input.report)) {
    await captureAiUsage({
      ...input,
      ...row,
      invocationId: `${input.invocationId}:${row.targetLocale ?? "unallocated"}`,
    });
  }
}

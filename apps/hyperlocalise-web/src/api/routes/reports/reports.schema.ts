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
import { MATCH_BUCKETS } from "@/lib/reporting/word-analysis";
const usd = z.string().regex(/^\d{1,12}(\.\d{1,8})?$/);
const step = z.enum(["translation", "review"]);
const day = z.iso.date();
export const reportQuerySchema = z
  .object({
    from: day.optional(),
    to: day.optional(),
    projectId: z.string().optional(),
    jobId: z.string().optional(),
    targetLocale: z.string().optional(),
    step: step.optional(),
    interval: z.enum(["day", "week"]).default("week"),
    view: z.enum(["overview", "words", "time", "costs"]).default("overview"),
    format: z.enum(["json", "csv"]).default("json"),
  })
  .refine((v) => !v.from || !v.to || v.from <= v.to, { message: "Invalid date range" });
export const rateSchema = z.object({
  name: z.string().trim().min(1).max(100),
  sourceLocale: z.string().min(2).max(35),
  targetLocale: z.string().min(2).max(35),
  step,
  basis: z.enum(["word", "hour"]),
  rate: usd,
  percentages: z.partialRecord(z.enum(MATCH_BUCKETS), z.number().int().min(0).max(100)).default({}),
});
export const budgetSchema = z.object({
  projectId: z.string().min(1),
  budget: usd,
  rateCardName: z.string().max(100).nullable().default(null),
});
export const taskRateSchema = z.object({
  jobId: z.string().min(1),
  step,
  rateId: z.uuid().nullable().default(null),
  estimatedMinutes: z.number().int().nonnegative().max(525600).nullable().default(null),
  overrideUsd: usd.nullable().default(null),
});
export const timeSchema = z.object({
  jobId: z.string().min(1),
  step,
  targetLocale: z.string().min(2).max(35),
  workDate: day,
  minutes: z.number().int().min(1).max(1440),
  note: z.string().max(2000).optional(),
});
export const expenseSchema = z.object({
  projectId: z.string().min(1),
  jobId: z.string().optional(),
  step,
  amountUsd: usd,
  note: z.string().trim().min(1).max(2000),
  operationKey: z.uuid(),
});

export const timeUpdateSchema = timeSchema.pick({ minutes: true, workDate: true, note: true });

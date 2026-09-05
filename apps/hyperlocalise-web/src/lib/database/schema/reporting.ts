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
import { sql } from "drizzle-orm";
import {
  pgTable,
  uuid,
  text,
  integer,
  timestamp,
  numeric,
  jsonb,
  index,
  uniqueIndex,
  boolean,
} from "drizzle-orm/pg-core";
import { organizations, users } from "./organizations";
import { projects } from "./projects";
import { jobs } from "./jobs";
import type { MatchBucket } from "@/lib/reporting/word-analysis";

const createdAt = () => timestamp("created_at", { withTimezone: true }).notNull().defaultNow();
const organizationId = () =>
  uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" });
const projectId = () => text("project_id").references(() => projects.id, { onDelete: "set null" });
const jobId = () => text("job_id").references(() => jobs.id, { onDelete: "set null" });
const amount = (name: string) => numeric(name, { precision: 24, scale: 8 });

/** A database default records rollout time without reconstructing prior activity. */
export const reportingRollout = pgTable("reporting_rollout", {
  id: integer("id").primaryKey().default(1),
  startedAt: timestamp("started_at", { withTimezone: true }).notNull().defaultNow(),
});
export const reportingRates = pgTable(
  "reporting_rates",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    name: text("name").notNull(),
    sourceLocale: text("source_locale").notNull(),
    targetLocale: text("target_locale").notNull(),
    step: text("step", { enum: ["translation", "review"] }).notNull(),
    basis: text("basis", { enum: ["word", "hour"] }).notNull(),
    rate: amount("rate").notNull(),
    percentages: jsonb("percentages")
      .$type<Partial<Record<MatchBucket, number>>>()
      .notNull()
      .default(sql`'{}'::jsonb`),
    createdAt: createdAt(),
  },
  (t) => [index("reporting_rates_org").on(t.organizationId)],
);
export const reportingBudgets = pgTable(
  "reporting_budgets",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    projectId: projectId(),
    budget: amount("budget").notNull(),
    rateCardName: text("rate_card_name"),
    createdAt: createdAt(),
  },
  (t) => [uniqueIndex("reporting_budgets_project").on(t.organizationId, t.projectId)],
);
export const reportingTaskRates = pgTable(
  "reporting_task_rates",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    jobId: jobId(),
    step: text("step", { enum: ["translation", "review"] }).notNull(),
    rateId: uuid("rate_id").references(() => reportingRates.id),
    estimatedMinutes: integer("estimated_minutes"),
    overrideUsd: amount("override_usd"),
    createdAt: createdAt(),
  },
  (t) => [uniqueIndex("reporting_task_rates_job_step").on(t.organizationId, t.jobId, t.step)],
);
export const reportingAnalyses = pgTable(
  "reporting_analyses",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    projectId: projectId(),
    jobId: jobId(),
    step: text("step", { enum: ["translation", "review"] })
      .notNull()
      .default("translation"),
    segmentId: text("segment_id").notNull(),
    sourceRevision: text("source_revision").notNull(),
    sourceLocale: text("source_locale").notNull(),
    targetLocale: text("target_locale").notNull(),
    words: integer("words"),
    billable: boolean("billable").notNull().default(false),
    isCurrent: boolean("is_current").notNull().default(true),
    matchScore: integer("match_score"),
    bucket: text("bucket").$type<MatchBucket>().notNull(),
    algorithmVersion: integer("algorithm_version").notNull().default(1),
    rateId: uuid("rate_id").references(() => reportingRates.id),
    createdAt: createdAt(),
  },
  (t) => [
    uniqueIndex("reporting_analysis_identity").on(
      t.organizationId,
      t.jobId,
      t.segmentId,
      t.sourceRevision,
      t.targetLocale,
      t.step,
    ),
    index("reporting_analysis_project").on(t.organizationId, t.projectId, t.createdAt),
  ],
);
export const reportingActivity = pgTable(
  "reporting_activity",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    projectId: projectId(),
    jobId: jobId(),
    operationKey: text("operation_key").notNull(),
    kind: text("kind", { enum: ["completion", "status", "workflow"] }).notNull(),
    step: text("step").notNull(),
    targetLocale: text("target_locale"),
    analysisId: uuid("analysis_id").references(() => reportingAnalyses.id),
    status: text("status"),
    durationMs: integer("duration_ms"),
    createdAt: createdAt(),
  },
  (t) => [
    uniqueIndex("reporting_activity_operation").on(t.organizationId, t.operationKey),
    index("reporting_activity_filter").on(t.organizationId, t.projectId, t.createdAt),
  ],
);
export const reportingTimeEntries = pgTable(
  "reporting_time_entries",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    projectId: projectId(),
    jobId: jobId(),
    contributorId: uuid("contributor_id")
      .notNull()
      .references(() => users.id),
    step: text("step", { enum: ["translation", "review"] }).notNull(),
    targetLocale: text("target_locale").notNull(),
    workDate: timestamp("work_date", { withTimezone: true }).notNull(),
    minutes: integer("minutes").notNull(),
    note: text("note"),
    rateId: uuid("rate_id").references(() => reportingRates.id),
    voided: boolean("voided").notNull().default(false),
    createdAt: createdAt(),
  },
  (t) => [index("reporting_time_filter").on(t.organizationId, t.projectId, t.workDate)],
);
export const reportingCosts = pgTable(
  "reporting_costs",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: organizationId(),
    projectId: projectId(),
    jobId: jobId(),
    operationKey: text("operation_key").notNull(),
    kind: text("kind", { enum: ["human", "ai", "expense"] }).notNull(),
    step: text("step").notNull(),
    targetLocale: text("target_locale"),
    amountUsd: amount("amount_usd"),
    basis: text("basis", {
      enum: ["rate", "reported", "estimated", "unpriced", "manual"],
    }).notNull(),
    rateId: uuid("rate_id").references(() => reportingRates.id),
    timeEntryId: uuid("time_entry_id").references(() => reportingTimeEntries.id),
    provider: text("provider"),
    model: text("model"),
    inputTokens: integer("input_tokens"),
    outputTokens: integer("output_tokens"),
    tokenCategories: jsonb("token_categories").$type<Record<string, number>>(),
    pricingVersion: text("pricing_version"),
    note: text("note"),
    voided: boolean("voided").notNull().default(false),
    createdAt: createdAt(),
  },
  (t) => [
    uniqueIndex("reporting_cost_operation").on(t.organizationId, t.operationKey),
    index("reporting_cost_filter").on(t.organizationId, t.projectId, t.createdAt),
  ],
);
export const reportingAudit = pgTable("reporting_audit", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: organizationId(),
  actorId: uuid("actor_id")
    .notNull()
    .references(() => users.id),
  resourceId: text("resource_id").notNull(),
  action: text("action").notNull(),
  before: jsonb("before"),
  after: jsonb("after"),
  createdAt: createdAt(),
});

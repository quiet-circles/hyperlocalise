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
import { Hono } from "hono";
import { and, eq, inArray, desc, sql } from "drizzle-orm";
import { workosAuthMiddleware, type AuthVariables } from "@/api/auth/workos";
import { hasCapability } from "@/api/auth/policy";
import { getAccessibleProjectIds, buildAccessibleJobsWhere } from "@/api/auth/team-access";
import { badRequestResponse, forbiddenResponse, notFoundResponse } from "@/api/response.schema";
import { db, schema } from "@/lib/database/client";
import { queryReport, reportCsv } from "@/lib/reporting/query";
import { hourlyCost } from "@/lib/reporting/money";
import { resolveReportingRate, reportingStart } from "@/lib/reporting/capture";
import { budgetSummary } from "@/lib/reporting/budgets";
import {
  reportQuerySchema,
  rateSchema,
  budgetSchema,
  taskRateSchema,
  timeSchema,
  timeUpdateSchema,
  expenseSchema,
} from "./reports.schema";

export function createReportsRoutes() {
  return new Hono<{ Variables: AuthVariables }>()
    .use("*", workosAuthMiddleware)
    .get("/", async (c) => {
      const parsed = reportQuerySchema.safeParse(c.req.query());
      if (!parsed.success) return badRequestResponse(c, "invalid_report_query");
      const financial = hasCapability(c.var.auth.membership.role, "billing:read");
      if (parsed.data.view === "costs" && !financial)
        return forbiddenResponse(c, "report_costs_forbidden");
      const report = await queryReport({
        organizationId: c.var.auth.organization.localOrganizationId,
        projectIds: await getAccessibleProjectIds(c.var.auth),
        financial,
        query: parsed.data,
      });
      if (parsed.data.format === "csv") {
        c.header("Content-Type", "text/csv; charset=utf-8");
        c.header("Content-Disposition", 'attachment; filename="translation-report.csv"');
        return c.body(reportCsv(report.rows, financial));
      }
      return c.json({ report });
    })
    .get("/settings", async (c) => {
      const organizationId = c.var.auth.organization.localOrganizationId;
      const projectIds = await getAccessibleProjectIds(c.var.auth);
      const projects = projectIds.length
        ? await db
            .select({ id: schema.projects.id, name: schema.projects.name })
            .from(schema.projects)
            .where(inArray(schema.projects.id, projectIds))
        : [];
      if (!hasCapability(c.var.auth.membership.role, "billing:read"))
        return c.json({
          settings: { projects, rates: [], budgets: [], financial: false, canManage: false },
        });
      const rates = await db
        .select()
        .from(schema.reportingRates)
        .where(eq(schema.reportingRates.organizationId, organizationId))
        .orderBy(desc(schema.reportingRates.createdAt));
      const budgets = await budgetSummary(organizationId, projectIds);
      return c.json({
        settings: {
          projects,
          rates,
          budgets,
          financial: true,
          canManage: hasCapability(c.var.auth.membership.role, "billing:write"),
        },
      });
    })
    .get("/tasks/:jobId", async (c) => {
      const organizationId = c.var.auth.organization.localOrganizationId;
      const [job] = await db
        .select()
        .from(schema.jobs)
        .where(
          and(eq(schema.jobs.id, c.req.param("jobId")), await buildAccessibleJobsWhere(c.var.auth)),
        );
      if (!job) return notFoundResponse(c, "job_not_found");
      const analyses = await db
        .select({
          id: schema.reportingAnalyses.id,
          segmentId: schema.reportingAnalyses.segmentId,
          sourceRevision: schema.reportingAnalyses.sourceRevision,
          targetLocale: schema.reportingAnalyses.targetLocale,
          words: schema.reportingAnalyses.words,
          bucket: schema.reportingAnalyses.bucket,
          matchScore: schema.reportingAnalyses.matchScore,
          createdAt: schema.reportingAnalyses.createdAt,
        })
        .from(schema.reportingAnalyses)
        .where(
          and(
            eq(schema.reportingAnalyses.organizationId, organizationId),
            eq(schema.reportingAnalyses.jobId, job.id),
          ),
        );
      const times = await db
        .select()
        .from(schema.reportingTimeEntries)
        .where(
          and(
            eq(schema.reportingTimeEntries.organizationId, organizationId),
            eq(schema.reportingTimeEntries.jobId, job.id),
            eq(schema.reportingTimeEntries.voided, false),
          ),
        );
      const financial = hasCapability(c.var.auth.membership.role, "billing:read");
      return c.json({
        taskReport: {
          analyses,
          timeEntries: times
            .filter((entry) => financial || entry.contributorId === c.var.auth.user.localUserId)
            .map(({ rateId: _rateId, ...entry }) => entry),
        },
      });
    })
    .post("/rates", async (c) => {
      if (!hasCapability(c.var.auth.membership.role, "billing:write"))
        return forbiddenResponse(c, "report_costs_forbidden");
      const parsed = rateSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_rate");
      const [rate] = await db
        .insert(schema.reportingRates)
        .values({ ...parsed.data, organizationId: c.var.auth.organization.localOrganizationId })
        .returning();
      return c.json({ rate }, 201);
    })
    .put("/budgets", async (c) => {
      if (!hasCapability(c.var.auth.membership.role, "billing:write"))
        return forbiddenResponse(c, "report_costs_forbidden");
      const parsed = budgetSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_budget");
      if (!(await getAccessibleProjectIds(c.var.auth)).includes(parsed.data.projectId))
        return notFoundResponse(c, "project_not_found");
      const [budget] = await db
        .insert(schema.reportingBudgets)
        .values({ ...parsed.data, organizationId: c.var.auth.organization.localOrganizationId })
        .onConflictDoUpdate({
          target: [schema.reportingBudgets.organizationId, schema.reportingBudgets.projectId],
          set: parsed.data,
        })
        .returning();
      return c.json({ budget });
    })
    .put("/task-rates", async (c) => {
      if (!hasCapability(c.var.auth.membership.role, "billing:write"))
        return forbiddenResponse(c, "report_costs_forbidden");
      const parsed = taskRateSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_task_rate");
      const organizationId = c.var.auth.organization.localOrganizationId;
      const [job] = await db
        .select()
        .from(schema.jobs)
        .where(
          and(eq(schema.jobs.id, parsed.data.jobId), await buildAccessibleJobsWhere(c.var.auth)),
        );
      if (!job) return notFoundResponse(c, "job_not_found");
      if (parsed.data.rateId) {
        const [rate] = await db
          .select()
          .from(schema.reportingRates)
          .where(
            and(
              eq(schema.reportingRates.id, parsed.data.rateId),
              eq(schema.reportingRates.organizationId, organizationId),
              eq(schema.reportingRates.step, parsed.data.step),
            ),
          );
        if (!rate) return badRequestResponse(c, "invalid_rate");
      }
      const taskRate = await db.transaction(async (tx) => {
        const [lockedJob] = await tx
          .select()
          .from(schema.jobs)
          .where(eq(schema.jobs.id, job.id))
          .for("update");
        const activity = await tx.execute<{ started: boolean }>(sql`
          select exists(select 1 from reporting_analyses where job_id=${job.id} and step=${parsed.data.step})
            or exists(select 1 from reporting_costs where job_id=${job.id} and step=${parsed.data.step})
            or exists(select 1 from reporting_time_entries where job_id=${job.id} and step=${parsed.data.step}) as started
        `);
        if (lockedJob.status !== "queued" || activity.rows[0].started) return null;
        const [saved] = await tx
          .insert(schema.reportingTaskRates)
          .values({ ...parsed.data, organizationId })
          .onConflictDoUpdate({
            target: [
              schema.reportingTaskRates.organizationId,
              schema.reportingTaskRates.jobId,
              schema.reportingTaskRates.step,
            ],
            set: parsed.data,
          })
          .returning();
        await tx.insert(schema.reportingAudit).values({
          organizationId,
          actorId: c.var.auth.user.localUserId,
          resourceId: saved.id,
          action: "task_rate_saved",
          after: saved,
        });
        return saved;
      });
      if (!taskRate)
        return badRequestResponse(
          c,
          "task_rate_locked",
          "Task rates can only change before work starts.",
        );
      return c.json({ taskRate });
    })
    .post("/time-entries", async (c) => {
      if (!hasCapability(c.var.auth.membership.role, "jobs:write"))
        return forbiddenResponse(c, "time_entry_forbidden");
      const parsed = timeSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_time_entry");
      const organizationId = c.var.auth.organization.localOrganizationId;
      const [job] = await db
        .select()
        .from(schema.jobs)
        .where(
          and(eq(schema.jobs.id, parsed.data.jobId), await buildAccessibleJobsWhere(c.var.auth)),
        );
      if (!job?.projectId) return notFoundResponse(c, "job_not_found");
      const [external] = await db
        .select()
        .from(schema.externalJobDetails)
        .where(eq(schema.externalJobDetails.jobId, job.id));
      if (external) return badRequestResponse(c, "native_job_required");
      const [project] = await db
        .select()
        .from(schema.projects)
        .where(eq(schema.projects.id, job.projectId));
      const payload = job.inputPayload as {
        sourceLocale?: string;
        targetLocales?: string[];
        targetLocale?: string;
      };
      const rate = await resolveReportingRate({
        organizationId,
        projectId: job.projectId,
        jobId: job.id,
        sourceLocale: payload.sourceLocale ?? project.sourceLocale ?? "en",
        targetLocale: parsed.data.targetLocale,
        step: parsed.data.step,
      });
      const timeEntry = await db.transaction(async (tx) => {
        await tx
          .select({ id: schema.jobs.id })
          .from(schema.jobs)
          .where(eq(schema.jobs.id, job.id))
          .for("update");
        const [taskRate] = await tx
          .select()
          .from(schema.reportingTaskRates)
          .where(
            and(
              eq(schema.reportingTaskRates.jobId, job.id),
              eq(schema.reportingTaskRates.step, parsed.data.step),
            ),
          );
        const [entry] = await tx
          .insert(schema.reportingTimeEntries)
          .values({
            ...parsed.data,
            workDate: new Date(parsed.data.workDate + "T00:00:00Z"),
            organizationId,
            projectId: job.projectId!,
            contributorId: c.var.auth.user.localUserId,
            rateId: rate?.id,
          })
          .returning();
        if (rate?.basis !== "word" && taskRate?.overrideUsd == null)
          await tx.insert(schema.reportingCosts).values({
            organizationId,
            projectId: job.projectId,
            jobId: job.id,
            operationKey: `time:${entry.id}`,
            kind: "human",
            step: entry.step,
            timeEntryId: entry.id,
            targetLocale: entry.targetLocale,
            rateId: rate?.id,
            amountUsd: rate ? hourlyCost(rate.rate, entry.minutes) : null,
            basis: rate ? "rate" : "unpriced",
            createdAt: entry.workDate,
          });
        await tx.insert(schema.reportingAudit).values({
          organizationId,
          actorId: c.var.auth.user.localUserId,
          resourceId: entry.id,
          action: "time_created",
          after: entry,
        });
        return entry;
      });
      await reportingStart();
      return c.json({ timeEntry: { id: timeEntry.id } }, 201);
    })
    .patch("/time-entries/:id", async (c) => {
      const parsed = timeUpdateSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_time_entry");
      const organizationId = c.var.auth.organization.localOrganizationId;
      const [entry] = await db
        .select()
        .from(schema.reportingTimeEntries)
        .where(
          and(
            eq(schema.reportingTimeEntries.organizationId, organizationId),
            eq(schema.reportingTimeEntries.id, c.req.param("id")),
            eq(schema.reportingTimeEntries.voided, false),
          ),
        );
      if (
        !entry?.projectId ||
        !(await getAccessibleProjectIds(c.var.auth)).includes(entry.projectId)
      )
        return notFoundResponse(c, "time_entry_not_found");
      if (
        !hasCapability(c.var.auth.membership.role, "jobs:write") ||
        (entry.contributorId !== c.var.auth.user.localUserId &&
          !hasCapability(c.var.auth.membership.role, "billing:write"))
      )
        return forbiddenResponse(c, "time_entry_forbidden");
      await db.transaction(async (tx) => {
        const [rate] = entry.rateId
          ? await tx
              .select()
              .from(schema.reportingRates)
              .where(eq(schema.reportingRates.id, entry.rateId))
          : [];
        const workDate = new Date(parsed.data.workDate + "T00:00:00Z");
        await tx
          .update(schema.reportingTimeEntries)
          .set({ ...parsed.data, workDate })
          .where(eq(schema.reportingTimeEntries.id, entry.id));
        if (rate?.basis !== "word")
          await tx
            .update(schema.reportingCosts)
            .set({
              amountUsd: rate ? hourlyCost(rate.rate, parsed.data.minutes) : null,
              createdAt: workDate,
            })
            .where(eq(schema.reportingCosts.timeEntryId, entry.id));
        await tx.insert(schema.reportingAudit).values({
          organizationId,
          actorId: c.var.auth.user.localUserId,
          resourceId: entry.id,
          action: "time_updated",
          before: entry,
          after: parsed.data,
        });
      });
      return c.json({ timeEntry: { id: entry.id } });
    })
    .delete("/time-entries/:id", async (c) => {
      const organizationId = c.var.auth.organization.localOrganizationId;
      const [entry] = await db
        .select()
        .from(schema.reportingTimeEntries)
        .where(
          and(
            eq(schema.reportingTimeEntries.organizationId, organizationId),
            eq(schema.reportingTimeEntries.id, c.req.param("id")),
          ),
        );
      if (
        !entry?.projectId ||
        !(await getAccessibleProjectIds(c.var.auth)).includes(entry.projectId)
      )
        return notFoundResponse(c, "time_entry_not_found");
      if (
        entry.contributorId !== c.var.auth.user.localUserId &&
        !hasCapability(c.var.auth.membership.role, "billing:write")
      )
        return forbiddenResponse(c, "time_entry_forbidden");
      await db.transaction(async (tx) => {
        await tx
          .update(schema.reportingTimeEntries)
          .set({ voided: true })
          .where(eq(schema.reportingTimeEntries.id, entry.id));
        await tx
          .update(schema.reportingCosts)
          .set({ voided: true })
          .where(eq(schema.reportingCosts.timeEntryId, entry.id));
        await tx.insert(schema.reportingAudit).values({
          organizationId,
          actorId: c.var.auth.user.localUserId,
          resourceId: entry.id,
          action: "time_voided",
          before: entry,
        });
      });
      return c.body(null, 204);
    })
    .post("/expenses", async (c) => {
      if (!hasCapability(c.var.auth.membership.role, "billing:write"))
        return forbiddenResponse(c, "report_costs_forbidden");
      const parsed = expenseSchema.safeParse(await c.req.json());
      if (!parsed.success) return badRequestResponse(c, "invalid_expense");
      const organizationId = c.var.auth.organization.localOrganizationId;
      if (!(await getAccessibleProjectIds(c.var.auth)).includes(parsed.data.projectId))
        return notFoundResponse(c, "project_not_found");
      if (parsed.data.jobId) {
        const [job] = await db
          .select()
          .from(schema.jobs)
          .where(
            and(
              eq(schema.jobs.id, parsed.data.jobId),
              eq(schema.jobs.projectId, parsed.data.projectId),
              eq(schema.jobs.organizationId, organizationId),
            ),
          );
        if (!job) return notFoundResponse(c, "job_not_found");
      }
      const [expense] = await db
        .insert(schema.reportingCosts)
        .values({ ...parsed.data, organizationId, kind: "expense", basis: "manual" })
        .onConflictDoNothing()
        .returning();
      return c.json({ expense: expense ?? null }, 201);
    });
}

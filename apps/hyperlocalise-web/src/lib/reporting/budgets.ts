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
import { and, eq, inArray, sql } from "drizzle-orm";
import { db, schema } from "@/lib/database/client";
import { budgetWarning, usdUnits, usdString, wordCost, hourlyCost } from "./money";

export async function budgetSummary(organizationId: string, projectIds: string[]) {
  if (!projectIds.length) return [];
  const budgets = await db
    .select()
    .from(schema.reportingBudgets)
    .where(
      and(
        eq(schema.reportingBudgets.organizationId, organizationId),
        inArray(schema.reportingBudgets.projectId, projectIds),
      ),
    );
  const summaries = [];
  for (const budget of budgets) {
    const costs = await db.execute<{ amount: string; unpriced: number }>(sql`
            select coalesce(sum(c.amount_usd),0)::text as amount, count(*) filter(where c.amount_usd is null)::integer as unpriced
            from reporting_costs c where c.organization_id=${organizationId} and c.project_id=${budget.projectId} and not c.voided

        `);
    const pending = await db.execute<{
      words: number | null;
      bucket: typeof schema.reportingAnalyses.$inferSelect.bucket;
      rate: string | null;
      basis: string | null;
      percentages: Record<string, number> | null;
    }>(sql`
            select a.words,a.bucket,r.rate,r.basis,r.percentages from reporting_analyses a left join reporting_rates r on r.id=a.rate_id join jobs j on j.id=a.job_id
            where a.organization_id=${organizationId} and a.project_id=${budget.projectId} and a.is_current and a.billable and j.status not in ('failed','cancelled')
            and not exists(select 1 from reporting_activity e where e.analysis_id=a.id and e.kind='completion' and e.step=a.step)
            and not exists(select 1 from reporting_task_rates t where t.organization_id=a.organization_id and t.job_id=a.job_id and t.step=a.step and t.override_usd is not null)
        `);
    let outstanding = BigInt(0),
      incomplete = costs.rows[0].unpriced > 0;
    for (const row of pending.rows) {
      if (row.basis === "hour") continue;
      if (row.words === null || !row.rate || row.bucket === "unavailable") {
        incomplete = true;
        continue;
      }
      outstanding += usdUnits(wordCost(row.rate, row.words, row.percentages?.[row.bucket] ?? 100));
    }
    const hours = await db.execute<{ estimated: number | null; logged: number; rate: string }>(sql`
            select t.estimated_minutes as estimated, coalesce((select sum(e.minutes) from reporting_time_entries e where e.job_id=a.job_id and e.step=r.step and not e.voided),0)::integer as logged, r.rate
            from (select distinct job_id,rate_id from reporting_analyses where organization_id=${organizationId} and project_id=${budget.projectId} and is_current and billable) a
            join reporting_rates r on r.id=a.rate_id join jobs j on j.id=a.job_id
            left join reporting_task_rates t on t.job_id=a.job_id and t.step=r.step
            where r.basis='hour' and t.override_usd is null and j.status not in ('succeeded','failed','cancelled')
        `);
    for (const row of hours.rows) {
      if (row.estimated === null) {
        incomplete = true;
        continue;
      }
      outstanding += usdUnits(hourlyCost(row.rate, Math.max(0, row.estimated - row.logged)));
    }
    const overrides = await db.execute<{ amount: string }>(sql`
      select coalesce(sum(r.override_usd),0)::text as amount from reporting_task_rates r join jobs j on j.id=r.job_id
      where r.organization_id=${organizationId} and j.project_id=${budget.projectId} and r.override_usd is not null
      and j.status not in ('failed','cancelled')
      and not exists(select 1 from reporting_costs c where c.organization_id=r.organization_id and c.operation_key='task-override:' || r.id)
    `);
    outstanding += usdUnits(overrides.rows[0].amount);
    const accrued = usdUnits(costs.rows[0].amount);
    summaries.push({
      ...budget,
      accruedUsd: usdString(accrued),
      outstandingUsd: usdString(outstanding),
      forecastUsd: usdString(accrued + outstanding),
      remainingUsd: usdString(usdUnits(budget.budget) - accrued),
      incomplete,
      spendWarning: budgetWarning(budget.budget, usdString(accrued)),
      forecastWarning: budgetWarning(budget.budget, usdString(accrued + outstanding)),
    });
  }
  return summaries;
}

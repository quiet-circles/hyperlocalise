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
import { db } from "@/lib/database/client";
import type { z } from "zod";
import type { reportQuerySchema } from "@/api/routes/reports/reports.schema";
import { csvCell } from "./word-analysis";

export type ReportQuery = z.infer<typeof reportQuerySchema>;
export type ReportRow = {
  period: string;
  projectId: string | null;
  jobId: string | null;
  targetLocale: string | null;
  step: string;
  bucket: string | null;
  kind: string;
  words: number;
  minutes: number;
  durationMs: number;
  amountUsd: string | null;
  inputTokens: number;
  outputTokens: number;
  unavailable: number;
  count: number;
};
export async function queryReport(input: {
  organizationId: string;
  projectIds: string[];
  financial: boolean;
  query: ReportQuery;
}) {
  const { query } = input;
  const from = query.from ?? new Date(Date.now() - 27 * 86400000).toISOString().slice(0, 10);
  const to = query.to ?? new Date().toISOString().slice(0, 10);
  const scope = input.projectIds.length
    ? sql`project_id in (${sql.join(
        input.projectIds.map((id) => sql`${id}`),
        sql`, `,
      )})`
    : sql`false`;
  const includeCosts = input.financial;
  const result = await db.execute<ReportRow>(sql`
        with facts as (
            select a.organization_id, a.project_id, a.job_id, a.target_locale, a.step, a.created_at at time zone 'UTC' as occurred_at,
                s.bucket, a.kind, coalesce(s.words, 0) as words, 0 as minutes, coalesce(a.duration_ms,0) as duration_ms,
                null::numeric as amount_usd, 0 as input_tokens, 0 as output_tokens,
                case when a.kind='completion' and (s.words is null or s.bucket='unavailable') then 1 else 0 end as unavailable
            from reporting_activity a left join reporting_analyses s on s.id=a.analysis_id
            union all
            select n.organization_id,w.project_id,null,null,n.node_type,n.finished_at at time zone 'UTC',null,'workflow',0,0,
              (extract(epoch from (n.finished_at-n.started_at))*1000)::integer,null,0,0,0
            from visual_workflow_node_runs n join visual_workflow_runs r on r.id=n.run_id join visual_workflows w on w.id=r.visual_workflow_id
            where n.started_at >= (select started_at from reporting_rollout where id=1) and n.finished_at is not null
            union all
            select organization_id,project_id,job_id,target_locale,step,work_date at time zone 'UTC',null,'time',0,minutes,0,null,0,0,0 from reporting_time_entries where not voided
            union all
            select c.organization_id,c.project_id,c.job_id,c.target_locale,c.step,c.created_at at time zone 'UTC',null,c.kind,0,0,0,
                case when ${includeCosts} then c.amount_usd else null end,c.input_tokens,c.output_tokens,
                case when c.amount_usd is null then 1 else 0 end
            from reporting_costs c where not c.voided and (${includeCosts} or c.kind='ai')

        )
        select to_char(date_trunc(${query.interval}, occurred_at),'YYYY-MM-DD') as period,
            project_id as "projectId",job_id as "jobId",target_locale as "targetLocale",step,bucket,kind,
            sum(words)::integer as words,sum(minutes)::integer as minutes,sum(duration_ms)::float8 as "durationMs",
            sum(amount_usd)::text as "amountUsd",coalesce(sum(input_tokens),0)::integer as "inputTokens",coalesce(sum(output_tokens),0)::integer as "outputTokens",
            sum(unavailable)::integer as unavailable,count(*)::integer as count
        from facts where organization_id=${input.organizationId} and (${scope} or (${input.financial} and project_id is null))
            and occurred_at >= ${from}::date and occurred_at < ${to}::date + interval '1 day'
            ${query.projectId ? sql`and project_id=${query.projectId}` : sql``}
            ${query.jobId ? sql`and job_id=${query.jobId}` : sql``}
            ${query.targetLocale ? sql`and (target_locale=${query.targetLocale} or (target_locale is null and kind in ('ai','human','expense')))` : sql``}
            ${query.step ? sql`and step=${query.step}` : sql``}
            ${query.view === "words" ? sql`and kind='completion'` : query.view === "time" ? sql`and kind in ('time','workflow','status')` : query.view === "costs" ? sql`and kind in ('human','ai','expense')` : sql``}
        group by period,project_id,job_id,target_locale,step,bucket,kind order by period desc,project_id,job_id,target_locale,step,bucket,kind
    `);
  const rollout = await db.execute<{ startedAt: string }>(
    sql`select started_at::text as "startedAt" from reporting_rollout where id=1`,
  );
  const sourceCounts = await db.execute<{ sourceWords: number; workloadWords: number }>(sql`
    select coalesce(sum(words),0)::integer as "sourceWords",coalesce(sum(workload),0)::integer as "workloadWords" from (
      select job_id,segment_id,source_revision,max(words) as words,sum(words) as workload from reporting_analyses
      where organization_id=${input.organizationId} and (${scope})
      and created_at >= ${from}::date and created_at < ${to}::date + interval '1 day'
      ${query.projectId ? sql`and project_id=${query.projectId}` : sql``}
      ${query.jobId ? sql`and job_id=${query.jobId}` : sql``}
      ${query.targetLocale ? sql`and target_locale=${query.targetLocale}` : sql``}
      and step=${query.step ?? "translation"}
      group by job_id,segment_id,source_revision
    ) source_scope
  `);
  return {
    analysis: sourceCounts.rows[0],
    rows: result.rows,
    startedAt: rollout.rows[0]?.startedAt ?? null,
    from,
    to,
    timezone: "UTC" as const,
    currency: "USD" as const,
    financial: input.financial,
  };
}
export function reportCsv(rows: ReportRow[], financial: boolean) {
  const columns: (keyof ReportRow)[] = [
    "period",
    "projectId",
    "jobId",
    "targetLocale",
    "step",
    "bucket",
    "kind",
    "words",
    "minutes",
    "durationMs",
    "inputTokens",
    "outputTokens",
    "unavailable",
    "count",
  ];
  if (financial) columns.push("amountUsd");
  return [
    columns.map(csvCell).join(","),
    ...rows.map((row) => columns.map((key) => csvCell(row[key])).join(",")),
  ].join("\r\n");
}

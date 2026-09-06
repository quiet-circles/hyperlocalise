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

export const OVERVIEW_LOOKBACK_DAYS = 7;
export const OVERVIEW_ACTIVITY_LIMIT = 4;
export const OVERVIEW_PROJECT_LIMIT = 2;
export const OVERVIEW_BOARD_LIMIT = 3;
export const OVERVIEW_AUTOMATION_LIMIT = 3;

export type OverviewMetricSeries = {
  count: number;
  series: number[];
};

export type OverviewJobKind =
  | "translation"
  | "research"
  | "review"
  | "proofread"
  | "sync"
  | "asset_management";

export type OverviewResolvedTitle =
  | { kind: "text"; text: string }
  | { kind: "review"; criteria: string }
  | { kind: "sync"; direction: string | null; connectorKind: string }
  | { kind: "id"; id: string };

export type OverviewActivityItem = {
  id: string;
  kind: "job" | "automation";
  title: OverviewResolvedTitle;
  projectName: string | null;
  jobKind: OverviewJobKind | null;
  jobType: string | null;
  status: string;
  href: string | null;
  updatedAt: string;
  attention: boolean;
};

export type OverviewProjectItem = {
  id: string;
  name: string;
  source: "native" | "external_tms";
  providerKind: string | null;
  domain: string | null;
  localeRoute: string;
  latestJobTitle: OverviewResolvedTitle | null;
  latestJobAt: string | null;
  openCount: number;
  failedCount: number;
  href: string;
};

export type OverviewBoardItem = {
  id: string;
  identifier: string;
  title: string;
  projectName: string;
  locale: string | null;
  priority: string | null;
  updatedAt: string;
  href: string;
};

export type OverviewAutomationItem = {
  id: string;
  automationId: string;
  name: string;
  triggerSource: string;
  status: string;
  updatedAt: string;
  href: string;
};

export type WorkspaceOverviewSnapshot = {
  metrics: {
    jobs: OverviewMetricSeries;
    translations: OverviewMetricSeries;
    automations: { total: number; paused: number } | null;
    issues: { open: number; p1: number };
  };
  activity: OverviewActivityItem[];
  projects: OverviewProjectItem[];
  board: OverviewBoardItem[];
  automations: OverviewAutomationItem[];
};

export type OverviewJobTitleInput = {
  id: string;
  kind: string;
  inputPayload: unknown;
  externalTitle?: string | null;
  reviewCriteria?: string | null;
  syncConnectorKind?: string | null;
  syncDirection?: string | null;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function utcDayKey(date: Date): string {
  return date.toISOString().slice(0, 10);
}

export function dailySeriesDays(now = new Date(), days = OVERVIEW_LOOKBACK_DAYS): Date[] {
  const startUtc = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  const dates: Date[] = [];

  for (let offset = days - 1; offset >= 0; offset -= 1) {
    dates.push(new Date(startUtc - offset * 86_400_000));
  }

  return dates;
}

export function fillDailySeries(
  rows: readonly { day: string; count: number }[],
  now = new Date(),
  days = OVERVIEW_LOOKBACK_DAYS,
): number[] {
  const counts = new Map(rows.map((row) => [row.day, row.count]));
  return dailySeriesDays(now, days).map((day) => counts.get(utcDayKey(day)) ?? 0);
}

export function resolveOverviewJobTitle(job: OverviewJobTitleInput): OverviewResolvedTitle {
  if (job.externalTitle?.trim()) {
    return { kind: "text", text: job.externalTitle.trim() };
  }

  const payload = isRecord(job.inputPayload) ? job.inputPayload : {};
  const metadata = isRecord(payload.metadata) ? payload.metadata : {};
  if (typeof metadata.title === "string" && metadata.title.trim()) {
    return { kind: "text", text: metadata.title.trim() };
  }

  if (job.kind === "review" && job.reviewCriteria?.trim()) {
    return { kind: "review", criteria: job.reviewCriteria.trim() };
  }

  if (job.kind === "sync" && job.syncConnectorKind) {
    return {
      kind: "sync",
      direction: job.syncDirection ?? null,
      connectorKind: job.syncConnectorKind,
    };
  }

  if (typeof payload.sourceText === "string" && payload.sourceText.trim()) {
    return { kind: "text", text: payload.sourceText.trim().slice(0, 80) };
  }

  if (typeof payload.sourceFileId === "string" && payload.sourceFileId.trim()) {
    return { kind: "text", text: payload.sourceFileId.trim() };
  }

  return { kind: "id", id: job.id };
}

export function overviewJobKindValue(job: { kind: string; type?: string | null }): {
  jobKind: OverviewJobKind | null;
  jobType: string | null;
} {
  const jobKind = isOverviewJobKind(job.kind) ? job.kind : null;
  if (jobKind === "translation" && job.type) {
    return { jobKind, jobType: job.type };
  }

  return { jobKind, jobType: null };
}

export function isOverviewJobKind(value: string): value is OverviewJobKind {
  return (
    value === "translation" ||
    value === "research" ||
    value === "review" ||
    value === "proofread" ||
    value === "sync" ||
    value === "asset_management"
  );
}

export function formatOverviewLocaleRoute(
  sourceLocale: string | null | undefined,
  targetLocales: readonly string[] | null | undefined,
): string {
  const source = sourceLocale?.trim() || "—";
  const targets = targetLocales?.filter((locale) => locale.trim().length > 0) ?? [];
  if (targets.length === 0) {
    return source;
  }

  const preview = targets.slice(0, 2).join(", ");
  const suffix = targets.length > 2 ? ` +${targets.length - 2}` : "";
  return `${source} → ${preview}${suffix}`;
}

export function rankOverviewActivity(
  items: readonly OverviewActivityItem[],
): OverviewActivityItem[] {
  return [...items]
    .toSorted((left, right) => {
      const leftFailed = left.status === "failed" ? 0 : 1;
      const rightFailed = right.status === "failed" ? 0 : 1;
      if (leftFailed !== rightFailed) {
        return leftFailed - rightFailed;
      }

      return new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime();
    })
    .slice(0, OVERVIEW_ACTIVITY_LIMIT);
}

export function shouldIncludeOverviewAutomations(input: {
  isWorkspaceOperator: boolean;
  automationsEnabled: boolean;
}): boolean {
  return input.isWorkspaceOperator && input.automationsEnabled;
}

export function countOverviewAutomationStatuses(
  rows: readonly { status: string; count: number }[],
): { total: number; paused: number } {
  let total = 0;
  let paused = 0;

  for (const row of rows) {
    if (row.status !== "active" && row.status !== "paused") {
      continue;
    }

    total += row.count;
    if (row.status === "paused") {
      paused += row.count;
    }
  }

  return { total, paused };
}

export function mergeOverviewProjectSources<T extends { id: string }>(input: {
  live: readonly T[];
  materialized: readonly T[];
  native: readonly T[];
}): T[] {
  const seen = new Set<string>();
  const merged: T[] = [];

  for (const project of [...input.live, ...input.materialized, ...input.native]) {
    if (seen.has(project.id)) {
      continue;
    }

    seen.add(project.id);
    merged.push(project);
  }

  return merged;
}

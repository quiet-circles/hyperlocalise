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
// @vitest-environment happy-dom

import type { ReactNode } from "react";
import { render, screen, within } from "@testing-library/react";
import { IntlProvider } from "react-intl";
import { describe, expect, it, vi } from "vite-plus/test";

import { TooltipProvider } from "@/components/ui/tooltip";
import { dailySeriesDays } from "@/lib/workspace/overview-snapshot-model";

import { dashboardOverviewFixture } from "./dashboard.fixture";
import { DashboardPageView } from "./dashboard-page-view";

vi.mock("next/image", () => ({
  default: ({ alt }: { alt?: string }) => <img alt={alt ?? ""} />,
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

function renderDashboard() {
  return render(
    <IntlProvider locale="en" messages={{}}>
      <TooltipProvider>
        <DashboardPageView
          organizationSlug="acme"
          overview={dashboardOverviewFixture}
          automationsEnabled
          onNewRequest={() => undefined}
        />
      </TooltipProvider>
    </IntlProvider>,
  );
}

function formatBarLabel(day: Date, count: number) {
  const date = new Intl.DateTimeFormat("en", {
    day: "numeric",
    month: "short",
    timeZone: "UTC",
  }).format(day);
  return `${date}: ${count}`;
}

describe("DashboardPageView sparkline", () => {
  it("labels each bar with its date and count", () => {
    renderDashboard();

    const jobsChart = screen.getByRole("group", { name: /Jobs by day:/ });
    expect(jobsChart).toHaveAccessibleName("Jobs by day: 4, 6, 5, 8, 7, 9, 9");

    const jobsCounts = dashboardOverviewFixture.metrics.jobs.series;
    const bars = within(jobsChart).getAllByRole("button");
    expect(bars.map((bar) => bar.getAttribute("aria-label"))).toEqual(
      dailySeriesDays().map((day, index) => formatBarLabel(day, jobsCounts[index] ?? 0)),
    );
    expect(bars[0]?.getAttribute("data-slot")).toBe("tooltip-trigger");
  });
});

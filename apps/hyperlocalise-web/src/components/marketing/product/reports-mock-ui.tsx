"use client";

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
import { useMemo, useState, type ReactNode } from "react";
import { FormattedMessage, useIntl } from "react-intl";

import { SAGE_MESH_GRADIENT_SRC } from "@/components/marketing/hero-frame-mesh-stage";
import { REQUEST_DEMO_URL } from "@/components/marketing/request-demo";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/primitives/cn";

import { reportsMockMessages } from "./reports-mock-ui.messages";
import {
  MarketingMockShell,
  type MarketingMockMeshPosition,
  type MarketingMockVariant,
} from "./marketing-mock-shell";
import { MarketingMockUseCaseSelector } from "./marketing-mock-use-case-selector";

type ReportFocus = "words" | "costs" | "time";

function ReportsDashboard({ focus }: { focus: ReportFocus }) {
  const intl = useIntl();

  const metrics = useMemo(() => {
    if (focus === "costs") {
      return [
        { label: intl.formatMessage(reportsMockMessages.accruedSpend), value: "$4,820" },
        { label: intl.formatMessage(reportsMockMessages.budgetLeft), value: "$1,180" },
      ];
    }

    if (focus === "time") {
      return [
        { label: intl.formatMessage(reportsMockMessages.hoursLogged), value: "86.5" },
        { label: intl.formatMessage(reportsMockMessages.reviewHours), value: "22.0" },
      ];
    }

    return [
      { label: intl.formatMessage(reportsMockMessages.sourceWords), value: "12,480" },
      { label: intl.formatMessage(reportsMockMessages.workloadWords), value: "18,210" },
    ];
  }, [focus, intl]);

  const rows: { locale: string; step: string; value: string; highlight?: boolean }[] =
    useMemo(() => {
      const locales = [
        intl.formatMessage(reportsMockMessages.localeJa),
        intl.formatMessage(reportsMockMessages.localeDe),
        intl.formatMessage(reportsMockMessages.localeFr),
      ];
      const translation = intl.formatMessage(reportsMockMessages.translationStep);
      const review = intl.formatMessage(reportsMockMessages.reviewStep);

      if (focus === "costs") {
        return [
          { locale: locales[0], step: translation, value: "$2,140" },
          { locale: locales[1], step: translation, value: "$1,560" },
          { locale: locales[2], step: review, value: "$1,120", highlight: true },
        ];
      }

      if (focus === "time") {
        return [
          { locale: locales[0], step: translation, value: "34.0 h" },
          { locale: locales[1], step: translation, value: "28.5 h" },
          { locale: locales[2], step: review, value: "24.0 h" },
        ];
      }

      return [
        { locale: locales[0], step: translation, value: "6,420" },
        { locale: locales[1], step: translation, value: "5,180" },
        { locale: locales[2], step: review, value: "6,610" },
      ];
    }, [focus, intl]);

  return (
    <div className="flex w-full flex-col overflow-hidden rounded-xl border border-border/80 bg-background/90 shadow-lg backdrop-blur-sm">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border/50 px-4 py-3">
        <div>
          <div className="text-sm font-semibold text-foreground">
            <FormattedMessage {...reportsMockMessages.panelTitle} />
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            <FormattedMessage {...reportsMockMessages.panelRange} />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 border-b border-border/50 px-4 py-4">
        {metrics.map((metric) => (
          <div key={metric.label} className="rounded-lg border border-border/60 px-3 py-2.5">
            <div className="text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
              {metric.label}
            </div>
            <div className="mt-1 font-heading text-2xl font-medium tabular-nums text-foreground">
              {metric.value}
            </div>
          </div>
        ))}
      </div>

      <ul className="divide-y divide-border/50">
        {rows.map((row) => (
          <li
            key={`${row.locale}-${row.step}`}
            className={cn(
              "flex items-center justify-between gap-3 px-4 py-2.5 text-xs",
              row.highlight ? "bg-primary/5" : undefined,
            )}
          >
            <span className="font-medium text-foreground">{row.locale}</span>
            <span className="text-muted-foreground">{row.step}</span>
            <span className="tabular-nums text-foreground">{row.value}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function ReportsMockUI({
  priority = false,
  pauseAutoplay: _pauseAutoplay = false,
  renderCta,
  variant = "full",
  aside,
  meshPosition = "left",
}: {
  priority?: boolean;
  pauseAutoplay?: boolean;
  renderCta?: () => ReactNode;
  variant?: MarketingMockVariant;
  aside?: ReactNode;
  meshPosition?: MarketingMockMeshPosition;
}) {
  const intl = useIntl();

  const useCases = useMemo(
    () => [
      {
        id: "words",
        title: intl.formatMessage(reportsMockMessages.useCaseWordsTitle),
        description: intl.formatMessage(reportsMockMessages.useCaseWordsDescription),
      },
      {
        id: "costs",
        title: intl.formatMessage(reportsMockMessages.useCaseCostsTitle),
        description: intl.formatMessage(reportsMockMessages.useCaseCostsDescription),
      },
      {
        id: "time",
        title: intl.formatMessage(reportsMockMessages.useCaseTimeTitle),
        description: intl.formatMessage(reportsMockMessages.useCaseTimeDescription),
      },
    ],
    [intl],
  );

  const [activeId, setActiveId] = useState(useCases[0]?.id ?? "words");
  const focus = (activeId as ReportFocus) || "words";

  const sidebar =
    variant === "full" ? (
      <MarketingMockUseCaseSelector
        eyebrow={reportsMockMessages.eyebrow}
        headline={reportsMockMessages.headline}
        useCases={useCases}
        activeId={activeId}
        onSelect={setActiveId}
        cta={
          renderCta === undefined ? (
            <Button
              variant="outline"
              size="lg"
              nativeButton={false}
              render={<a href={REQUEST_DEMO_URL} target="_blank" rel="noopener noreferrer" />}
              className="cursor-pointer rounded-sm"
            >
              <FormattedMessage {...reportsMockMessages.requestDemo} />
            </Button>
          ) : (
            renderCta()
          )
        }
      />
    ) : undefined;

  return (
    <MarketingMockShell
      visual={<ReportsDashboard focus={focus} />}
      sidebar={sidebar}
      aside={aside}
      meshSrc={SAGE_MESH_GRADIENT_SRC}
      priority={priority}
      variant={variant}
      meshPosition={meshPosition}
    />
  );
}

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
import type { ReactNode } from "react";
import { useIntl } from "react-intl";

import {
  PageHeader,
  WorkspacePageShell,
} from "@/app/[lang]/(authenticated)/org/[organizationSlug]/_components/workspace-resource-shared";
import { AutomationsMockUI } from "@/components/marketing/product/automations-mock-ui";
import { DomainsMockUI } from "@/components/marketing/product/domains-mock-ui";
import { GuidelineMockUI } from "@/components/marketing/product/guideline-mock-ui";
import { HyperlabMockUI } from "@/components/marketing/product/hyperlab-mock-ui";
import { ReportsMockUI } from "@/components/marketing/product/reports-mock-ui";
import type { MarketingMockMeshPosition } from "@/components/marketing/product/marketing-mock-shell";

import { FeatureTeaserCtaPanel } from "./feature-teaser-cta-panel";
import {
  featureTeaserContactSubjects,
  featureTeaserMessages,
  featureTeaserRegistry,
  type FeatureTeaserId,
  type FeatureTeaserScope,
} from "./feature-teaser-registry";

const teaserMockProps = {
  variant: "embedded" as const,
  pauseAutoplay: false,
  renderCta: () => null,
  meshPosition: "left" as const satisfies MarketingMockMeshPosition,
};

function FeatureTeaserShowcase({ feature, aside }: { feature: FeatureTeaserId; aside: ReactNode }) {
  switch (feature) {
    case "automations":
      return <AutomationsMockUI {...teaserMockProps} aside={aside} />;
    case "guideline":
      return <GuidelineMockUI {...teaserMockProps} aside={aside} />;
    case "domains":
      return <DomainsMockUI {...teaserMockProps} aside={aside} />;
    case "hyperlab":
      return <HyperlabMockUI {...teaserMockProps} aside={aside} />;
    case "reports":
      return <ReportsMockUI {...teaserMockProps} aside={aside} />;
  }
}

export function FeatureTeaserPage({
  feature,
  scope = "workspace",
}: {
  feature: FeatureTeaserId;
  scope?: FeatureTeaserScope;
}) {
  const intl = useIntl();
  const config = featureTeaserRegistry[feature];
  const isProjectScope = scope === "project";

  const ctaPanel = (
    <FeatureTeaserCtaPanel
      title={config.earlyAccessTitle}
      description={config.earlyAccessDescription}
      benefits={config.benefits}
      contactSubject={featureTeaserContactSubjects[feature]}
    />
  );

  return (
    <WorkspacePageShell>
      <PageHeader
        icon={config.icon}
        label={intl.formatMessage(isProjectScope ? config.pageLabelProject : config.pageLabel)}
        title={intl.formatMessage(config.pageTitle)}
        description={intl.formatMessage(
          isProjectScope ? config.pageDescriptionProject : config.pageDescription,
        )}
        statusLabel={intl.formatMessage(featureTeaserMessages.previewBadge)}
      />

      <section aria-label={intl.formatMessage(config.pageTitle)}>
        <FeatureTeaserShowcase feature={feature} aside={ctaPanel} />
      </section>
    </WorkspacePageShell>
  );
}

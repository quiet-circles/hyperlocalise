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
import { defineMessages, type MessageDescriptor } from "react-intl";
import {
  Bookmark01Icon,
  ChartHistogramIcon,
  FlashIcon,
  FlaskConicalIcon,
  Globe02Icon,
} from "@hugeicons/core-free-icons";

import type { NavigationIcon } from "@/components/app-shell/navigation-config";

export type FeatureTeaserId = "automations" | "guideline" | "domains" | "hyperlab" | "reports";

export type FeatureTeaserScope = "workspace" | "project";

export type FeatureTeaserConfig = {
  icon: NavigationIcon;
  pageLabel: MessageDescriptor;
  pageLabelProject: MessageDescriptor;
  pageTitle: MessageDescriptor;
  pageDescription: MessageDescriptor;
  pageDescriptionProject: MessageDescriptor;
  earlyAccessTitle: MessageDescriptor;
  earlyAccessDescription: MessageDescriptor;
  benefits: readonly MessageDescriptor[];
};

export const featureTeaserMessages = defineMessages({
  previewBadge: {
    defaultMessage: "Preview",
    id: "slYILV02yb",
    description: "Badge shown on feature teaser pages and gated navigation items",
  },
  requestDemo: {
    defaultMessage: "Request a demo",
    id: "oKauJvNSsW",
    description: "Primary call to action on feature teaser pages",
  },
  contactSupport: {
    defaultMessage: "Contact us",
    id: "RcUVixjzZV",
    description: "Secondary call to action on feature teaser pages",
  },
  contactSubjectAutomations: {
    defaultMessage: "Demo request: Automations",
    id: "++6KcoXi6F",
    description: "Email subject for automations feature teaser contact link",
  },
  contactSubjectGuideline: {
    defaultMessage: "Demo request: Guideline",
    id: "BHmWNCONbs",
    description: "Email subject for guideline feature teaser contact link",
  },
  contactSubjectDomains: {
    defaultMessage: "Demo request: Domains",
    id: "XztnKIGYn5",
    description: "Email subject for domains feature teaser contact link",
  },
  contactSubjectHyperlab: {
    defaultMessage: "Demo request: Hyperlab",
    id: "omHOw+4o9P",
    description: "Email subject for Hyperlab feature teaser contact link",
  },
  contactSubjectReports: {
    defaultMessage: "Demo request: Reports",
    id: "QRlf1XEwRc",
    description: "Email subject for Reports feature teaser contact link",
  },

  automationsPageLabel: {
    defaultMessage: "Workspace",
    id: "SQywnBh/uh",
    description: "Feature teaser page label for workspace automations",
  },
  automationsPageLabelProject: {
    defaultMessage: "Project",
    id: "ioiYh1dY+T",
    description: "Feature teaser page label for project automations",
  },
  automationsTitle: {
    defaultMessage: "Automations",
    id: "uQuuO1WuxO",
    description: "Feature teaser page title for automations",
  },
  automationsDescription: {
    defaultMessage:
      "Put repetitive global content work on autopilot so your team ships to more markets, faster.",
    id: "D3q/aG4RpR",
    description: "Feature teaser page description for workspace automations",
  },
  automationsDescriptionProject: {
    defaultMessage: "Keep this project releasing on time without chasing manual handoffs.",
    id: "99UKjV7Fo4",
    description: "Feature teaser page description for project automations",
  },
  automationsEarlyAccessTitle: {
    defaultMessage: "Ship to more markets without growing the team",
    id: "PG2jivJB3X",
    description: "Feature teaser early access title for automations",
  },
  automationsEarlyAccessDescription: {
    defaultMessage:
      "Automations handles repetitive steps between content, review, and release so your team can focus on quality. Available in early access. Book a demo to see it on your stack.",
    id: "j0WZLPTpqA",
    description: "Feature teaser early access description for automations",
  },
  automationsBenefit0: {
    defaultMessage: "Cut hours lost to manual review and delivery handoffs",
    id: "+je2CrABfF",
    description: "Feature teaser benefit for automations",
  },
  automationsBenefit1: {
    defaultMessage: "Release on schedule, even as locale count grows",
    id: "8JgsbTFb9B",
    description: "Feature teaser benefit for automations",
  },
  automationsBenefit2: {
    defaultMessage: "Scale output without scaling headcount",
    id: "7DcMnrFZgv",
    description: "Feature teaser benefit for automations",
  },

  guidelinePageLabel: {
    defaultMessage: "Workspace",
    id: "Ez0AubJEsT",
    description: "Feature teaser page label for workspace guideline",
  },
  guidelinePageLabelProject: {
    defaultMessage: "Project",
    id: "s0SZ6JCFMT",
    description: "Feature teaser page label for project guideline",
  },
  guidelineTitle: {
    defaultMessage: "Guideline",
    id: "UQT1fdMFbW",
    description: "Feature teaser page title for guideline",
  },
  guidelineDescription: {
    defaultMessage:
      "Capture style, market, and compliance guidance so teams scale into new markets with confidence.",
    id: "4y4y0oNFFl",
    description: "Feature teaser page description for workspace guideline",
  },
  guidelineDescriptionProject: {
    defaultMessage: "Give this project the GTM context it needs to launch and grow in new markets.",
    id: "FcomHiDTaX",
    description: "Feature teaser page description for project guideline",
  },
  guidelineEarlyAccessTitle: {
    defaultMessage: "One playbook for global growth in every market",
    id: "ESsYtK4MNy",
    description: "Feature teaser early access title for guideline",
  },
  guidelineEarlyAccessDescription: {
    defaultMessage:
      "Guideline stores style, market, and compliance rules in one place for teams and AI. Available in early access. Book a demo to see it in action.",
    id: "nY6L1mpX2I",
    description: "Feature teaser early access description for guideline",
  },
  guidelineBenefit0: {
    defaultMessage: "Launch campaigns that fit each market from day one",
    id: "yHuJniMEb/",
    description: "Feature teaser benefit for guideline",
  },
  guidelineBenefit1: {
    defaultMessage: "New regions ramp up faster with shared GTM playbooks",
    id: "66l3idHXx9",
    description: "Feature teaser benefit for guideline",
  },
  guidelineBenefit2: {
    defaultMessage: "Decisions compound as teams learn what works in each market",
    id: "plDoLEcfb7",
    description: "Feature teaser benefit for guideline",
  },

  domainsPageLabel: {
    defaultMessage: "Workspace",
    id: "e39hEFri57",
    description: "Feature teaser page label for domains",
  },
  domainsPageLabelProject: {
    defaultMessage: "Workspace",
    id: "0IVc9Jxawk",
    description: "Feature teaser page label for domains (project scope unused)",
  },
  domainsTitle: {
    defaultMessage: "Domains",
    id: "hSuFqgtras",
    description: "Feature teaser page title for domains",
  },
  domainsDescription: {
    defaultMessage:
      "Audit your websites for localisation, SEO, and AEO. See what is blocking discoverability in every market.",
    id: "2icICV/DAq",
    description: "Feature teaser page description for domains",
  },
  domainsDescriptionProject: {
    defaultMessage:
      "Audit your websites for localisation, SEO, and AEO. See what is blocking discoverability in every market.",
    id: "PyZoyjoXDF",
    description: "Feature teaser page description for domains (project scope unused)",
  },
  domainsEarlyAccessTitle: {
    defaultMessage: "See what is hurting search and AI answers in every locale",
    id: "SWDbxCkEGV",
    description: "Feature teaser early access title for domains",
  },
  domainsEarlyAccessDescription: {
    defaultMessage:
      "Domains crawls your sites and scores localisation, SEO, and AEO readiness. Fix the highest-impact issues first. Available in early access. Book a demo to run an audit.",
    id: "AZAD/rhnHJ",
    description: "Feature teaser early access description for domains",
  },
  domainsBenefit0: {
    defaultMessage: "Find hreflang errors, missing locales, and content gaps",
    id: "iMzelu42WD",
    description: "Feature teaser benefit for domains",
  },
  domainsBenefit1: {
    defaultMessage: "Improve SEO and AEO discoverability across markets",
    id: "J7hBoTRR5O",
    description: "Feature teaser benefit for domains",
  },
  domainsBenefit2: {
    defaultMessage: "Track audit scores and open issues as you expand",
    id: "4Y5ejA62oY",
    description: "Feature teaser benefit for domains",
  },

  hyperlabPageLabel: {
    defaultMessage: "Workspace",
    id: "y4aVDPth8a",
    description: "Feature teaser page label for Hyperlab",
  },
  hyperlabPageLabelProject: {
    defaultMessage: "Workspace",
    id: "mVLREix+gB",
    description: "Feature teaser page label for Hyperlab when opened from a project",
  },
  hyperlabTitle: {
    defaultMessage: "Hyperlab",
    id: "+i+mcUZPR+",
    description: "Feature teaser title for Hyperlab",
  },
  hyperlabDescription: {
    defaultMessage:
      "Ship flags and experiments from this workspace, then evaluate them in your apps.",
    id: "smm3qbe1rP",
    description: "Feature teaser description for Hyperlab",
  },
  hyperlabDescriptionProject: {
    defaultMessage:
      "Ship flags and experiments from this workspace, then evaluate them in your apps.",
    id: "uScOO0+uJ9",
    description: "Feature teaser description for Hyperlab when opened from a project",
  },
  hyperlabEarlyAccessTitle: {
    defaultMessage: "Target, split, and ship without another vendor",
    id: "YWi5RNeyc2",
    description: "Feature teaser early access title for Hyperlab",
  },
  hyperlabEarlyAccessDescription: {
    defaultMessage:
      "Hyperlab lets your team define flags, audiences, and rollouts, then evaluate them over OFREP. Available in early access.",
    id: "AHcwXmS1XC",
    description: "Feature teaser early access description for Hyperlab",
  },
  hyperlabBenefit0: {
    defaultMessage: "Create experiment and config flags unique to your workspace",
    id: "OpL4o1Rl3O",
    description: "Feature teaser benefit for Hyperlab",
  },
  hyperlabBenefit1: {
    defaultMessage: "Target visitors with attribute rules evaluated live",
    id: "32sQGI6Lre",
    description: "Feature teaser benefit for Hyperlab",
  },
  hyperlabBenefit2: {
    defaultMessage: "Evaluate from any OpenFeature SDK over OFREP",
    id: "cGCurVuGU/",
    description: "Feature teaser benefit for Hyperlab",
  },

  reportsPageLabel: {
    defaultMessage: "Workspace",
    id: "tznfLKixWB",
    description: "Feature teaser page label for workspace reports",
  },
  reportsPageLabelProject: {
    defaultMessage: "Project",
    id: "aDWMRaDSsk",
    description: "Feature teaser page label for project reports",
  },
  reportsTitle: {
    defaultMessage: "Reports",
    id: "b/0QCcEDip",
    description: "Feature teaser page title for reports",
  },
  reportsDescription: {
    defaultMessage: "See word counts, time, and cost across projects so you can forecast spend.",
    id: "Y2qBs5EsBK",
    description: "Feature teaser page description for workspace reports",
  },
  reportsDescriptionProject: {
    defaultMessage: "See word counts, time, and cost for this project before work overruns.",
    id: "k1Ku5BR4yb",
    description: "Feature teaser page description for project reports",
  },
  reportsEarlyAccessTitle: {
    defaultMessage: "Know what translation is costing before the invoice arrives",
    id: "3iwjDXakJS",
    description: "Feature teaser early access title for reports",
  },
  reportsEarlyAccessDescription: {
    defaultMessage:
      "Reports rolls word counts, time, and vendor rates into one workspace view. Available in early access. Book a demo to see it on your volume.",
    id: "0/uc5H+XGR",
    description: "Feature teaser early access description for reports",
  },
  reportsBenefit0: {
    defaultMessage: "Track words, time, and cost by project and locale",
    id: "uG/a2iMgAc",
    description: "Feature teaser benefit for reports",
  },
  reportsBenefit1: {
    defaultMessage: "Set rates and budgets before work overruns",
    id: "T+CONpD3t0",
    description: "Feature teaser benefit for reports",
  },
  reportsBenefit2: {
    defaultMessage: "Export the numbers finance and vendors already ask for",
    id: "B3BorgouDo",
    description: "Feature teaser benefit for reports",
  },
});

export const featureTeaserRegistry: Record<FeatureTeaserId, FeatureTeaserConfig> = {
  automations: {
    icon: FlashIcon,
    pageLabel: featureTeaserMessages.automationsPageLabel,
    pageLabelProject: featureTeaserMessages.automationsPageLabelProject,
    pageTitle: featureTeaserMessages.automationsTitle,
    pageDescription: featureTeaserMessages.automationsDescription,
    pageDescriptionProject: featureTeaserMessages.automationsDescriptionProject,
    earlyAccessTitle: featureTeaserMessages.automationsEarlyAccessTitle,
    earlyAccessDescription: featureTeaserMessages.automationsEarlyAccessDescription,
    benefits: [
      featureTeaserMessages.automationsBenefit0,
      featureTeaserMessages.automationsBenefit1,
      featureTeaserMessages.automationsBenefit2,
    ],
  },
  guideline: {
    icon: Bookmark01Icon,
    pageLabel: featureTeaserMessages.guidelinePageLabel,
    pageLabelProject: featureTeaserMessages.guidelinePageLabelProject,
    pageTitle: featureTeaserMessages.guidelineTitle,
    pageDescription: featureTeaserMessages.guidelineDescription,
    pageDescriptionProject: featureTeaserMessages.guidelineDescriptionProject,
    earlyAccessTitle: featureTeaserMessages.guidelineEarlyAccessTitle,
    earlyAccessDescription: featureTeaserMessages.guidelineEarlyAccessDescription,
    benefits: [
      featureTeaserMessages.guidelineBenefit0,
      featureTeaserMessages.guidelineBenefit1,
      featureTeaserMessages.guidelineBenefit2,
    ],
  },
  domains: {
    icon: Globe02Icon,
    pageLabel: featureTeaserMessages.domainsPageLabel,
    pageLabelProject: featureTeaserMessages.domainsPageLabelProject,
    pageTitle: featureTeaserMessages.domainsTitle,
    pageDescription: featureTeaserMessages.domainsDescription,
    pageDescriptionProject: featureTeaserMessages.domainsDescriptionProject,
    earlyAccessTitle: featureTeaserMessages.domainsEarlyAccessTitle,
    earlyAccessDescription: featureTeaserMessages.domainsEarlyAccessDescription,
    benefits: [
      featureTeaserMessages.domainsBenefit0,
      featureTeaserMessages.domainsBenefit1,
      featureTeaserMessages.domainsBenefit2,
    ],
  },
  hyperlab: {
    icon: FlaskConicalIcon,
    pageLabel: featureTeaserMessages.hyperlabPageLabel,
    pageLabelProject: featureTeaserMessages.hyperlabPageLabelProject,
    pageTitle: featureTeaserMessages.hyperlabTitle,
    pageDescription: featureTeaserMessages.hyperlabDescription,
    pageDescriptionProject: featureTeaserMessages.hyperlabDescriptionProject,
    earlyAccessTitle: featureTeaserMessages.hyperlabEarlyAccessTitle,
    earlyAccessDescription: featureTeaserMessages.hyperlabEarlyAccessDescription,
    benefits: [
      featureTeaserMessages.hyperlabBenefit0,
      featureTeaserMessages.hyperlabBenefit1,
      featureTeaserMessages.hyperlabBenefit2,
    ],
  },
  reports: {
    icon: ChartHistogramIcon,
    pageLabel: featureTeaserMessages.reportsPageLabel,
    pageLabelProject: featureTeaserMessages.reportsPageLabelProject,
    pageTitle: featureTeaserMessages.reportsTitle,
    pageDescription: featureTeaserMessages.reportsDescription,
    pageDescriptionProject: featureTeaserMessages.reportsDescriptionProject,
    earlyAccessTitle: featureTeaserMessages.reportsEarlyAccessTitle,
    earlyAccessDescription: featureTeaserMessages.reportsEarlyAccessDescription,
    benefits: [
      featureTeaserMessages.reportsBenefit0,
      featureTeaserMessages.reportsBenefit1,
      featureTeaserMessages.reportsBenefit2,
    ],
  },
};

export const featureTeaserContactSubjects: Record<FeatureTeaserId, MessageDescriptor> = {
  automations: featureTeaserMessages.contactSubjectAutomations,
  guideline: featureTeaserMessages.contactSubjectGuideline,
  domains: featureTeaserMessages.contactSubjectDomains,
  hyperlab: featureTeaserMessages.contactSubjectHyperlab,
  reports: featureTeaserMessages.contactSubjectReports,
};

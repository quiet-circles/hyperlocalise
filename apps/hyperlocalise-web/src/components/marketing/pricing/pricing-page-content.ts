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
import { getIntlShape } from "@/lib/app-i18n/intl";

export type PricingPlanId = "free" | "starter" | "growth" | "enterprise";

export type PricingPlanCta = {
  label: string;
  kind: "coming_soon" | "demo";
};

export type PricingPlan = {
  id: PricingPlanId;
  name: string;
  price: string;
  priceSuffix: string | null;
  description: string;
  popular: boolean;
  includesFrom: string | null;
  features: string[];
  cta: PricingPlanCta;
};

export type PricingMatrixCell =
  | { kind: "check" }
  | { kind: "dash" }
  | { kind: "text"; value: string };

export type PricingMatrixRow = {
  id: string;
  label: string;
  cells: Record<PricingPlanId, PricingMatrixCell>;
};

export type PricingMatrixSection = {
  id: string;
  title: string;
  description: string;
  rows: PricingMatrixRow[];
};

export const pricingPlanOrder: readonly PricingPlanId[] = [
  "free",
  "starter",
  "growth",
  "enterprise",
] as const;

export function getPricingPlans(locale: string): PricingPlan[] {
  const intl = getIntlShape(locale);
  const comingSoon = intl.formatMessage({
    defaultMessage: "Coming soon",
    id: "DwOm/o1t0O",
    description: "Disabled CTA label on Free, Starter, and Growth pricing cards",
  });
  const perMonth = intl.formatMessage({
    defaultMessage: "/mo.",
    id: "clr4hixI6B",
    description: "Monthly price suffix on pricing cards",
  });

  return [
    {
      id: "free",
      name: intl.formatMessage({
        defaultMessage: "Free",
        id: "7QPFtqQQDE",
        description: "Free plan name on the pricing page",
      }),
      price: intl.formatMessage({
        defaultMessage: "$0",
        id: "BBge+5Sj1e",
        description: "Free plan price on the pricing page",
      }),
      priceSuffix: perMonth,
      description: intl.formatMessage({
        defaultMessage: "Evaluate Hyperlocalise with a single-seat workspace.",
        id: "4Xya9zkpLQ",
        description: "Free plan description on the pricing page",
      }),
      popular: false,
      includesFrom: null,
      features: [
        intl.formatMessage({
          defaultMessage: "2 integrations",
          id: "9LcPjakBL/",
          description: "Free plan feature: integration limit",
        }),
        intl.formatMessage({
          defaultMessage: "1 project",
          id: "73DwGFhZM/",
          description: "Free plan feature: project limit",
        }),
        intl.formatMessage({
          defaultMessage: "1 seat",
          id: "V1lR4mPTs5",
          description: "Free plan feature: seat limit",
        }),
      ],
      cta: { label: comingSoon, kind: "coming_soon" },
    },
    {
      id: "starter",
      name: intl.formatMessage({
        defaultMessage: "Starter",
        id: "dBWJx9vBQt",
        description: "Starter plan name on the pricing page",
      }),
      price: intl.formatMessage({
        defaultMessage: "$20",
        id: "cTTX9n9kbk",
        description: "Starter plan price on the pricing page",
      }),
      priceSuffix: perMonth,
      description: intl.formatMessage({
        defaultMessage: "For small teams that need more seats and projects.",
        id: "x9qwrwtKqR",
        description: "Starter plan description on the pricing page",
      }),
      popular: false,
      includesFrom: intl.formatMessage({
        defaultMessage: "All Free features, plus:",
        id: "yoOShm3Zjk",
        description: "Starter plan intro above incremental features",
      }),
      features: [
        intl.formatMessage({
          defaultMessage: "Unlimited projects",
          id: "iRIRASFRNC",
          description: "Starter plan feature: unlimited projects",
        }),
        intl.formatMessage({
          defaultMessage: "5 seats",
          id: "uc3WQiSxaw",
          description: "Starter plan feature: seat limit",
        }),
      ],
      cta: { label: comingSoon, kind: "coming_soon" },
    },
    {
      id: "growth",
      name: intl.formatMessage({
        defaultMessage: "Growth",
        id: "01P9p4MvIA",
        description: "Growth plan name on the pricing page",
      }),
      price: intl.formatMessage({
        defaultMessage: "$2,000",
        id: "y9blutdtEE",
        description: "Growth plan price on the pricing page",
      }),
      priceSuffix: perMonth,
      description: intl.formatMessage({
        defaultMessage: "For teams running localisation in production every week.",
        id: "8yfcbh9AO6",
        description: "Growth plan description on the pricing page",
      }),
      popular: true,
      includesFrom: intl.formatMessage({
        defaultMessage: "All Starter features, plus:",
        id: "4HG5INMxUv",
        description: "Growth plan intro above incremental features",
      }),
      features: [
        intl.formatMessage({
          defaultMessage: "2,000 agent runs per month",
          id: "saGDzW8Kz5",
          description: "Growth plan feature: agent run quota",
        }),
        intl.formatMessage({
          defaultMessage: "2,000 AI tokens per month",
          id: "LzbIVm9zfG",
          description: "Growth plan feature: AI token quota",
        }),
        intl.formatMessage({
          defaultMessage: "20 agent automations",
          id: "D92aikcKtb",
          description: "Growth plan feature: automation limit",
        }),
        intl.formatMessage({
          defaultMessage: "5 integrations",
          id: "j0iC3VoekG",
          description: "Growth plan feature: integration limit",
        }),
        intl.formatMessage({
          defaultMessage: "Unlimited seats",
          id: "7X4JZ1+tNc",
          description: "Growth plan feature: unlimited seats",
        }),
        intl.formatMessage({
          defaultMessage: "Unlimited translation jobs",
          id: "tFGX4qNX4h",
          description: "Growth plan feature: unlimited translation jobs",
        }),
        intl.formatMessage({
          defaultMessage: "AI features",
          id: "YtKo2CQ5hL",
          description: "Growth plan feature: AI feature access",
        }),
      ],
      cta: { label: comingSoon, kind: "coming_soon" },
    },
    {
      id: "enterprise",
      name: intl.formatMessage({
        defaultMessage: "Enterprise",
        id: "Bgy156rCP9",
        description: "Enterprise plan name on the pricing page",
      }),
      price: intl.formatMessage({
        defaultMessage: "Custom",
        id: "fMyeM5BW3s",
        description: "Enterprise plan price label on the pricing page",
      }),
      priceSuffix: null,
      description: intl.formatMessage({
        defaultMessage: "For organizations that need custom limits and support.",
        id: "ZBij4iUX44",
        description: "Enterprise plan description on the pricing page",
      }),
      popular: false,
      includesFrom: intl.formatMessage({
        defaultMessage: "All Growth features, plus:",
        id: "uxTXRcu5Em",
        description: "Enterprise plan intro above incremental features",
      }),
      features: [
        intl.formatMessage({
          defaultMessage: "Custom usage limits",
          id: "e6PVBNU9Sg",
          description: "Enterprise plan feature: custom limits",
        }),
        intl.formatMessage({
          defaultMessage: "SSO / SAML",
          id: "OtoAcs/psJ",
          description: "Enterprise plan feature: SSO",
        }),
        intl.formatMessage({
          defaultMessage: "Service-level agreement",
          id: "AUcojcp0lZ",
          description: "Enterprise plan feature: SLA",
        }),
        intl.formatMessage({
          defaultMessage: "Dedicated support",
          id: "1M4Hu48OSE",
          description: "Enterprise plan feature: dedicated support",
        }),
        intl.formatMessage({
          defaultMessage: "Security review assistance",
          id: "vUcO7nkRNE",
          description: "Enterprise plan feature: security review help",
        }),
      ],
      cta: {
        label: intl.formatMessage({
          defaultMessage: "Contact Sales",
          id: "MRoAAfbdki",
          description: "Enterprise plan CTA label on the pricing page",
        }),
        kind: "demo",
      },
    },
  ];
}

export function getPricingMatrixSections(locale: string): PricingMatrixSection[] {
  const intl = getIntlShape(locale);
  const unlimited = intl.formatMessage({
    defaultMessage: "Unlimited",
    id: "SNl5ipNJ6A",
    description: "Matrix cell value for unlimited plan limits",
  });
  const custom = intl.formatMessage({
    defaultMessage: "Custom",
    id: "yEwFJfSgTY",
    description: "Matrix cell value for custom enterprise limits",
  });

  return [
    {
      id: "workspace",
      title: intl.formatMessage({
        defaultMessage: "Workspace",
        id: "DRiVk5mEAF",
        description: "Pricing matrix section title for workspace limits",
      }),
      description: intl.formatMessage({
        defaultMessage: "Projects, seats, and connected tools for your team.",
        id: "0HGkduChrS",
        description: "Pricing matrix section description for workspace limits",
      }),
      rows: [
        {
          id: "projects",
          label: intl.formatMessage({
            defaultMessage: "Projects",
            id: "Pd3edSzEYJ",
            description: "Pricing matrix row label for projects",
          }),
          cells: {
            free: { kind: "text", value: "1" },
            starter: { kind: "text", value: unlimited },
            growth: { kind: "text", value: unlimited },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "seats",
          label: intl.formatMessage({
            defaultMessage: "Seats",
            id: "X96S8fajCU",
            description: "Pricing matrix row label for seats",
          }),
          cells: {
            free: { kind: "text", value: "1" },
            starter: { kind: "text", value: "5" },
            growth: { kind: "text", value: unlimited },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "integrations",
          label: intl.formatMessage({
            defaultMessage: "Integrations",
            id: "3Zjg+fxxP+",
            description: "Pricing matrix row label for integrations",
          }),
          cells: {
            free: { kind: "text", value: "2" },
            starter: { kind: "text", value: "2" },
            growth: { kind: "text", value: "5" },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "automations",
          label: intl.formatMessage({
            defaultMessage: "Agent Automations",
            id: "kwCQeTqeyE",
            description: "Pricing matrix row label for automations",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "text", value: "20" },
            enterprise: { kind: "text", value: custom },
          },
        },
      ],
    },
    {
      id: "usage",
      title: intl.formatMessage({
        defaultMessage: "Usage",
        id: "Kxj1NjlA8v",
        description: "Pricing matrix section title for usage quotas",
      }),
      description: intl.formatMessage({
        defaultMessage: "Monthly agent, token, and translation capacity.",
        id: "zEqfVrKbs/",
        description: "Pricing matrix section description for usage quotas",
      }),
      rows: [
        {
          id: "agent-runs",
          label: intl.formatMessage({
            defaultMessage: "Agent runs / month",
            id: "u1HpnCoRxX",
            description: "Pricing matrix row label for agent runs",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "text", value: "2,000" },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "ai-tokens",
          label: intl.formatMessage({
            defaultMessage: "AI tokens / month",
            id: "1wUO7p+IGh",
            description: "Pricing matrix row label for AI tokens",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "text", value: "2,000" },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "translation-jobs",
          label: intl.formatMessage({
            defaultMessage: "Translation jobs / month",
            id: "4WS98Fwfj2",
            description: "Pricing matrix row label for translation jobs",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "text", value: unlimited },
            enterprise: { kind: "text", value: custom },
          },
        },
        {
          id: "ai-features",
          label: intl.formatMessage({
            defaultMessage: "AI features",
            id: "bUBd4XGtBI",
            description: "Pricing matrix row label for AI features",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "check" },
            enterprise: { kind: "check" },
          },
        },
      ],
    },
    {
      id: "enterprise",
      title: intl.formatMessage({
        defaultMessage: "Enterprise",
        id: "8tw6rEaK51",
        description: "Pricing matrix section title for enterprise controls",
      }),
      description: intl.formatMessage({
        defaultMessage: "Security, support, and commercial controls for larger orgs.",
        id: "ilusC30/97",
        description: "Pricing matrix section description for enterprise controls",
      }),
      rows: [
        {
          id: "sso",
          label: intl.formatMessage({
            defaultMessage: "SSO / SAML",
            id: "pjTufHJtA+",
            description: "Pricing matrix row label for SSO",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "dash" },
            enterprise: { kind: "check" },
          },
        },
        {
          id: "sla",
          label: intl.formatMessage({
            defaultMessage: "Service-level agreement",
            id: "q+3+ldE+Vn",
            description: "Pricing matrix row label for SLA",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "dash" },
            enterprise: { kind: "check" },
          },
        },
        {
          id: "dedicated-support",
          label: intl.formatMessage({
            defaultMessage: "Dedicated support",
            id: "YJhLdtyhdj",
            description: "Pricing matrix row label for dedicated support",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "dash" },
            enterprise: { kind: "check" },
          },
        },
        {
          id: "security-review",
          label: intl.formatMessage({
            defaultMessage: "Security review assistance",
            id: "EJHq4HC1h1",
            description: "Pricing matrix row label for security review assistance",
          }),
          cells: {
            free: { kind: "dash" },
            starter: { kind: "dash" },
            growth: { kind: "dash" },
            enterprise: { kind: "check" },
          },
        },
      ],
    },
  ];
}

export type PricingAiFeature = {
  id: string;
  title: string;
  description: string;
};

export function getPricingAiFeatures(locale: string): PricingAiFeature[] {
  const intl = getIntlShape(locale);

  return [
    {
      id: "ask-about-any-string",
      title: intl.formatMessage({
        defaultMessage: "Ask about any string",
        id: "rDFwnPnMxA",
        description: "AI feature capability title: ask about strings",
      }),
      description: intl.formatMessage({
        defaultMessage: "Get meaning, where it appears in the product, and how to translate it.",
        id: "GJZJnd3J8t",
        description: "AI feature capability body: ask about strings",
      }),
    },
    {
      id: "visual-context",
      title: intl.formatMessage({
        defaultMessage: "See the UI context",
        id: "2Y27cUhCTS",
        description: "AI feature capability title: visual context",
      }),
      description: intl.formatMessage({
        defaultMessage: "Screenshots so reviewers know where copy shows up.",
        id: "VFYJ+T/TTl",
        description: "AI feature capability body: visual context",
      }),
    },
    {
      id: "translate-in-chat",
      title: intl.formatMessage({
        defaultMessage: "Translate in chat",
        id: "rw4HCHelCR",
        description: "AI feature capability title: translate in chat",
      }),
      description: intl.formatMessage({
        defaultMessage: "Send files, text, or images in web, Slack, or email.",
        id: "9acpAUlA8Z",
        description: "AI feature capability body: translate in chat",
      }),
    },
    {
      id: "recent-change-briefs",
      title: intl.formatMessage({
        defaultMessage: "Catch what changed",
        id: "YNbxvilhkH",
        description: "AI feature capability title: recent change briefs",
      }),
      description: intl.formatMessage({
        defaultMessage: "Briefs on new or updated source copy, with context.",
        id: "36xVSohq0P",
        description: "AI feature capability body: recent change briefs",
      }),
    },
    {
      id: "organization-memory",
      title: intl.formatMessage({
        defaultMessage: "Remember your rules",
        id: "sVUx4aIt3x",
        description: "AI feature capability title: organization memory",
      }),
      description: intl.formatMessage({
        defaultMessage: "Keep tone and terminology guidance for the next job.",
        id: "hDNxzYjQkl",
        description: "AI feature capability body: organization memory",
      }),
    },
    {
      id: "tms-aware-drafting-qa",
      title: intl.formatMessage({
        defaultMessage: "Draft and check in your TMS",
        id: "hIOpWNAWDG",
        description: "AI feature capability title: TMS-aware drafting and QA",
      }),
      description: intl.formatMessage({
        defaultMessage: "Fill missing locales and run QA while people stay in control.",
        id: "6t3EqSwEpq",
        description: "AI feature capability body: TMS-aware drafting and QA",
      }),
    },
    {
      id: "agent-automations",
      title: intl.formatMessage({
        defaultMessage: "Automate the busywork",
        id: "iOJxHuvYs1",
        description: "AI feature capability title: agent automations",
      }),
      description: intl.formatMessage({
        defaultMessage:
          "Sync, validate, and notify the team on a schedule or when something lands.",
        id: "jNUm0wTEWX",
        description: "AI feature capability body: agent automations",
      }),
    },
    {
      id: "bring-your-own-llm",
      title: intl.formatMessage({
        defaultMessage: "Use the model you prefer",
        id: "hz8O9/ilG1",
        description: "AI feature capability title: bring your own LLM",
      }),
      description: intl.formatMessage({
        defaultMessage: "OpenAI, Anthropic, or Gemini — without changing how you work.",
        id: "kmECxTtxS5",
        description: "AI feature capability body: bring your own LLM",
      }),
    },
  ];
}

export function getPricingPageCopy(locale: string) {
  const intl = getIntlShape(locale);

  return {
    headline: intl.formatMessage({
      defaultMessage: "Scale localisation. Control your costs.",
      id: "gfUQK+bPpD",
      description: "Primary headline on the marketing pricing page",
    }),
    subcopy: intl.formatMessage({
      defaultMessage: "Simple plans for evaluating Hyperlocalise today.",
      id: "qRVA5sizhD",
      description: "Supporting copy under the pricing page headline",
    }),
    popularBadge: intl.formatMessage({
      defaultMessage: "Popular",
      id: "BhWHI3Tdnm",
      description: "Badge label on the featured Growth pricing plan",
    }),
    compareHeading: intl.formatMessage({
      defaultMessage: "Compare plans",
      id: "kqDNTw/sEr",
      description: "Heading above the pricing comparison matrix",
    }),
    compareSubcopy: intl.formatMessage({
      defaultMessage: "See what each plan includes before you choose a path.",
      id: "wNfaNj8oeP",
      description: "Supporting copy above the pricing comparison matrix",
    }),
    includedAriaLabel: intl.formatMessage({
      defaultMessage: "Included",
      id: "xEaoAlugPJ",
      description: "Accessible label for a checkmark in the pricing matrix",
    }),
    notIncludedAriaLabel: intl.formatMessage({
      defaultMessage: "Not included",
      id: "z9cqkREGyZ",
      description: "Accessible label for a dash in the pricing matrix",
    }),
    aiFeaturesHeading: intl.formatMessage({
      defaultMessage: "AI features",
      id: "Lmup4mp1Uh",
      description: "Heading for the AI features explainer on the pricing page",
    }),
    aiFeaturesSubcopy: intl.formatMessage({
      defaultMessage:
        "Included on Growth and Enterprise. Clear answers on your copy, UI context, and localisation workflow.",
      id: "KHXRS/XkKV",
      description: "Supporting copy under the AI features heading on the pricing page",
    }),
    undecidedHeading: intl.formatMessage({
      defaultMessage: "Can't decide?",
      id: "dOaxbDzrv5",
      description: "Heading for the undecided CTA band below the pricing FAQ",
    }),
    talkToSales: intl.formatMessage({
      defaultMessage: "Talk to sales",
      id: "MZ5aQR65ew",
      description: "Secondary CTA on the undecided pricing band",
    }),
    requestDemo: intl.formatMessage({
      defaultMessage: "Request a demo",
      id: "+iY8hxQCxr",
      description: "Primary CTA on the undecided pricing band",
    }),
  };
}

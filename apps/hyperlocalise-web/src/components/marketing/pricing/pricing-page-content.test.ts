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
import { describe, expect, it } from "vite-plus/test";

import { buildPricingFaqJsonLd, getPricingFaqItems } from "./pricing-faq-content";
import {
  getPricingAiFeatures,
  getPricingMatrixSections,
  getPricingPlans,
  pricingPlanOrder,
} from "./pricing-page-content";

describe("pricing page content", () => {
  it("exposes four plans with coming-soon CTAs except Enterprise demo", () => {
    const plans = getPricingPlans("en");

    expect(plans.map((plan) => plan.id)).toEqual([...pricingPlanOrder]);
    expect(plans.find((plan) => plan.id === "free")?.features).toEqual([
      "2 integrations",
      "1 project",
      "1 seat",
    ]);
    expect(plans.find((plan) => plan.id === "starter")?.price).toBe("$20");
    expect(plans.find((plan) => plan.id === "growth")?.price).toBe("$2,000");
    expect(plans.find((plan) => plan.id === "growth")?.popular).toBe(true);
    expect(plans.find((plan) => plan.id === "growth")?.features).toEqual([
      "2,000 agent runs per month",
      "2,000 AI tokens per month",
      "20 agent automations",
      "5 integrations",
      "Unlimited seats",
      "Unlimited translation jobs",
      "AI features",
    ]);
    expect(plans.filter((plan) => plan.cta.kind === "coming_soon")).toHaveLength(3);
    expect(plans.find((plan) => plan.id === "enterprise")?.cta).toEqual({
      kind: "demo",
      label: "Contact Sales",
    });
  });

  it("builds a matrix covering every plan column", () => {
    const sections = getPricingMatrixSections("en");
    const rows = sections.flatMap((section) => section.rows);

    expect(sections.length).toBeGreaterThan(0);
    expect(rows.find((row) => row.id === "automations")?.label).toBe("Agent Automations");
    for (const row of rows) {
      for (const planId of pricingPlanOrder) {
        expect(row.cells[planId]).toBeTruthy();
      }
    }
  });

  it("lists eight AI feature capabilities", () => {
    const features = getPricingAiFeatures("en");

    expect(features).toHaveLength(8);
    expect(features.map((feature) => feature.id)).toEqual([
      "ask-about-any-string",
      "visual-context",
      "translate-in-chat",
      "recent-change-briefs",
      "organization-memory",
      "tms-aware-drafting-qa",
      "agent-automations",
      "bring-your-own-llm",
    ]);
    expect(features.map((feature) => feature.title)).toEqual([
      "Ask about any string",
      "See the UI context",
      "Translate in chat",
      "Catch what changed",
      "Remember your rules",
      "Draft and check in your TMS",
      "Automate the busywork",
      "Use the model you prefer",
    ]);
    expect(features.map((feature) => feature.description)).toEqual([
      "Get meaning, where it appears in the product, and how to translate it.",
      "Screenshots so reviewers know where copy shows up.",
      "Send files, text, or images in web, Slack, or email.",
      "Briefs on new or updated source copy, with context.",
      "Keep tone and terminology guidance for the next job.",
      "Fill missing locales and run QA while people stay in control.",
      "Sync, validate, and notify the team on a schedule or when something lands.",
      "OpenAI, Anthropic, or Gemini — without changing how you work.",
    ]);
  });

  it("builds matching FAQ content and FAQPage structured data", () => {
    const items = getPricingFaqItems("en");
    const jsonLd = buildPricingFaqJsonLd(items);

    expect(items.length).toBeGreaterThan(0);
    expect(jsonLd).toMatchObject({
      "@type": "FAQPage",
      mainEntity: items.map((item) => ({
        "@type": "Question",
        name: item.question,
        acceptedAnswer: {
          "@type": "Answer",
          text: item.answer,
        },
      })),
    });
  });
});

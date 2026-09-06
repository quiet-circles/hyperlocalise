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
import type { FAQPage, WithContext } from "schema-dts";

import { getIntlShape } from "@/lib/app-i18n/intl";

export type HomepageFaqItem = {
  question: string;
  answer: string;
};

export function getHomepageFaqItems(locale: string): HomepageFaqItem[] {
  const intl = getIntlShape(locale);

  return [
    {
      question: intl.formatMessage({
        defaultMessage: "What is Hyperlocalise?",
        id: "zw8s3RUBzA",
        description: "Homepage FAQ question about platform topic 1",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Hyperlocalise is AI-native infrastructure for multilingual content operations. Content Studio, Automation Workflow, Domains, Hyperlab, and Guidelines connect content creation, orchestration, publishing, and optimisation.",
        id: "5Ad03j9j5u",
        description: "Homepage FAQ answer about platform topic 1",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "What can my team do in Content Studio?",
        id: "UxoXtyoyQB",
        description: "Homepage FAQ question about platform topic 2",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Create and refine multilingual content in a shared workspace, with context and human review close at hand.",
        id: "f05Tt/0FpU",
        description: "Homepage FAQ answer about platform topic 2",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "How does Automation Workflow fit into the platform?",
        id: "FS53HcU/f1",
        description: "Homepage FAQ question about platform topic 3",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Automation Workflow connects tasks, AI agents, and human review into repeatable content operations.",
        id: "+IYjyCmBJt",
        description: "Homepage FAQ answer about platform topic 3",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "What is Domains?",
        id: "Y+KPva7CsX",
        description: "Homepage FAQ question about platform topic 4",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Domains is for content publishing: a CMS with automated AEO and SEO operations to help teams manage content and discovery across markets.",
        id: "wyGfdlT+34",
        description: "Homepage FAQ answer about platform topic 4",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "What is Hyperlab?",
        id: "DzGtKmIsFh",
        description: "Homepage FAQ question about platform topic 5",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Hyperlab provides A/B testing for your CMS and distribution, so your team can compare content variants and learn what performs.",
        id: "3vl9Ga2VYG",
        description: "Homepage FAQ answer about platform topic 5",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "How do Guidelines support the work?",
        id: "hqrN/Zh7Ck",
        description: "Homepage FAQ question about platform topic 6",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Guidelines provide shared brand guidance, terminology, and market context for your team and AI agents throughout content operations.",
        id: "Jk6f+RV4Ho",
        description: "Homepage FAQ answer about platform topic 6",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "Can we ask an agent to do the work?",
        id: "2ioZpgZgy2",
        description: "Homepage FAQ question about platform topic 7",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Yes. Hyperlocalise agents live in integrations such as Slack, Teams, and GitHub, so teams can ask for work where they already collaborate. Available actions depend on the integration and workspace configuration.",
        id: "3URR9qLWsH",
        description: "Homepage FAQ answer about platform topic 7",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "Can we connect our own AI tools?",
        id: "QWHemDFkex",
        description: "Homepage FAQ question about platform topic 8",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Yes. MCP gives compatible AI assistants access to Hyperlocalise capabilities, bringing content operations into your existing AI workspace.",
        id: "G8YOSjr1Lc",
        description: "Homepage FAQ answer about platform topic 8",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "Do I need to replace my TMS?",
        id: "owW1Hvk/pe",
        description: "Homepage FAQ question about platform topic 9",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "No. Hyperlocalise works alongside your existing TMS. Keep using Phrase, Lokalise, Crowdin or Smartling while Hyperlocalise orchestrates AI agents, review workflows, context discovery and quality checks.",
        id: "dAN96XLBhS",
        description: "Homepage FAQ answer about platform topic 9",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "Can I use my preferred AI models?",
        id: "z5PrXgdEMz",
        description: "Homepage FAQ question about platform topic 10",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Yes. Hyperlocalise is LLM-agnostic. Use OpenAI, Anthropic, Gemini or your preferred provider, and switch models without changing your workflow.",
        id: "jXzA8Zex1U",
        description: "Homepage FAQ answer about platform topic 10",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "Who is Hyperlocalise built for?",
        id: "/mP31XTUTm",
        description: "Homepage FAQ question about platform topic 11",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Content, localisation, marketing, and development teams working together to create, publish, and improve multilingual content.",
        id: "8KBorulqxU",
        description: "Homepage FAQ answer about platform topic 11",
      }),
    },
    {
      question: intl.formatMessage({
        defaultMessage: "How can we get started?",
        id: "gXu61zDcr/",
        description: "Homepage FAQ question about platform topic 12",
      }),
      answer: intl.formatMessage({
        defaultMessage:
          "Request a demo to explore the products and discuss how Hyperlocalise can fit your team’s content operations and existing tools.",
        id: "zPGts4kyVb",
        description: "Homepage FAQ answer about platform topic 12",
      }),
    },
  ];
}

export function buildHomepageFaqJsonLd(items: readonly HomepageFaqItem[]): WithContext<FAQPage> {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: items.map((item) => ({
      "@type": "Question",
      name: item.question,
      acceptedAnswer: {
        "@type": "Answer",
        text: item.answer,
      },
    })),
  };
}

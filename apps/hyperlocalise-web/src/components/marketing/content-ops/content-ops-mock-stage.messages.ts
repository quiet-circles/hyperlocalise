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
import { defineMessages } from "react-intl";

export const contentOpsMockStageMessages = defineMessages({
  autoplayPause: {
    defaultMessage: "Pause demo rotation",
    id: "VbgjobTz9L",
    description: "Accessible label for pausing content ops tab autoplay",
  },
  autoplayResume: {
    defaultMessage: "Resume demo rotation",
    id: "vf2HgEHa0O",
    description: "Accessible label for resuming content ops tab autoplay",
  },

  mockWorkspaceName: {
    defaultMessage: "Acme Corp",
    id: "ZpIy+48cVo",
    description: "Workspace name in the marketing app shell mock",
  },
  mockNavInbox: {
    defaultMessage: "Inbox",
    id: "Aicm++b5Ie",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockNavIssues: {
    defaultMessage: "Board",
    id: "2Q9B8y1llx",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockNavDashboard: {
    defaultMessage: "Overview",
    id: "FBzKoGNaC+",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockNavProjects: {
    defaultMessage: "Projects",
    id: "WLVv78wgsr",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockNavAutomations: {
    defaultMessage: "Automations",
    id: "FG27myF7Bk",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockNavKnowledge: {
    defaultMessage: "Knowledge",
    id: "Ehcjh40kDc",
    description: "Sidebar nav label in marketing app shell mock",
  },
  mockBreadcrumbInbox: {
    defaultMessage: "Acme · Inbox",
    id: "HD1VQg9qxJ",
    description: "Header breadcrumb in marketing app shell mock for inbox triage",
  },
  mockBreadcrumbIssues: {
    defaultMessage: "Acme · Board",
    id: "08+4Nn7rgH",
    description: "Header breadcrumb in marketing app shell mock",
  },
  mockBreadcrumbCampaign: {
    defaultMessage: "Acme · Campaign brief",
    id: "Dz/TZAVlPR",
    description: "Header breadcrumb in marketing app shell mock",
  },
  mockBreadcrumbSeo: {
    defaultMessage: "Acme · Multilingual blog workflow",
    id: "k5tXS6aCpT",
    description: "Header breadcrumb in marketing app shell mock",
  },
  mockBreadcrumbBrand: {
    defaultMessage: "Acme · Brand review",
    id: "nn0iMJAwx4",
    description: "Header breadcrumb in marketing app shell mock",
  },
  mockBreadcrumbEditor: {
    defaultMessage: "Acme · Web launch · Editor",
    id: "Zol/03gW7r",
    description: "Header breadcrumb in marketing app shell mock",
  },
  mockShellPlan: {
    defaultMessage: "Pro · 12k strings remaining",
    id: "LGsstatwcF",
    description: "Footer plan summary in marketing app shell mock",
  },
  mockShellPlanButton: {
    defaultMessage: "Pro",
    id: "RKym//U9I4",
    description: "Plan button label in marketing app shell mock editor footer",
  },
  mockShellSupport: {
    defaultMessage: "Support",
    id: "+wkfFZiCrl",
    description: "Footer support link label in marketing app shell mock",
  },

  tabTriage: {
    defaultMessage: "Triage open questions",
    id: "Eb4afW9TDL",
    description: "Content ops mock tab label for issues triage",
  },
  tabCampaign: {
    defaultMessage: "Localize campaign copy",
    id: "756ODq6uhK",
    description: "Content ops mock tab label for campaign localisation",
  },
  tabSeoBlog: {
    defaultMessage: "Publish Multilingual Blog",
    id: "OU5qhClfKo",
    description: "Content ops mock tab label for SEO blog publishing",
  },
  tabBrand: {
    defaultMessage: "Brand Usage Check",
    id: "Cef/Y5xN6Q",
    description: "Content ops mock tab label for brand governance",
  },
  tabEditor: {
    defaultMessage: "Content Studio",
    id: "B5P0vwqbH1",
    description: "Content ops mock tab label for the CAT file content editor",
  },

  editorSceneFileContent: {
    defaultMessage: "Review file content",
    id: "kXJXJXMmTt",
    description: "Editor mock use case pill for reviewing source file strings",
  },
  editorSceneGlossary: {
    defaultMessage: "Glossary checks",
    id: "kM/dmaarD3",
    description: "Editor mock use case pill for glossary term enforcement",
  },
  editorSceneIssues: {
    defaultMessage: "Linked issues",
    id: "K76rTMej1a",
    description: "Editor mock use case pill for issues linked to strings",
  },
  editorSceneIntelligence: {
    defaultMessage: "Translation intelligence",
    id: "zjJkVSEvVu",
    description: "Editor mock use case pill for TM, context, and AI suggestions",
  },
  editorIssuesPanelTitle: {
    defaultMessage: "Board · this string",
    id: "hYi0NizHJf",
    description: "Title on the editor mock board overlay",
  },
  editorHighlightEditor: {
    defaultMessage: "File editor",
    id: "/ApELcd47S",
    description: "Highlight badge on the CAT file content editor panel",
  },
  editorHighlightGlossary: {
    defaultMessage: "Glossary",
    id: "hmabVdb4tH",
    description: "Highlight badge on the glossary section in intelligence panel",
  },
  editorHighlightIntelligence: {
    defaultMessage: "Intelligence",
    id: "E451cDD6K/",
    description: "Highlight badge on the translation intelligence panel",
  },

  botLabel: {
    defaultMessage: "Use Hyperlocalise Agent",
    id: "bH9+5GUIgE",
    description: "Agent terminal title in content ops mock",
  },
  agentRunLabel: {
    defaultMessage: "Live run",
    id: "wVJMgYMlXo",
    description: "Live execution panel label in content ops agent mock",
  },

  campaignAutomationName: {
    defaultMessage: "Q2 GTM launch",
    id: "1N4AtoHPR5",
    description: "Automation name in campaign agent setup mock",
  },
  campaignInstructions: {
    defaultMessage:
      "When a GTM brief is approved, localize landing page copy for each target market. Draft FR and DE first, route for review, publish to staging, and notify #gtm.",
    id: "+TAJxozALX",
    description: "Agent instructions in campaign setup mock",
  },
  seoAutomationName: {
    defaultMessage: "Monthly SEO blogs",
    id: "bEs1gxswwN",
    description: "Automation name in SEO agent setup mock",
  },
  seoInstructions: {
    defaultMessage:
      "On the 1st of each month, research high-intent keyword gaps per locale, draft localised SEO posts with adapted meta and H1, run QA, write to CMS, and notify #content.",
    id: "b8s+83F3Ce",
    description: "Agent instructions in SEO setup mock",
  },

  toolCmsDescription: {
    defaultMessage: "Publish localized drafts to staging.",
    id: "/S9UR8V526",
    description: "CMS tool description in content ops agent setup mock",
  },
  toolTranslateDescription: {
    defaultMessage: "Generate locale-adapted copy from the brief.",
    id: "PkcpFYcLp9",
    description: "Translate tool description in content ops agent setup mock",
  },
  toolSlackGtmDescription: {
    defaultMessage: "Notify #gtm when drafts are ready for review.",
    id: "lkoT8UVKCp",
    description: "Slack tool description in campaign agent setup mock",
  },
  toolSearchDescription: {
    defaultMessage: "Compare search volume and intent across locales.",
    id: "QO4Q6r8vEP",
    description: "Search tool description in SEO agent setup mock",
  },
  toolAhrefsDescription: {
    defaultMessage: "Pull keyword gap data for target markets.",
    id: "DXeIUKV+HK",
    description: "Ahrefs tool description in SEO agent setup mock",
  },
  toolSlackSeoDescription: {
    defaultMessage: "Notify #content when drafts land in CMS.",
    id: "F4InMNOq7F",
    description: "Slack tool description in SEO agent setup mock",
  },

  triggerGtmBrief: {
    defaultMessage: "GTM brief approved · Q2 launch",
    id: "MNCmDu8OgB",
    description: "GTM trigger label in content ops campaign mock",
  },
  triggerSeoSchedule: {
    defaultMessage: "Scheduled run · 1st of month",
    id: "Tl0CYs/kQf",
    description: "SEO schedule trigger label in content ops mock",
  },

  toolCms: {
    defaultMessage: "CMS",
    id: "Zrmv5cA+Lc",
    description: "CMS tool chip in content ops terminal mock",
  },
  toolTranslate: {
    defaultMessage: "Translate",
    id: "mWl7zKHq+b",
    description: "Translate tool chip in content ops terminal mock",
  },
  toolSlack: {
    defaultMessage: "Slack",
    id: "AIFi7wN5lY",
    description: "Slack tool chip in content ops terminal mock",
  },
  toolSearch: {
    defaultMessage: "Search",
    id: "QgqbdcDGCR",
    description: "Search tool chip in content ops terminal mock",
  },
  toolAhrefs: {
    defaultMessage: "Ahrefs",
    id: "eQIxzTUwry",
    description: "Ahrefs tool chip in content ops terminal mock",
  },

  stepGtm1: {
    defaultMessage: "Brief received · 4 markets, 12 assets",
    id: "mlbMDyLQpv",
    description: "Campaign mock step 1",
  },
  stepGtm2: {
    defaultMessage: "Generating localized landing page drafts...",
    id: "EVH8TUWi0h",
    description: "Campaign mock step 2",
  },
  stepGtm3: {
    defaultMessage: "FR and DE routed for review",
    id: "6JT4j+EKwy",
    description: "Campaign mock step 3",
  },
  stepGtm4: {
    defaultMessage: "Published to staging · notified #gtm",
    id: "ZV8a5iOSNX",
    description: "Campaign mock step 4",
  },

  stepSeo1: {
    defaultMessage: "Pulling search volume for core product terms",
    id: "nlurA4/IRm",
    description: "SEO blog mock step 1",
  },
  stepSeo2: {
    defaultMessage: "EN · FR · DE · JA demand compared",
    id: "a0ooXAWwaq",
    description: "SEO blog mock step 2",
  },
  stepSeo3: {
    defaultMessage: "12 high-intent gaps found in DE",
    id: "tIIkL73dX1",
    description: "SEO blog mock step 3",
  },
  stepSeo4: {
    defaultMessage: "Localised SEO draft · meta + H1 adapted",
    id: "Wnrfau6YGS",
    description: "SEO blog mock step 4",
  },
  stepSeo5: {
    defaultMessage: "QA passed · draft written to CMS · #content notified",
    id: "ieOgNf/WnJ",
    description: "SEO blog mock step 5",
  },

  issuesTitle: {
    defaultMessage: "Board · acme workspace",
    id: "tX5h+jHyR2",
    description: "Board panel title in content ops mock",
  },
  inboxTitle: {
    defaultMessage: "Inbox",
    id: "TYvjU7uCVI",
    description: "Inbox panel title in content ops mock",
  },
  inboxConvDeCtaTitle: {
    defaultMessage: "DE checkout CTA review",
    id: "oLZRmhKHIV",
    description: "Inbox conversation title in content ops mock",
  },
  inboxConvDeCtaPreview: {
    defaultMessage: "Does this CTA follow our brand voice for German checkout?",
    id: "9J33Cf6m0j",
    description: "Inbox conversation preview in content ops mock",
  },
  inboxConvDeCtaMeta: {
    defaultMessage: "Slack · 2h ago",
    id: "NQK86OKSk0",
    description: "Inbox conversation meta in content ops mock",
  },
  inboxConvGlossaryTitle: {
    defaultMessage: "Onboarding glossary check",
    id: "P0tbpI+zHV",
    description: "Inbox conversation title in content ops mock",
  },
  inboxConvGlossaryPreview: {
    defaultMessage: "Should AcmePay stay untranslated in ES onboarding?",
    id: "d1YNCoenDE",
    description: "Inbox conversation preview in content ops mock",
  },
  inboxConvGlossaryMeta: {
    defaultMessage: "Email · yesterday",
    id: "fsHdGSEIEq",
    description: "Inbox conversation meta in content ops mock",
  },
  inboxIssueWeb2Preview: {
    defaultMessage: "Mina assigned you · Payment button label too long",
    id: "19Hyjf99DQ",
    description: "Inbox issue notification preview in content ops mock",
  },
  inboxIssueWeb2Description: {
    defaultMessage:
      "The French payment button exceeds the 32-character limit in checkout.json. Review the translation and shorten the CTA while keeping meaning intact.",
    id: "sFgi3RFLqg",
    description: "Inbox issue detail description in content ops mock",
  },
  inboxIssueWeb2Project: {
    defaultMessage: "Web launch",
    id: "j53Gofjd1b",
    description: "Inbox issue project label in content ops mock",
  },
  inboxIssueNotificationMeta: {
    defaultMessage: "Issue update · 45m ago",
    id: "sQA+edg3pH",
    description: "Inbox issue notification meta in content ops mock",
  },
  inboxIssueProjectLabel: {
    defaultMessage: "Project",
    id: "TWitJgQdIP",
    description: "Inbox issue detail project label in content ops mock",
  },
  inboxIssueSourceLabel: {
    defaultMessage: "Source file",
    id: "wHFmp4/+wj",
    description: "Inbox issue detail source label in content ops mock",
  },
  inboxIssueCommentAuthor: {
    defaultMessage: "Mina Chen",
    id: "urSbyPXHeu",
    description: "Inbox issue comment author in content ops mock",
  },
  inboxIssueComment: {
    defaultMessage: "The FR payment button still overflows the 32-character limit on mobile.",
    id: "5hFx7hW2gZ",
    description: "Inbox issue comment body in content ops mock",
  },
  inboxAssistantTitle: {
    defaultMessage: "Hyperlocalise Agent",
    id: "EeV19O9AuX",
    description: "Inbox assistant panel title in content ops mock",
  },
  inboxAssistantDeCtaQuestion: {
    defaultMessage: 'Does "Nutzen Sie unsere innovative Plattform" follow our DE brand voice?',
    id: "/WYfcQHn0k",
    description: "Inbox assistant user question in content ops mock",
  },
  inboxAssistantDeCtaAnswer: {
    defaultMessage:
      'Off-brand for checkout — too formal. Use a short verb CTA like "Jetzt starten" (max 24 characters).',
    id: "fOCWpBnaX6",
    description: "Inbox assistant answer in content ops mock",
  },
  inboxAssistantWeb2Question: {
    defaultMessage: "What's the fastest fix for the FR checkout button length issue?",
    id: "NIHlitteoo",
    description: "Inbox assistant user question for issue in content ops mock",
  },
  inboxAssistantWeb2Answer: {
    defaultMessage:
      'Shorten to "Payer maintenant" (16 chars). Open checkout.json in the Content Editor to apply and re-run QA.',
    id: "7A4K42htKk",
    description: "Inbox assistant answer for issue in content ops mock",
  },
  inboxAssistantGlossaryQuestion: {
    defaultMessage: "Should AcmePay stay untranslated in Spanish onboarding?",
    id: "/QZxo6jgZM",
    description: "Inbox assistant glossary question in content ops mock",
  },
  inboxAssistantGlossaryAnswer: {
    defaultMessage:
      'Yes — glossary rule "Product names: never translate" applies. Flag MOB-1 and link the onboarding strings.',
    id: "SS1sIH3qu1",
    description: "Inbox assistant glossary answer in content ops mock",
  },
  issuesSummary: {
    defaultMessage: "{open} open · {inProgress} in progress · {resolved} resolved",
    id: "MMFb5MTldp",
    description: "Issues summary line in content ops mock",
  },
  issueWeb2Title: {
    defaultMessage: "Translation mistake in checkout",
    id: "LE5VKYLxU+",
    description: "Issue row title in content ops mock",
  },
  issueWeb2Detail: {
    defaultMessage: "Payment button label too long · checkout.json",
    id: "Mk8D3Ht2wh",
    description: "Issue row detail in content ops mock",
  },
  issueMob1Title: {
    defaultMessage: "Glossary violation in onboarding",
    id: "qjpFG9aLeM",
    description: "Issue row title in content ops mock",
  },
  issueMob1Detail: {
    defaultMessage: "Product name should stay untranslated",
    id: "pHD22l59Uc",
    description: "Issue row detail in content ops mock",
  },
  issueWeb3Title: {
    defaultMessage: "QA failure on hero headline",
    id: "2a9969NmAV",
    description: "Issue row title in content ops mock",
  },
  issueWeb3Detail: {
    defaultMessage: "Length check failed for German headline",
    id: "PvTDEYL5Rz",
    description: "Issue row detail in content ops mock",
  },
  statusOpen: {
    defaultMessage: "Open",
    id: "BA/hJPgmbP",
    description: "Issue status open",
  },
  statusInProgress: {
    defaultMessage: "In progress",
    id: "MXGPxAzEmG",
    description: "Issue status in progress",
  },
  statusResolved: {
    defaultMessage: "Resolved",
    id: "e+WPqvuP5X",
    description: "Issue status resolved",
  },
  openInContentEditor: {
    defaultMessage: "Open in Content Editor",
    id: "y9MQtQc7gb",
    description: "Link label to open issue in the Content Editor",
  },

  brandStyleTitle: {
    defaultMessage: "Brand voice · Style guide",
    id: "wr3AKpuOi3",
    description: "Brand style guide panel title",
  },
  brandStyleSubtitle: {
    defaultMessage: "Applied while reviewing DE checkout CTA",
    id: "zL0L4Vvw8O",
    description: "Brand style guide panel subtitle",
  },
  brandRuleTone: {
    defaultMessage: "Tone: friendly, direct",
    id: "6ZNG5E8172",
    description: "Brand style rule chip",
  },
  brandRuleCta: {
    defaultMessage: "CTA: short verb",
    id: "4yshvtwLy5",
    description: "Brand style rule chip",
  },
  brandGuidelineToneTitle: {
    defaultMessage: "Tone",
    id: "SBN/+r4YAI",
    description: "Brand guideline tone section title in content ops mock",
  },
  brandGuidelineToneBody: {
    defaultMessage:
      "Friendly and direct. Avoid formal Sie-form and jargon in checkout flows. Write like you are helping a customer finish quickly.",
    id: "EbI5T06YK7",
    description: "Brand guideline tone body in content ops mock",
  },
  brandGuidelineCtaTitle: {
    defaultMessage: "Call to action",
    id: "RrEtMWT3+n",
    description: "Brand guideline CTA section title in content ops mock",
  },
  brandGuidelineCtaBody: {
    defaultMessage:
      'Use a short verb. Maximum 24 characters. Prefer "Jetzt starten" over descriptive marketing phrases.',
    id: "3wZP3Mai9s",
    description: "Brand guideline CTA body in content ops mock",
  },
  brandGuidelineTermsTitle: {
    defaultMessage: "Product terms",
    id: "z2BXL0WPeh",
    description: "Brand guideline terms section title in content ops mock",
  },
  brandGuidelineTermsBody: {
    defaultMessage: "Keep product names such as AcmePay untranslated across all markets.",
    id: "Ga2xU2YzE3",
    description: "Brand guideline terms body in content ops mock",
  },
  brandUploadedGuideLabel: {
    defaultMessage: "Uploaded style guide",
    id: "wkXL+SvV0U",
    description: "Label for uploaded PDF in brand mock",
  },
  brandUploadedGuideFilename: {
    defaultMessage: "brand-voice-style-guide.pdf",
    id: "szjULYmU5z",
    description: "Filename for uploaded PDF in brand mock",
  },
  brandUploadedGuideSize: {
    defaultMessage: "2.4 MB · 12 pages",
    id: "anFNHwBPyz",
    description: "File size for uploaded PDF in brand mock",
  },
  brandUploadedGuideExcerpt: {
    defaultMessage:
      "DE checkout CTAs should use informal du, stay under 24 characters, and lead with a verb.",
    id: "62TJERGYuU",
    description: "Excerpt text shown in PDF preview in brand mock",
  },
  brandBeforeLabel: {
    defaultMessage: "Before",
    id: "IuT/Zm3PzS",
    description: "Before copy label in brand mock",
  },
  brandAfterLabel: {
    defaultMessage: "After",
    id: "oiaT335RnA",
    description: "After copy label in brand mock",
  },
  brandBeforeCopy: {
    defaultMessage: "Nutzen Sie unsere innovative Plattform",
    id: "44BRBm86tt",
    description: "Before copy sample in brand mock",
  },
  brandAfterCopy: {
    defaultMessage: "Jetzt starten",
    id: "EwGaDUWfx3",
    description: "After copy sample in brand mock",
  },
  brandAppliedBadge: {
    defaultMessage: "Applied",
    id: "mq4AvDs4i5",
    description: "Applied badge on brand style correction",
  },
  brandChatTitle: {
    defaultMessage: "Brand review chat",
    id: "US+0MJLvUO",
    description: "Brand chat dock title",
  },
  brandChatEmptyTitle: {
    defaultMessage: "Ask about brand voice",
    id: "0VtfeCWDjZ",
    description: "Empty state title in brand chat mock",
  },
  brandChatEmptySubtitle: {
    defaultMessage: "Check copy against style guides and market rules before publish.",
    id: "O/4w7DJ8Jn",
    description: "Empty state subtitle in brand chat mock",
  },
  brandSuggestionCta: {
    defaultMessage: "Review this German CTA",
    id: "6fJrfRBNdK",
    description: "Suggestion chip that starts the brand chat mock",
  },
  brandContextPill: {
    defaultMessage: "Brand voice guide",
    id: "ngwhjoKY67",
    description: "Context pill in brand chat mock composer",
  },
  brandComposerPlaceholder: {
    defaultMessage: "Ask a follow-up…",
    id: "Fm3OW372/a",
    description: "Composer placeholder in brand chat mock after send",
  },
  brandCollapseLabel: {
    defaultMessage: "Collapse chat",
    id: "DQNOkP5v4I",
    description: "Decorative collapse control in brand chat mock",
  },
  brandCloseLabel: {
    defaultMessage: "Close chat",
    id: "5n8Zj0Pp9F",
    description: "Decorative close control in brand chat mock",
  },
  brandChatPrompt: {
    defaultMessage:
      'Does this German CTA follow our brand guidelines? "Nutzen Sie unsere innovative Plattform"',
    id: "DYjZeqplYa",
    description: "User prompt in brand chat mock",
  },
  brandToolName: {
    defaultMessage: "recall_knowledge_files",
    id: "U9t/eIwVYM",
    description: "Tool name shown in brand chat mock",
  },
  brandToolDetail: {
    defaultMessage: "brand-voice-style-guide.pdf",
    id: "/4z8WJXCPb",
    description: "Tool detail in brand chat mock",
  },
  brandVerdictLabel: {
    defaultMessage: "Verdict",
    id: "wi8LFLVfQY",
    description: "Verdict section in brand chat answer",
  },
  brandVerdictBody: {
    defaultMessage: "Off-brand — too formal and jargon-heavy for DE checkout.",
    id: "I/IBBrTtGZ",
    description: "Verdict body in brand chat answer",
  },
  brandGuidelineLabel: {
    defaultMessage: "Guideline",
    id: "d3ASNb05IP",
    description: "Guideline section in brand chat answer",
  },
  brandGuidelineBody: {
    defaultMessage: "Tone: friendly, direct · CTA: short verb, max 24 characters.",
    id: "MVrchtGazs",
    description: "Guideline body in brand chat answer",
  },
  brandSuggestLabel: {
    defaultMessage: "Suggested copy",
    id: "DsRoK8kvzv",
    description: "Suggested copy section in brand chat answer",
  },
  brandSuggestBody: {
    defaultMessage: "Jetzt starten",
    id: "jn5bLV1i0/",
    description: "Suggested copy body in brand chat answer",
  },
  brandSend: {
    defaultMessage: "Send",
    id: "N0qPsNFkOT",
    description: "Send button in brand chat mock",
  },
  brandReplay: {
    defaultMessage: "Replay",
    id: "aRwWx+LFBU",
    description: "Replay button in brand chat mock",
  },

  flowTitle: {
    defaultMessage: "Multilingual blog · workflow",
    id: "7DodroC7Kp",
    description: "Flow panel title",
  },
  flowNodeKindTrigger: {
    defaultMessage: "Trigger",
    id: "znMxHXQMhI",
    description: "Workflow node type label for triggers in content ops flow mock",
  },
  flowNodeKindAction: {
    defaultMessage: "Action",
    id: "h08QDRNRM3",
    description: "Workflow node type label for actions in content ops flow mock",
  },
  flowBriefDescription: {
    defaultMessage:
      "On schedule, research keywords, draft content, localise, review, then notify Slack and publish to CMS.",
    id: "zVfLTcACWy",
    description: "Workflow description for multilingual blog template in content ops mock",
  },
  flowCampaignDescription: {
    defaultMessage: "Launch copy to staging and notify the team when review clears.",
    id: "oXTxX56WPY",
    description: "Workflow description for campaign template in content ops mock",
  },
  flowSeoDescription: {
    defaultMessage: "Research keywords, localise drafts, and write to CMS on schedule.",
    id: "L/LTF1REU5",
    description: "Workflow description for SEO template in content ops flow mock",
  },
  flowDragHint: {
    defaultMessage: "Drag nodes to explore",
    id: "RTV8YbVJ6F",
    description: "Hint shown on the draggable workflow canvas in content ops mock",
  },
  flowTemplateCampaign: {
    defaultMessage: "Campaign",
    id: "+ek/FknpQJ",
    description: "Flow template pill for campaign",
  },
  flowTemplateSeo: {
    defaultMessage: "SEO blog",
    id: "1WYC40KrqC",
    description: "Flow template pill for SEO blog",
  },
  flowTemplateBrief: {
    defaultMessage: "Multilingual blog",
    id: "1rvztL9O9G",
    description: "Flow template pill for multilingual blog publishing",
  },
  flowNodeBrief: {
    defaultMessage: "GTM brief",
    id: "0h6ygVHD06",
    description: "Flow node label",
  },
  flowNodeLocalise: {
    defaultMessage: "Localise",
    id: "F3gTJhCvdi",
    description: "Flow node label",
  },
  flowNodeBrandQa: {
    defaultMessage: "Brand QA",
    id: "qDJ1AfMuj+",
    description: "Flow node label",
  },
  flowNodeReview: {
    defaultMessage: "Review",
    id: "vZn7h5Bl2j",
    description: "Flow node label",
  },
  flowNodeCms: {
    defaultMessage: "CMS publish",
    id: "2JUwOE/z9B",
    description: "Flow node label",
  },
  flowNodeSchedule: {
    defaultMessage: "Scheduled run",
    id: "cvuRKz2hMS",
    description: "Flow node label",
  },
  flowNodeKeywords: {
    defaultMessage: "Keyword research",
    id: "3wOLsPOjf5",
    description: "Flow node label",
  },
  flowNodeCreateContent: {
    defaultMessage: "Create content draft",
    id: "B5M7/Y7enl",
    description: "Flow node label for content creation in brief-to-publish workflow",
  },
  flowNodeDraft: {
    defaultMessage: "CMS draft",
    id: "DN35BCM/UE",
    description: "Flow node label",
  },
  flowNodeSlack: {
    defaultMessage: "Slack notify",
    id: "RicTjBrN9L",
    description: "Flow node label",
  },
  flowNodeStaging: {
    defaultMessage: "Staging",
    id: "U499tVTudb",
    description: "Flow node label",
  },
});

export type ContentOpsMockTabId = "triage" | "campaign" | "seo-blog" | "brand" | "editor";

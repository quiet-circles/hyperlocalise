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
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  File01Icon,
  LanguageSquareIcon,
  PauseIcon,
  PlayIcon,
  Rocket01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import Image from "next/image";
import { useIntl } from "react-intl";

import {
  LAVENDER_MESH_GRADIENT_SRC,
  MeshStage,
  SAGE_MESH_GRADIENT_SRC,
} from "@/components/marketing/hero-frame-mesh-stage";
import { cn } from "@/lib/primitives/cn";

import {
  ContentOpsAgentTerminal,
  type ContentOpsTerminalScene,
} from "./content-ops-agent-terminal";
import { ContentOpsBrandPanel } from "./content-ops-brand-panel";
import { ContentOpsEditorPanel } from "./content-ops-editor-panel";
import { ContentOpsFlowPanel } from "./content-ops-flow-panel";
import { ContentOpsMockAppShell } from "./content-ops-mock-app-shell";
import { ContentOpsInboxPanel } from "./content-ops-inbox-panel";
import {
  contentOpsMockStageMessages,
  type ContentOpsMockTabId,
} from "./content-ops-mock-stage.messages";

const TAB_HOLD_MS = 9000;
const TAB_ORDER: ContentOpsMockTabId[] = ["triage", "campaign", "seo-blog", "brand", "editor"];

const MESH_BY_TAB: Record<ContentOpsMockTabId, string> = {
  triage: SAGE_MESH_GRADIENT_SRC,
  campaign: LAVENDER_MESH_GRADIENT_SRC,
  "seo-blog": SAGE_MESH_GRADIENT_SRC,
  brand: LAVENDER_MESH_GRADIENT_SRC,
  editor: LAVENDER_MESH_GRADIENT_SRC,
};

type TabConfig = {
  id: ContentOpsMockTabId;
  labelKey: keyof typeof contentOpsMockStageMessages;
};

const TABS: TabConfig[] = [
  { id: "triage", labelKey: "tabTriage" },
  { id: "campaign", labelKey: "tabCampaign" },
  { id: "seo-blog", labelKey: "tabSeoBlog" },
  { id: "brand", labelKey: "tabBrand" },
  { id: "editor", labelKey: "tabEditor" },
];

export function ContentOpsMockStage({
  className,
  priority = false,
}: {
  className?: string;
  priority?: boolean;
}) {
  const intl = useIntl();
  const shouldReduceMotion = useReducedMotion() ?? false;
  const [activeTab, setActiveTab] = useState<ContentOpsMockTabId>("triage");
  const [autoplayEnabled, setAutoplayEnabled] = useState(true);
  const [triageHighlightIndex, setTriageHighlightIndex] = useState(0);

  const campaignScene: ContentOpsTerminalScene = useMemo(
    () => ({
      id: "campaign",
      automationName: intl.formatMessage(contentOpsMockStageMessages.campaignAutomationName),
      triggerIcon: <HugeiconsIcon icon={Rocket01Icon} strokeWidth={1.8} className="size-3.5" />,
      triggerLabel: intl.formatMessage(contentOpsMockStageMessages.triggerGtmBrief),
      instructions: intl.formatMessage(contentOpsMockStageMessages.campaignInstructions),
      tools: [
        {
          icon: <HugeiconsIcon icon={File01Icon} strokeWidth={1.8} className="size-3.5" />,
          label: intl.formatMessage(contentOpsMockStageMessages.toolCms),
          description: intl.formatMessage(contentOpsMockStageMessages.toolCmsDescription),
        },
        {
          icon: <HugeiconsIcon icon={LanguageSquareIcon} strokeWidth={1.8} className="size-3.5" />,
          label: intl.formatMessage(contentOpsMockStageMessages.toolTranslate),
          description: intl.formatMessage(contentOpsMockStageMessages.toolTranslateDescription),
        },
        {
          icon: (
            <Image
              src="/images/slack-logo.svg"
              alt=""
              width={14}
              height={14}
              className="size-3.5"
            />
          ),
          label: intl.formatMessage(contentOpsMockStageMessages.toolSlack),
          description: intl.formatMessage(contentOpsMockStageMessages.toolSlackGtmDescription),
        },
      ],
      steps: [
        intl.formatMessage(contentOpsMockStageMessages.stepGtm1),
        intl.formatMessage(contentOpsMockStageMessages.stepGtm2),
        intl.formatMessage(contentOpsMockStageMessages.stepGtm3),
        intl.formatMessage(contentOpsMockStageMessages.stepGtm4),
      ],
    }),
    [intl],
  );

  const handleTabSelect = useCallback((tabId: ContentOpsMockTabId) => {
    setAutoplayEnabled(false);
    setActiveTab(tabId);
    setTriageHighlightIndex(0);
  }, []);

  const handleAutoplayToggle = useCallback(() => {
    setAutoplayEnabled((enabled) => !enabled);
  }, []);

  useEffect(() => {
    if (!autoplayEnabled || shouldReduceMotion) {
      return;
    }

    const timer = window.setTimeout(() => {
      setActiveTab((current) => {
        const index = TAB_ORDER.indexOf(current);
        return TAB_ORDER[(index + 1) % TAB_ORDER.length]!;
      });
      setTriageHighlightIndex(0);
    }, TAB_HOLD_MS);

    return () => window.clearTimeout(timer);
  }, [activeTab, autoplayEnabled, shouldReduceMotion]);

  const pauseAutoplay = !autoplayEnabled || shouldReduceMotion;

  useEffect(() => {
    if (activeTab !== "triage" || pauseAutoplay) {
      return;
    }

    const timer = window.setInterval(() => {
      setTriageHighlightIndex((step) => step + 1);
    }, 2600);

    return () => window.clearInterval(timer);
  }, [activeTab, pauseAutoplay]);

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex items-center gap-2">
        <div className="-mx-1 flex min-w-0 flex-1 gap-2 overflow-x-auto pb-1">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => handleTabSelect(tab.id)}
              className={cn(
                "shrink-0 cursor-pointer rounded-full border px-4 py-2 text-left text-sm font-medium transition-all duration-200",
                activeTab === tab.id
                  ? "border-primary/35 bg-primary/8 text-foreground shadow-sm"
                  : "border-border/60 bg-background text-muted-foreground hover:border-border hover:text-foreground",
              )}
            >
              {intl.formatMessage(contentOpsMockStageMessages[tab.labelKey])}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={handleAutoplayToggle}
          className="inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-full border border-border/60 bg-background text-muted-foreground transition-colors hover:border-border hover:text-foreground"
          aria-label={intl.formatMessage(
            autoplayEnabled && !shouldReduceMotion
              ? contentOpsMockStageMessages.autoplayPause
              : contentOpsMockStageMessages.autoplayResume,
          )}
        >
          <HugeiconsIcon
            icon={autoplayEnabled && !shouldReduceMotion ? PauseIcon : PlayIcon}
            strokeWidth={2}
            className="size-3.5"
          />
        </button>
      </div>

      <MeshStage
        meshSrc={MESH_BY_TAB[activeTab]}
        priority={priority}
        layout="breakout"
        entranceAnimation="none"
      >
        <AnimatePresence mode="wait">
          <motion.div
            key={activeTab}
            initial={shouldReduceMotion ? false : { opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={shouldReduceMotion ? undefined : { opacity: 0 }}
            transition={{ duration: shouldReduceMotion ? 0 : 0.2 }}
            className="w-full"
          >
            <ContentOpsMockAppShell activeTab={activeTab}>
              {activeTab === "triage" ? (
                <ContentOpsInboxPanel highlightedIndex={triageHighlightIndex % 3} />
              ) : null}
              {activeTab === "campaign" ? <ContentOpsAgentTerminal scene={campaignScene} /> : null}
              {activeTab === "seo-blog" ? (
                <ContentOpsFlowPanel pauseAutoplay={pauseAutoplay} />
              ) : null}
              {activeTab === "brand" ? <ContentOpsBrandPanel /> : null}
              {activeTab === "editor" ? (
                <ContentOpsEditorPanel pauseAutoplay={pauseAutoplay} />
              ) : null}
            </ContentOpsMockAppShell>
          </motion.div>
        </AnimatePresence>
      </MeshStage>
    </div>
  );
}

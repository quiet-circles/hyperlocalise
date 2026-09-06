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
import { useCallback, useEffect, useEffectEvent, useRef, useState } from "react";
import {
  BookOpenTextIcon,
  Cancel01Icon,
  Chat01Icon,
  File01Icon,
  FileSearchIcon,
  RefreshIcon,
  SentIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { FormattedMessage, useIntl } from "react-intl";

import { Tool, ToolHeader } from "@/components/ai-elements/tool";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/primitives/cn";

import { CONTENT_OPS_MOCK_INNER_CLASSNAME } from "./content-ops-mock-stage.constants";
import { contentOpsMockStageMessages } from "./content-ops-mock-stage.messages";

const TOOL_RESOLVE_MS = 520;
const STEP_MS = 720;
const EASE_OUT = [0.19, 1, 0.22, 1] as const;
const COLLAPSE_GLYPH = "−";
const MENTION_GLYPH = "@";

export type BrandPlaybackPhase = "idle" | "playing" | "done";

export function ContentOpsBrandPanel({
  autoStart = true,
  onPhaseChange,
}: {
  autoStart?: boolean;
  onPhaseChange?: (phase: BrandPlaybackPhase) => void;
}) {
  const intl = useIntl();
  const shouldReduceMotion = useReducedMotion() ?? false;
  const hasAutoStartedRef = useRef(false);
  const playingRef = useRef(false);
  const [phase, setPhase] = useState<BrandPlaybackPhase>("idle");
  const [showTool, setShowTool] = useState(false);
  const [toolResolved, setToolResolved] = useState(false);
  const [showAnswer, setShowAnswer] = useState(false);
  const timersRef = useRef<ReturnType<typeof setTimeout>[]>([]);
  const transcriptRef = useRef<HTMLDivElement>(null);

  const prompt = intl.formatMessage(contentOpsMockStageMessages.brandChatPrompt);
  const guidelineSections = [
    {
      title: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineToneTitle),
      body: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineToneBody),
    },
    {
      title: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineCtaTitle),
      body: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineCtaBody),
    },
    {
      title: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineTermsTitle),
      body: intl.formatMessage(contentOpsMockStageMessages.brandGuidelineTermsBody),
    },
  ];

  const clearTimers = () => {
    for (const timer of timersRef.current) {
      clearTimeout(timer);
    }
    timersRef.current = [];
  };

  const notifyPhaseChange = useEffectEvent((nextPhase: BrandPlaybackPhase) => {
    onPhaseChange?.(nextPhase);
  });

  const resetPlayback = () => {
    clearTimers();
    playingRef.current = false;
    setPhase("idle");
    setShowTool(false);
    setToolResolved(false);
    setShowAnswer(false);
    notifyPhaseChange("idle");
  };

  const schedule = (fn: () => void, delay: number) => {
    const timer = setTimeout(fn, delay);
    timersRef.current.push(timer);
  };

  const startPlayback = useCallback(() => {
    if (playingRef.current) {
      return;
    }

    playingRef.current = true;
    clearTimers();
    setPhase("playing");
    setShowTool(false);
    setToolResolved(false);
    setShowAnswer(false);
    notifyPhaseChange("playing");

    let elapsed = shouldReduceMotion ? 0 : 180;

    schedule(() => setShowTool(true), elapsed);
    elapsed += shouldReduceMotion ? 0 : TOOL_RESOLVE_MS;
    schedule(() => setToolResolved(true), elapsed);
    elapsed += shouldReduceMotion ? 0 : STEP_MS;
    schedule(() => {
      setShowAnswer(true);
      setPhase("done");
      playingRef.current = false;
      notifyPhaseChange("done");
    }, elapsed);
  }, [shouldReduceMotion]);

  useEffect(() => {
    if (!autoStart) {
      return;
    }

    const timer = setTimeout(
      () => {
        if (hasAutoStartedRef.current) {
          return;
        }

        hasAutoStartedRef.current = true;
        startPlayback();
      },
      shouldReduceMotion ? 0 : 400,
    );

    return () => clearTimeout(timer);
  }, [autoStart, shouldReduceMotion, startPlayback]);

  useEffect(() => () => clearTimers(), []);

  useEffect(() => {
    if (!transcriptRef.current) {
      return;
    }
    transcriptRef.current.scrollTop = transcriptRef.current.scrollHeight;
  }, [showTool, toolResolved, showAnswer]);

  const isBusy = phase === "playing";

  return (
    <div className={cn(CONTENT_OPS_MOCK_INNER_CLASSNAME, "lg:grid lg:grid-cols-2")}>
      <div className="flex min-h-0 max-h-56 flex-col overflow-y-auto border-b border-border/50 lg:max-h-none lg:border-b-0 lg:border-r">
        <div className="border-b border-border/50 px-5 py-4">
          <div className="text-base font-semibold text-foreground">
            <FormattedMessage {...contentOpsMockStageMessages.brandStyleTitle} />
          </div>
          <p className="mt-1 text-pretty text-xs text-muted-foreground">
            <FormattedMessage {...contentOpsMockStageMessages.brandStyleSubtitle} />
          </p>
        </div>

        <div className="space-y-4 p-4">
          <div className="flex flex-wrap gap-2">
            {[
              contentOpsMockStageMessages.brandRuleTone,
              contentOpsMockStageMessages.brandRuleCta,
            ].map((rule) => (
              <span
                key={rule.id}
                className="rounded-full border border-primary/40 bg-primary/10 px-3 py-1 text-[11px] font-medium text-foreground"
              >
                <FormattedMessage {...rule} />
              </span>
            ))}
          </div>

          <div className="space-y-4 rounded-xl border border-border/60 bg-muted/20 p-4">
            {guidelineSections.map((section) => (
              <div key={section.title} className="space-y-1">
                <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  {section.title}
                </h4>
                <p className="text-pretty text-sm leading-relaxed text-foreground">
                  {section.body}
                </p>
              </div>
            ))}
          </div>

          <div className="overflow-hidden rounded-xl border border-border/60 bg-background">
            <div className="flex items-center gap-2 border-b border-border/50 px-3 py-2.5">
              <HugeiconsIcon
                icon={File01Icon}
                strokeWidth={1.8}
                className="size-4 shrink-0 text-primary"
              />
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium text-foreground">
                  <FormattedMessage {...contentOpsMockStageMessages.brandUploadedGuideFilename} />
                </p>
                <p className="text-[10px] text-muted-foreground">
                  <FormattedMessage {...contentOpsMockStageMessages.brandUploadedGuideSize} />
                </p>
              </div>
              <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                PDF
              </span>
            </div>

            <div className="space-y-2 bg-muted/25 p-4">
              <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
                <FormattedMessage {...contentOpsMockStageMessages.brandUploadedGuideLabel} />
              </p>
              <div className="space-y-2 rounded-lg border border-border/50 bg-background p-3 shadow-sm">
                <div className="h-2 w-2/3 rounded bg-muted" />
                <div className="h-2 w-full rounded bg-muted" />
                <div className="h-2 w-5/6 rounded bg-muted" />
                <p className="pt-1 text-pretty text-xs leading-relaxed text-muted-foreground">
                  <FormattedMessage {...contentOpsMockStageMessages.brandUploadedGuideExcerpt} />
                </p>
                <div className="h-2 w-4/5 rounded bg-muted" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="flex min-h-96 min-w-0 flex-1 flex-col overflow-hidden p-3 lg:min-h-0 lg:p-4">
        <div
          className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-border bg-background shadow-2xl shadow-black/15"
          role="region"
          aria-label={intl.formatMessage(contentOpsMockStageMessages.brandChatTitle)}
        >
          <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3">
            <p className="min-w-0 flex-1 truncate text-sm font-medium text-foreground">
              <FormattedMessage {...contentOpsMockStageMessages.brandChatTitle} />
            </p>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              tabIndex={-1}
              aria-label={intl.formatMessage(contentOpsMockStageMessages.brandCollapseLabel)}
            >
              <span aria-hidden className="text-base leading-none">
                {COLLAPSE_GLYPH}
              </span>
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              tabIndex={-1}
              aria-label={intl.formatMessage(contentOpsMockStageMessages.brandCloseLabel)}
            >
              <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} className="size-3.5" />
            </Button>
          </header>

          <div ref={transcriptRef} className="flex min-h-0 flex-1 flex-col overflow-y-auto">
            {phase === "idle" ? (
              <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-5 px-5 py-8 text-center">
                <div className="flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                  <HugeiconsIcon icon={Chat01Icon} strokeWidth={1.8} className="size-5" />
                </div>
                <div className="max-w-sm space-y-1">
                  <h3 className="text-balance text-sm font-semibold text-foreground">
                    <FormattedMessage {...contentOpsMockStageMessages.brandChatEmptyTitle} />
                  </h3>
                  <p className="text-pretty text-sm text-muted-foreground">
                    <FormattedMessage {...contentOpsMockStageMessages.brandChatEmptySubtitle} />
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-8 gap-1.5 rounded-full bg-background text-xs font-medium"
                  onClick={startPlayback}
                >
                  <HugeiconsIcon icon={FileSearchIcon} strokeWidth={1.8} className="size-3.5" />
                  <FormattedMessage {...contentOpsMockStageMessages.brandSuggestionCta} />
                </Button>
              </div>
            ) : (
              <div className="flex flex-col gap-4 px-4 py-5">
                <div className="ms-auto max-w-[90%] rounded-2xl bg-muted px-3.5 py-2.5 text-pretty text-sm leading-6 text-foreground">
                  {prompt}
                </div>

                <div className="space-y-3">
                  <AnimatePresence initial={false}>
                    {showTool ? (
                      <motion.div
                        key="brand-tool"
                        initial={shouldReduceMotion ? false : { opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{
                          duration: shouldReduceMotion ? 0 : 0.28,
                          ease: EASE_OUT,
                        }}
                        className="rounded-lg border border-border/70 bg-muted/20 px-3 py-2"
                      >
                        <Tool defaultOpen={toolResolved}>
                          <ToolHeader
                            type="dynamic-tool"
                            toolName={intl.formatMessage(contentOpsMockStageMessages.brandToolName)}
                            state={toolResolved ? "output-available" : "input-available"}
                            detail={intl.formatMessage(contentOpsMockStageMessages.brandToolDetail)}
                            input={{
                              query: "brand voice CTA DE",
                            }}
                          />
                        </Tool>
                      </motion.div>
                    ) : null}
                  </AnimatePresence>

                  {showAnswer ? (
                    <motion.div
                      initial={shouldReduceMotion ? false : { opacity: 0, y: 12 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: shouldReduceMotion ? 0 : 0.35, ease: EASE_OUT }}
                      className="space-y-4 text-sm leading-6 text-foreground"
                    >
                      <div className="space-y-1">
                        <p className="font-medium text-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandVerdictLabel} />
                        </p>
                        <p className="text-pretty text-muted-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandVerdictBody} />
                        </p>
                      </div>
                      <div className="space-y-1">
                        <p className="font-medium text-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandGuidelineLabel} />
                        </p>
                        <p className="text-pretty text-muted-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandGuidelineBody} />
                        </p>
                      </div>
                      <div className="space-y-1">
                        <p className="font-medium text-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandSuggestLabel} />
                        </p>
                        <p className="text-pretty text-muted-foreground">
                          <FormattedMessage {...contentOpsMockStageMessages.brandSuggestBody} />
                        </p>
                      </div>
                    </motion.div>
                  ) : null}
                </div>
              </div>
            )}
          </div>

          <form
            className="shrink-0 border-t border-border bg-background p-3"
            onSubmit={(event) => {
              event.preventDefault();
              if (phase === "done") {
                resetPlayback();
              }

              startPlayback();
            }}
          >
            <div className="overflow-hidden rounded-xl border border-border bg-muted/30 shadow-sm">
              <div className="flex flex-wrap gap-1.5 px-3 pt-3">
                <span className="inline-flex items-center gap-1 rounded-full border border-border bg-background px-2 py-0.5 text-[0.7rem] text-muted-foreground">
                  {MENTION_GLYPH}
                </span>
                <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2 py-0.5 text-[0.7rem] text-foreground">
                  <HugeiconsIcon icon={BookOpenTextIcon} strokeWidth={1.8} className="size-3" />
                  <FormattedMessage {...contentOpsMockStageMessages.brandContextPill} />
                </span>
              </div>
              <div className="flex items-end gap-2 px-3 py-3">
                <p className="min-w-0 flex-1 text-pretty text-sm leading-5 text-foreground">
                  {phase === "idle" ? (
                    prompt
                  ) : (
                    <span className="text-muted-foreground">
                      <FormattedMessage {...contentOpsMockStageMessages.brandComposerPlaceholder} />
                    </span>
                  )}
                </p>
                {phase === "done" ? (
                  <Button
                    type="submit"
                    size="sm"
                    variant="secondary"
                    className="h-8 rounded-full px-3"
                  >
                    <HugeiconsIcon
                      data-icon="inline-start"
                      icon={RefreshIcon}
                      strokeWidth={2}
                      className="size-3.5"
                    />
                    <FormattedMessage {...contentOpsMockStageMessages.brandReplay} />
                  </Button>
                ) : (
                  <Button
                    type="submit"
                    size="sm"
                    className="h-8 rounded-full px-3"
                    disabled={isBusy}
                    aria-label={intl.formatMessage(contentOpsMockStageMessages.brandSend)}
                  >
                    <HugeiconsIcon
                      data-icon="inline-start"
                      icon={SentIcon}
                      strokeWidth={2}
                      className="size-3.5"
                    />
                    <FormattedMessage {...contentOpsMockStageMessages.brandSend} />
                  </Button>
                )}
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

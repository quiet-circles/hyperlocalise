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
import { useEffect, useState, useSyncExternalStore } from "react";
import {
  BookOpenTextIcon,
  CheckmarkCircle02Icon,
  Copy01Icon,
  CustomerSupportIcon,
  MinusSignCircleIcon,
  TextFontIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { FormattedMessage, useIntl } from "react-intl";

import type { InboxCurrentUser } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/inbox/_components/inbox-types";
import {
  ChatDockBridge,
  ChatDockFooterControls,
  ChatDockPanelHost,
} from "@/components/app-shell/chat-dock/chat-dock";
import { ChatDockErrorBoundary } from "@/components/app-shell/chat-dock/chat-dock-error-boundary";
import { PlanUsageFooterControl } from "@/components/billing/plan-usage-summary";
import {
  getCatGlossaryGuidanceServerSnapshot,
  getCatGlossaryGuidanceStatus,
  requestCatGlossaryGuidance,
  subscribeCatGlossaryGuidance,
} from "@/components/content-editor/intelligence/content-editor-glossary-guidance-event";
import {
  getCatIssueGuidanceServerSnapshot,
  getCatIssueGuidanceStatus,
  requestCatIssueGuidance,
  subscribeCatIssueGuidance,
} from "@/components/content-editor/issues/content-editor-issue-guidance-event";
import { ContentEditorStyleGuideSheet } from "@/components/content-editor/style-guide/content-editor-style-guide-sheet";
import { Button } from "@/components/ui/button";
import { Box } from "@/components/ui/layout/box";
import { Column } from "@/components/ui/layout/column";
import { Columns } from "@/components/ui/layout/columns";
import { Row } from "@/components/ui/layout/row";
import { SUPPORT_EMAIL } from "@/lib/support-contact";

import { appShellFooterMessages } from "./app-shell-footer.messages";

export function AppShellFooter({
  organizationSlug,
  projectId = null,
  showPlan,
  showGlossaryGuidance = false,
  showIssueGuidance = false,
  showStyleGuide = false,
  currentUser,
}: {
  organizationSlug: string;
  projectId?: string | null;
  showPlan: boolean;
  showGlossaryGuidance?: boolean;
  showIssueGuidance?: boolean;
  showStyleGuide?: boolean;
  currentUser?: InboxCurrentUser;
}) {
  const intl = useIntl();
  const showChatDock = Boolean(organizationSlug && currentUser);
  const [styleGuideOpen, setStyleGuideOpen] = useState(false);
  const canShowStyleGuide = showStyleGuide && Boolean(projectId);

  useEffect(() => {
    if (!canShowStyleGuide) {
      setStyleGuideOpen(false);
    }
  }, [canShowStyleGuide]);
  const glossaryGuidanceStatus = useSyncExternalStore(
    subscribeCatGlossaryGuidance,
    getCatGlossaryGuidanceStatus,
    getCatGlossaryGuidanceServerSnapshot,
  );
  const issueGuidanceStatus = useSyncExternalStore(
    subscribeCatIssueGuidance,
    getCatIssueGuidanceStatus,
    getCatIssueGuidanceServerSnapshot,
  );

  return (
    <footer
      className="fixed inset-x-0 bottom-0 z-40 flex flex-col border-t border-border bg-background"
      style={{ viewTransitionName: "app-shell-footer" }}
    >
      {showChatDock ? <ChatDockBridge organizationSlug={organizationSlug} /> : null}
      {showChatDock && currentUser ? (
        <ChatDockErrorBoundary organizationSlug={organizationSlug}>
          <ChatDockPanelHost organizationSlug={organizationSlug} currentUser={currentUser} />
        </ChatDockErrorBoundary>
      ) : null}

      <div className="h-(--app-shell-plan-footer-height) shrink-0">
        <Box paddingX="1u" display="flex" alignItems="center" height="full" width="full">
          <Columns spacing="1u" alignY="center" align={showPlan ? "spaceBetween" : "end"}>
            {showPlan ? (
              <Column width="content">
                <PlanUsageFooterControl organizationSlug={organizationSlug} />
              </Column>
            ) : null}
            <Column width="content">
              <Row spacing="1u" alignY="center">
                {canShowStyleGuide && projectId ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => setStyleGuideOpen(true)}
                    aria-label={intl.formatMessage(appShellFooterMessages.styleGuideAriaLabel)}
                  >
                    <HugeiconsIcon icon={TextFontIcon} strokeWidth={2} data-icon="inline-start" />
                    <FormattedMessage {...appShellFooterMessages.styleGuideLabel} />
                  </Button>
                ) : null}
                {showGlossaryGuidance ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={requestCatGlossaryGuidance}
                    aria-label={intl.formatMessage(
                      glossaryGuidanceStatus.matchCount > 0 ||
                        glossaryGuidanceStatus.preferredCount > 0 ||
                        glossaryGuidanceStatus.notRecommendedCount > 0
                        ? appShellFooterMessages.glossaryGuidanceAvailableAriaLabel
                        : appShellFooterMessages.glossaryGuidanceAriaLabel,
                    )}
                  >
                    <HugeiconsIcon
                      icon={BookOpenTextIcon}
                      strokeWidth={2}
                      data-icon="inline-start"
                    />
                    <FormattedMessage {...appShellFooterMessages.glossaryGuidanceLabel} />
                    {glossaryGuidanceStatus.preferredCount > 0 ? (
                      <span className="inline-flex items-center gap-0.5 text-xs font-medium text-emerald-500">
                        <HugeiconsIcon
                          icon={CheckmarkCircle02Icon}
                          strokeWidth={2}
                          className="size-4"
                          aria-hidden="true"
                        />
                        <span className="tabular-nums">
                          {glossaryGuidanceStatus.preferredCount}
                        </span>
                      </span>
                    ) : null}
                    {glossaryGuidanceStatus.notRecommendedCount > 0 ? (
                      <span className="inline-flex items-center gap-0.5 text-xs font-medium text-rose-500">
                        <HugeiconsIcon
                          icon={MinusSignCircleIcon}
                          strokeWidth={2}
                          className="size-4"
                          aria-hidden="true"
                        />
                        <span className="tabular-nums">
                          {glossaryGuidanceStatus.notRecommendedCount}
                        </span>
                      </span>
                    ) : null}
                  </Button>
                ) : null}
                {showIssueGuidance && issueGuidanceStatus.available ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={requestCatIssueGuidance}
                    aria-label={intl.formatMessage(
                      issueGuidanceStatus.openIssueCount > 0
                        ? appShellFooterMessages.issueGuidanceAvailableAriaLabel
                        : appShellFooterMessages.issueGuidanceAriaLabel,
                      issueGuidanceStatus.openIssueCount > 0
                        ? { count: issueGuidanceStatus.openIssueCount }
                        : undefined,
                    )}
                  >
                    <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} data-icon="inline-start" />
                    <FormattedMessage {...appShellFooterMessages.issueGuidanceLabel} />
                    {issueGuidanceStatus.openIssueCount > 0 ? (
                      <span className="tabular-nums text-xs font-medium text-flame-900 dark:text-flame-100">
                        {issueGuidanceStatus.openIssueCount}
                      </span>
                    ) : null}
                  </Button>
                ) : null}
                {showChatDock ? (
                  <ChatDockFooterControls organizationSlug={organizationSlug} />
                ) : null}
                <Button
                  variant="ghost"
                  render={<a href={`mailto:${SUPPORT_EMAIL}`} />}
                  aria-label={intl.formatMessage(appShellFooterMessages.emailSupportAriaLabel)}
                >
                  <HugeiconsIcon
                    icon={CustomerSupportIcon}
                    strokeWidth={2}
                    data-icon="inline-start"
                  />
                  <FormattedMessage {...appShellFooterMessages.supportLabel} />
                </Button>
              </Row>
            </Column>
          </Columns>
        </Box>
      </div>
      {canShowStyleGuide && projectId ? (
        <ContentEditorStyleGuideSheet
          organizationSlug={organizationSlug}
          projectId={projectId}
          open={styleGuideOpen}
          onOpenChange={setStyleGuideOpen}
        />
      ) : null}
    </footer>
  );
}

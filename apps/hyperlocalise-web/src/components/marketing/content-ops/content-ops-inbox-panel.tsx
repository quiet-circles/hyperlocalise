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
import {
  BubbleChatNotificationIcon,
  Calendar03Icon,
  LanguageCircleIcon,
  Tag01Icon,
  User02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { FormattedMessage, useIntl } from "react-intl";

import { IssueStatusIcon } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/_components/issue-detail/issue-status-icon";
import { conversationPanelMessages } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/inbox/_components/conversation-panel.messages";
import {
  getConversationListItemVisual,
  getNotificationListItemVisual,
  type InboxListItemVisual,
} from "@/app/[lang]/(authenticated)/org/[organizationSlug]/inbox/_components/inbox-list-item-visuals";
import {
  formatRelativeTime,
  getSourceLabel,
  getStatusLabel,
  statusStyles,
  type Conversation,
} from "@/app/[lang]/(authenticated)/org/[organizationSlug]/inbox/_components/inbox-types";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Box } from "@/components/ui/layout/box";
import { Column } from "@/components/ui/layout/column";
import { Columns } from "@/components/ui/layout/columns";
import { Row } from "@/components/ui/layout/row";
import { Rows } from "@/components/ui/layout/rows";
import { Message, MessageAvatar, MessageContent, MessageFooter } from "@/components/ui/message";
import {
  TypographyH4,
  TypographyMuted,
  TypographyP,
  TypographySmall,
} from "@/components/ui/typography";
import { cn } from "@/lib/primitives/cn";

import { CONTENT_OPS_MOCK_INNER_CLASSNAME } from "./content-ops-mock-stage.constants";
import { contentOpsMockStageMessages } from "./content-ops-mock-stage.messages";

const TWO_HOURS_MS = 2 * 60 * 60 * 1000;
const FORTY_FIVE_MINUTES_MS = 45 * 60 * 1000;
const YESTERDAY_MS = 26 * 60 * 60 * 1000;

type MockConversationItem = {
  kind: "conversation";
  id: string;
  title: string;
  preview: string;
  avatarLabel: string;
  source: Conversation["source"];
  createdAt: string;
  userMessage: string;
  assistantMessage: string;
};

type MockIssueItem = {
  kind: "issue";
  id: string;
  title: string;
  preview: string;
  avatarLabel: string;
  createdAt: string;
  unread: true;
};

type MockInboxItem = MockConversationItem | MockIssueItem;

function isoAgo(ms: number) {
  return new Date(Date.now() - ms).toISOString();
}

function listItemClassName(isSelected: boolean, isUnread = false) {
  return cn(
    "grid w-full grid-cols-[auto_minmax(0,1fr)] gap-3 rounded-md px-2 py-2.5 text-left transition-colors",
    isSelected
      ? "bg-accent text-foreground"
      : "text-foreground hover:bg-muted hover:text-foreground",
    isUnread && !isSelected && "bg-muted/40",
  );
}

function InboxListItemAvatar({
  visual,
  children,
}: {
  visual: InboxListItemVisual;
  children: ReactNode;
}) {
  return (
    <div className="relative shrink-0">
      <Avatar className="bg-muted">{children}</Avatar>
      <span
        className="absolute -end-0.5 -bottom-0.5 z-10 flex size-[18px] items-center justify-center rounded-full bg-card shadow-sm ring-2 ring-background"
        aria-label={visual.typeIconLabel}
      >
        <HugeiconsIcon
          icon={visual.typeIcon}
          strokeWidth={2}
          size={12}
          className={cn("shrink-0", visual.badgeClassName)}
        />
      </span>
    </div>
  );
}

function InboxListItemContent({
  title,
  subtitle,
  timestamp,
  titleWeight = "regular",
}: {
  title: ReactNode;
  subtitle: ReactNode;
  timestamp: string;
  titleWeight?: "regular" | "bold";
}) {
  return (
    <div className="min-w-0">
      <div className="flex items-start justify-between gap-2">
        <TypographySmall lineClamp={1} weight={titleWeight === "bold" ? "bold" : undefined}>
          {title}
        </TypographySmall>
        <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground">
          {timestamp}
        </span>
      </div>
      <TypographyMuted className="mt-0.5" lineClamp={1}>
        {subtitle}
      </TypographyMuted>
    </div>
  );
}

function MockInboxListItem({
  item,
  selected,
  timestamp,
  visual,
}: {
  item: MockInboxItem;
  selected: boolean;
  timestamp: string;
  visual: InboxListItemVisual;
}) {
  const unread = item.kind === "issue" && item.unread;

  return (
    <div className={listItemClassName(selected, unread)}>
      <InboxListItemAvatar visual={visual}>
        <AvatarFallback className="bg-muted text-xs font-medium text-foreground">
          {item.avatarLabel}
        </AvatarFallback>
      </InboxListItemAvatar>
      <InboxListItemContent
        title={item.title}
        subtitle={item.preview}
        timestamp={timestamp}
        titleWeight={unread ? "bold" : "regular"}
      />
    </div>
  );
}

function ConversationThreadMessage({
  role,
  avatarLabel,
  createdAt,
  children,
}: {
  role: "user" | "assistant";
  avatarLabel?: string;
  createdAt: string;
  children: ReactNode;
}) {
  const intl = useIntl();
  const timestamp = formatRelativeTime(createdAt, intl);

  if (role === "assistant") {
    return (
      <Message className="w-full max-w-full">
        <MessageContent className="w-full max-w-full leading-6">
          {children}
          <MessageFooter className="px-0">
            <TypographyMuted size="xsmall">{timestamp}</TypographyMuted>
          </MessageFooter>
        </MessageContent>
      </Message>
    );
  }

  return (
    <Message align="end" className="max-w-[85%]">
      <MessageAvatar className="size-8 self-start">
        <Avatar className="size-8 shrink-0 bg-muted">
          <AvatarFallback className="bg-muted text-[10px] font-medium text-foreground">
            {avatarLabel}
          </AvatarFallback>
        </Avatar>
      </MessageAvatar>
      <MessageContent className="leading-6">
        <div className="w-fit max-w-full rounded-lg bg-muted px-4 py-3 text-foreground">
          {children}
        </div>
        <MessageFooter className="px-0">
          <TypographyMuted size="xsmall">{timestamp}</TypographyMuted>
        </MessageFooter>
      </MessageContent>
    </Message>
  );
}

function ConversationDetailMock({ item }: { item: MockConversationItem }) {
  const intl = useIntl();

  return (
    <section className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <header className="border-b border-border">
        <Box paddingX="3u" paddingY="1.5u" display="flex" alignItems="center">
          <Row spacing="1.5u" alignY="start">
            <HugeiconsIcon
              icon={BubbleChatNotificationIcon}
              strokeWidth={1.8}
              className="mt-0.5 size-5 shrink-0 text-muted-foreground"
            />
            <Rows spacing="1u">
              <TypographyH4 lineClamp={1} size="medium">
                {item.title}
              </TypographyH4>
              <Box display="flex" flexWrap="wrap" alignItems="center" gap="1u">
                <Badge variant="outline" className="border-border bg-muted text-foreground">
                  {getSourceLabel(item.source, intl)}
                </Badge>
                <Badge variant="outline" className={statusStyles.active}>
                  {getStatusLabel("active", intl)}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  <FormattedMessage
                    {...conversationPanelMessages.createdAt}
                    values={{ relativeTime: formatRelativeTime(item.createdAt, intl) }}
                  />
                </span>
              </Box>
            </Rows>
          </Row>
        </Box>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-4 px-4 py-5 sm:px-6">
          <ConversationThreadMessage
            role="user"
            avatarLabel={item.avatarLabel}
            createdAt={item.createdAt}
          >
            <TypographyP className="whitespace-pre-wrap leading-6">{item.userMessage}</TypographyP>
          </ConversationThreadMessage>
          <ConversationThreadMessage role="assistant" createdAt={item.createdAt}>
            <TypographyP className="whitespace-pre-wrap leading-6">
              {item.assistantMessage}
            </TypographyP>
          </ConversationThreadMessage>
        </div>
      </div>
    </section>
  );
}

function IssueDetailMock({
  title,
  description,
  commentAuthor,
  commentBody,
  createdAt,
}: {
  title: string;
  description: string;
  commentAuthor: string;
  commentBody: string;
  createdAt: string;
}) {
  const intl = useIntl();
  const timestamp = formatRelativeTime(createdAt, intl);

  return (
    <section className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="grid h-full min-h-0 flex-1 grid-rows-[minmax(0,1fr)] overflow-hidden lg:grid-cols-[minmax(0,1fr)_3rem]">
        <div className="flex min-h-0 min-w-0 flex-col gap-3 overflow-y-auto px-6 py-5">
          <span className="font-mono text-xs text-muted-foreground tabular-nums">WEB-2</span>
          <p className="text-lg font-semibold leading-snug text-foreground md:text-xl">{title}</p>
          <TypographyP className="text-sm leading-relaxed text-foreground">
            {description}
          </TypographyP>

          <section className="mt-2 grid gap-3 border-t border-border pt-4">
            <div className="flex items-start gap-2">
              <Avatar size="sm" className="mt-0.5 size-6">
                <AvatarFallback className="text-[10px]">MC</AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                  <span className="text-sm font-medium text-foreground">{commentAuthor}</span>
                  <span className="text-xs text-muted-foreground">{timestamp}</span>
                </div>
                <TypographyP className="mt-0.5 text-sm">{commentBody}</TypographyP>
              </div>
            </div>
          </section>
        </div>

        <div className="hidden min-h-0 flex-col items-center gap-1 border-s border-border py-3 lg:flex">
          <span className="flex size-8 items-center justify-center text-muted-foreground">
            <IssueStatusIcon status="in_progress" className="size-3.5" />
          </span>
          <span className="flex size-8 items-center justify-center text-muted-foreground">
            <HugeiconsIcon icon={User02Icon} strokeWidth={1.8} className="size-3.5" />
          </span>
          <span className="flex size-8 items-center justify-center text-muted-foreground">
            <HugeiconsIcon icon={Tag01Icon} strokeWidth={1.8} className="size-3.5" />
          </span>
          <span className="flex size-8 items-center justify-center text-muted-foreground">
            <HugeiconsIcon icon={LanguageCircleIcon} strokeWidth={1.8} className="size-3.5" />
          </span>
          <span className="flex size-8 items-center justify-center text-muted-foreground">
            <HugeiconsIcon icon={Calendar03Icon} strokeWidth={1.8} className="size-3.5" />
          </span>
        </div>
      </div>
    </section>
  );
}

export function ContentOpsInboxPanel({ highlightedIndex = 0 }: { highlightedIndex?: number }) {
  const intl = useIntl();
  const twoHoursAgo = isoAgo(TWO_HOURS_MS);
  const fortyFiveMinutesAgo = isoAgo(FORTY_FIVE_MINUTES_MS);
  const yesterday = isoAgo(YESTERDAY_MS);

  const items: MockInboxItem[] = [
    {
      kind: "conversation",
      id: "conv-de-cta",
      title: intl.formatMessage(contentOpsMockStageMessages.inboxConvDeCtaTitle),
      preview: intl.formatMessage(contentOpsMockStageMessages.inboxConvDeCtaPreview),
      avatarLabel: "MC",
      source: "slack_agent",
      createdAt: twoHoursAgo,
      userMessage: intl.formatMessage(contentOpsMockStageMessages.inboxAssistantDeCtaQuestion),
      assistantMessage: intl.formatMessage(contentOpsMockStageMessages.inboxAssistantDeCtaAnswer),
    },
    {
      kind: "issue",
      id: "issue-web-2",
      title: intl.formatMessage(contentOpsMockStageMessages.issueWeb2Title),
      preview: intl.formatMessage(contentOpsMockStageMessages.inboxIssueWeb2Preview),
      avatarLabel: "M",
      createdAt: fortyFiveMinutesAgo,
      unread: true,
    },
    {
      kind: "conversation",
      id: "conv-glossary",
      title: intl.formatMessage(contentOpsMockStageMessages.inboxConvGlossaryTitle),
      preview: intl.formatMessage(contentOpsMockStageMessages.inboxConvGlossaryPreview),
      avatarLabel: "AK",
      source: "email_agent",
      createdAt: yesterday,
      userMessage: intl.formatMessage(contentOpsMockStageMessages.inboxAssistantGlossaryQuestion),
      assistantMessage: intl.formatMessage(
        contentOpsMockStageMessages.inboxAssistantGlossaryAnswer,
      ),
    },
  ];

  const selectedIndex = highlightedIndex % items.length;
  const selectedItem = items[selectedIndex]!;

  return (
    <div className={cn(CONTENT_OPS_MOCK_INNER_CLASSNAME, "min-h-0")}>
      <Columns spacing="0" height="full" collapseBelow="large">
        <Column width="1/4">
          <section className="flex max-h-[40%] min-h-0 shrink-0 flex-col overflow-hidden border-border lg:h-full lg:max-h-none lg:shrink lg:border-r">
            <div className="min-h-0 flex-1 overflow-y-auto p-2">
              <div className="flex flex-col gap-1">
                {items.map((item, index) => {
                  const visual =
                    item.kind === "issue"
                      ? getNotificationListItemVisual("assigned", intl)
                      : getConversationListItemVisual(item.source, intl);

                  return (
                    <MockInboxListItem
                      key={item.id}
                      item={item}
                      selected={index === selectedIndex}
                      timestamp={formatRelativeTime(item.createdAt, intl)}
                      visual={visual}
                    />
                  );
                })}
              </div>
            </div>
          </section>
        </Column>

        <Column width="fluid">
          {selectedItem.kind === "issue" ? (
            <IssueDetailMock
              title={selectedItem.title}
              description={intl.formatMessage(
                contentOpsMockStageMessages.inboxIssueWeb2Description,
              )}
              commentAuthor={intl.formatMessage(
                contentOpsMockStageMessages.inboxIssueCommentAuthor,
              )}
              commentBody={intl.formatMessage(contentOpsMockStageMessages.inboxIssueComment)}
              createdAt={selectedItem.createdAt}
            />
          ) : (
            <ConversationDetailMock item={selectedItem} />
          )}
        </Column>
      </Columns>
    </div>
  );
}

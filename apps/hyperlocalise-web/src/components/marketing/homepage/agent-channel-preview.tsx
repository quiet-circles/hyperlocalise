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
import Image from "next/image";
import { FormattedMessage, useIntl } from "react-intl";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/primitives/cn";
import { homepageMessages as m } from "./homepage.messages";

const CHANNELS = ["slack", "teams", "github"] as const;
type ChannelId = (typeof CHANNELS)[number];

const CHANNEL_META: Record<ChannelId, { name: string; src: string; invertInChrome?: boolean }> = {
  slack: { name: "Slack", src: "/images/slack-logo.svg" },
  teams: { name: "Teams", src: "/images/microsoft-teams-logo.svg" },
  github: { name: "GitHub", src: "/images/github-logo.svg", invertInChrome: true },
};

function Conversation({ channel }: { channel: ChannelId }) {
  const isGitHub = channel === "github";
  return (
    <div className="flex flex-col gap-6 p-5 sm:p-6">
      <div className="flex gap-3">
        <span aria-hidden className="relative size-9 shrink-0 overflow-hidden rounded-lg">
          <Image
            src="/images/profile/michael.png"
            alt=""
            fill
            sizes="36px"
            className="object-cover"
          />
        </span>
        <div className="min-w-0">
          <p className="mb-1.5 text-xs font-semibold">Jamie</p>
          <p className="text-sm leading-6 text-pretty">
            <FormattedMessage {...m.agentRequest} />
          </p>
        </div>
      </div>
      <div className="flex gap-3">
        <span
          aria-hidden
          className="relative size-9 shrink-0 overflow-hidden rounded-lg bg-[#123c3b]"
        >
          <Image src="/images/logo.png" alt="" fill sizes="36px" className="object-cover" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="mb-1.5 text-xs font-semibold">Hyperlocalise</p>
          <p className="text-sm leading-6 text-pretty">
            <FormattedMessage {...m.agentReply} />
          </p>
          <div
            className={cn(
              "my-4 rounded-lg border p-4",
              isGitHub ? "border-[#30363d] bg-[#161b22]" : "border-border bg-muted/40",
            )}
          >
            <p className="mb-2 text-xs font-semibold">
              <FormattedMessage {...m.agentFinding} />
            </p>
            <p
              className={cn(
                "text-xs leading-6 text-pretty",
                isGitHub ? "text-[#8b949e]" : "text-muted-foreground",
              )}
            >
              <FormattedMessage {...m.agentFindingBody} />
            </p>
          </div>
          <Badge variant="secondary">
            <FormattedMessage {...m.agentResult} />
          </Badge>
        </div>
      </div>
    </div>
  );
}

function SlackChrome() {
  return (
    <div className="light flex min-h-[28rem] overflow-hidden rounded-xl border border-[#3f0e40]/40 bg-white text-[#1d1c1d] shadow-lg">
      <aside className="hidden w-[11.5rem] shrink-0 flex-col bg-[#3f0e40] text-white sm:flex">
        <div className="border-b border-white/10 px-4 py-4">
          <p className="text-sm font-bold">Acme</p>
          <p className="mt-1 text-[0.65rem] text-white/55">Workspace</p>
        </div>
        <div className="flex flex-col gap-1 px-2 py-3 text-[0.8rem]">
          <span className="rounded-md bg-white/15 px-2 py-1.5 font-medium"># content-ops</span>
          <span className="px-2 py-1.5 text-white/55"># launches</span>
          <span className="px-2 py-1.5 text-white/55"># reviews</span>
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-3 border-b border-[#e8e8e8] px-5 py-3.5">
          <Image src="/images/slack-logo.svg" alt="" width={18} height={18} />
          <span className="text-sm font-bold"># content-ops</span>
        </div>
        <Conversation channel="slack" />
      </div>
    </div>
  );
}

function TeamsChrome() {
  return (
    <div className="light flex min-h-[28rem] overflow-hidden rounded-xl border border-[#5b5fc7]/25 bg-[#f5f5f5] text-[#242424] shadow-lg">
      <aside className="hidden w-16 shrink-0 flex-col items-center gap-4 bg-[#5b5fc7] py-4 text-white sm:flex">
        <Image src="/images/microsoft-teams-logo.svg" alt="" width={22} height={22} />
        <span aria-hidden className="size-8 rounded-md bg-white/20" />
        <span aria-hidden className="size-8 rounded-md bg-white/10" />
      </aside>
      <aside className="hidden w-[11.5rem] shrink-0 flex-col bg-[#ebebeb] sm:flex">
        <div className="border-b border-black/5 px-4 py-4">
          <p className="text-sm font-bold">Content ops</p>
          <p className="mt-1 text-[0.65rem] text-[#616161]">Team</p>
        </div>
        <div className="flex flex-col gap-1 px-2 py-3 text-[0.8rem]">
          <span className="rounded-md bg-white px-2 py-1.5 font-medium shadow-sm">General</span>
          <span className="px-2 py-1.5 text-[#616161]">Launches</span>
          <span className="px-2 py-1.5 text-[#616161]">Reviews</span>
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col bg-white">
        <div className="flex items-center gap-3 border-b border-[#e0e0e0] px-5 py-3.5">
          <span className="text-sm font-bold">General</span>
        </div>
        <Conversation channel="teams" />
      </div>
    </div>
  );
}

function GitHubChrome() {
  return (
    <div className="flex min-h-[28rem] flex-col overflow-hidden rounded-xl border border-[#30363d] bg-[#0d1117] text-[#e6edf3] shadow-lg">
      <div className="flex items-center gap-3 border-b border-[#30363d] bg-[#010409] px-5 py-3.5">
        <Image src="/images/github-logo.svg" alt="" width={18} height={18} className="invert" />
        <span className="truncate text-sm text-[#8b949e]">acme / launch-copy</span>
      </div>
      <div className="border-b border-[#30363d] px-5 py-4">
        <p className="text-sm font-semibold">
          <FormattedMessage {...m.githubPrTitle} />
        </p>
        <p className="mt-2 text-xs text-[#3fb950]">
          <FormattedMessage {...m.githubPrMeta} />
        </p>
      </div>
      <Conversation channel="github" />
    </div>
  );
}

export function AgentChannelPreview() {
  const intl = useIntl();
  return (
    <Tabs defaultValue="slack" className="gap-5">
      <TabsList
        aria-label={intl.formatMessage(m.agentChannelSwitchAria)}
        className="h-auto min-h-11"
      >
        {CHANNELS.map((channel) => (
          <TabsTrigger key={channel} value={channel} className="min-h-10 gap-2 px-4">
            <Image
              src={CHANNEL_META[channel].src}
              alt=""
              width={16}
              height={16}
              className={cn(CHANNEL_META[channel].invertInChrome && "dark:invert")}
            />
            {CHANNEL_META[channel].name}
          </TabsTrigger>
        ))}
      </TabsList>
      <TabsContent value="slack">
        <SlackChrome />
      </TabsContent>
      <TabsContent value="teams">
        <TeamsChrome />
      </TabsContent>
      <TabsContent value="github">
        <GitHubChrome />
      </TabsContent>
    </Tabs>
  );
}

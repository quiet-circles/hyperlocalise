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
import { useEffect, type CSSProperties, type ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expect, fn, userEvent, within } from "storybook/test";

import { AppShellStoreProvider } from "@/components/app-shell/store/app-shell-store-context";
import {
  APP_SHELL_STORY_ORGANIZATION_SLUG,
  appShellStoryUser,
} from "@/components/app-shell/app-shell.stories.fixture";
import {
  CAT_ISSUE_GUIDANCE_OPEN_EVENT,
  EMPTY_CAT_ISSUE_GUIDANCE_STATUS,
  setCatIssueGuidanceStatus,
} from "@/components/content-editor/issues/content-editor-issue-guidance-event";
import {
  CAT_GLOSSARY_GUIDANCE_OPEN_EVENT,
  EMPTY_CAT_GLOSSARY_GUIDANCE_STATUS,
  setCatGlossaryGuidanceStatus,
  type ContentEditorGlossaryGuidanceStatus,
} from "@/components/content-editor/intelligence/content-editor-glossary-guidance-event";

import { AppShellFooter } from "./app-shell-footer";

const currentUser = {
  avatarUrl: null,
  email: appShellStoryUser.email,
  name: appShellStoryUser.name,
};

function FooterStoryFrame({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient()}>
      <AppShellStoreProvider defaultNavigationGroups={[]}>
        <div
          className="min-h-screen bg-muted/20 text-foreground"
          style={{ "--app-shell-plan-footer-height": "3rem" } as CSSProperties}
        >
          <div className="mx-auto max-w-5xl px-6 py-10">
            <p className="text-sm text-muted-foreground">
              App shell content placeholder so the fixed footer is shown in context.
            </p>
          </div>
          {children}
        </div>
      </AppShellStoreProvider>
    </QueryClientProvider>
  );
}

function GuidanceStatusFrame({ children }: { children: ReactNode }) {
  setCatIssueGuidanceStatus({ available: true, openIssueCount: 2 });
  setCatGlossaryGuidanceStatus({ preferredCount: 2, notRecommendedCount: 1, matchCount: 2 });

  useEffect(() => {
    return () => {
      setCatIssueGuidanceStatus(EMPTY_CAT_ISSUE_GUIDANCE_STATUS);
      setCatGlossaryGuidanceStatus(EMPTY_CAT_GLOSSARY_GUIDANCE_STATUS);
    };
  }, []);

  return children;
}

function GlossaryStatusFrame({
  status,
  children,
}: {
  status: ContentEditorGlossaryGuidanceStatus;
  children: ReactNode;
}) {
  setCatGlossaryGuidanceStatus(status);

  useEffect(() => {
    return () => setCatGlossaryGuidanceStatus(EMPTY_CAT_GLOSSARY_GUIDANCE_STATUS);
  }, []);

  return children;
}

const meta = {
  title: "App Shell/Footer",
  component: AppShellFooter,
  parameters: {
    layout: "fullscreen",
    nextjs: {
      appDirectory: true,
      navigation: {
        pathname: "/org/acme/dashboard",
      },
    },
  },
  decorators: [
    (Story) => (
      <FooterStoryFrame>
        <Story />
      </FooterStoryFrame>
    ),
  ],
  args: {
    organizationSlug: APP_SHELL_STORY_ORGANIZATION_SLUG,
    showPlan: false,
    showGlossaryGuidance: false,
    showIssueGuidance: false,
  },
} satisfies Meta<typeof AppShellFooter>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SupportOnly: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("link", { name: "Email support" })).toBeInTheDocument();
    await expect(canvas.queryByRole("button", { name: "New request" })).not.toBeInTheDocument();
    await expect(canvas.queryByText("Board")).not.toBeInTheDocument();
  },
};

export const GuidanceAvailable: Story = {
  args: {
    showGlossaryGuidance: true,
    showIssueGuidance: true,
  },
  decorators: [
    (Story) => (
      <GuidanceStatusFrame>
        <Story />
      </GuidanceStatusFrame>
    ),
  ],
  play: async ({ canvas }) => {
    const glossaryGuidanceOpened = fn();
    const issueGuidanceOpened = fn();
    window.addEventListener(CAT_GLOSSARY_GUIDANCE_OPEN_EVENT, glossaryGuidanceOpened);
    window.addEventListener(CAT_ISSUE_GUIDANCE_OPEN_EVENT, issueGuidanceOpened);

    const glossaryButton = canvas.getByRole("button", {
      name: "Glossary guidance, concept matches available",
    });
    const issuesButton = canvas.getByRole("button", { name: "Open board, 2 open" });

    await expect(glossaryButton).toBeInTheDocument();
    await expect(issuesButton).toBeInTheDocument();
    await expect(within(glossaryButton).getByText("2")).toBeInTheDocument();
    await expect(within(glossaryButton).getByText("1")).toBeInTheDocument();
    await expect(within(issuesButton).getByText("2")).toBeInTheDocument();

    try {
      await userEvent.click(glossaryButton);
      await expect(glossaryGuidanceOpened).toHaveBeenCalledTimes(1);

      await userEvent.click(issuesButton);
      await expect(issueGuidanceOpened).toHaveBeenCalledTimes(1);
    } finally {
      window.removeEventListener(CAT_GLOSSARY_GUIDANCE_OPEN_EVENT, glossaryGuidanceOpened);
      window.removeEventListener(CAT_ISSUE_GUIDANCE_OPEN_EVENT, issueGuidanceOpened);
    }
  },
};

export const GlossaryDraftMatchAvailable: Story = {
  args: {
    showGlossaryGuidance: true,
  },
  decorators: [
    (Story) => (
      <GlossaryStatusFrame status={{ preferredCount: 0, notRecommendedCount: 0, matchCount: 1 }}>
        <Story />
      </GlossaryStatusFrame>
    ),
  ],
  play: async ({ canvas }) => {
    const glossaryButton = canvas.getByRole("button", {
      name: "Glossary guidance, concept matches available",
    });

    await expect(glossaryButton).toBeInTheDocument();
    await expect(within(glossaryButton).queryByText("1")).not.toBeInTheDocument();
    await expect(within(glossaryButton).queryByText("2")).not.toBeInTheDocument();
  },
};

export const GlossaryDefault: Story = {
  args: {
    showGlossaryGuidance: true,
  },
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("button", { name: "Open glossary guidance" }),
    ).toBeInTheDocument();
  },
};

export const StyleGuide: Story = {
  args: {
    showStyleGuide: true,
    projectId: "project_website",
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: "Open style guide" })).toBeInTheDocument();
  },
};

export const ChatEnabled: Story = {
  args: {
    currentUser,
  },
  play: async ({ canvas }) => {
    const newRequest = canvas.getByRole("button", { name: "New request" });
    await expect(newRequest).toBeInTheDocument();

    await userEvent.click(newRequest);

    await expect(canvas.getByRole("tablist", { name: "Chat conversations" })).toBeInTheDocument();
    await expect(canvas.getByRole("tab", { name: /New chat/i })).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Email support" })).toBeInTheDocument();
  },
};

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
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect, userEvent, within } from "storybook/test";

import {
  APP_SHELL_STORY_ORGANIZATION_SLUG,
  APP_SHELL_STORY_PROJECT_ID,
  AppShellHeaderActionDemo,
  AppShellStoryFrame,
  appShellStoryTmsConnectCta,
  appShellStoryUser,
} from "./app-shell.stories.fixture";

const meta = {
  title: "App Shell/Shell",
  parameters: {
    layout: "fullscreen",
    nextjs: {
      appDirectory: true,
      navigation: {
        pathname: `/org/${APP_SHELL_STORY_ORGANIZATION_SLUG}/dashboard`,
      },
    },
  },
  render: () => <AppShellStoryFrame />,
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const WorkspaceDashboard: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: /Account/i })).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Email support" })).toBeInTheDocument();
    await expect(canvas.getByText("Overview")).toBeInTheDocument();
  },
};

export const ProjectSettings: Story = {
  parameters: {
    nextjs: {
      appDirectory: true,
      navigation: {
        pathname: `/org/${APP_SHELL_STORY_ORGANIZATION_SLUG}/projects/${APP_SHELL_STORY_PROJECT_ID}/settings`,
      },
    },
  },
  render: () => (
    <AppShellStoryFrame>
      <AppShellHeaderActionDemo />
    </AppShellStoryFrame>
  ),
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: "Save changes" })).toBeInTheDocument();
    await expect(canvas.getByText("Settings")).toBeInTheDocument();
  },
};

export const TmsConnectPrompt: Story = {
  parameters: {
    nextjs: {
      appDirectory: true,
      navigation: {
        pathname: `/org/${APP_SHELL_STORY_ORGANIZATION_SLUG}/dashboard`,
      },
    },
  },
  render: () => <AppShellStoryFrame tmsUserConnectCta={appShellStoryTmsConnectCta} />,
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: /Connect Crowdin/i })).toBeInTheDocument();
  },
};

export const ContentEditorFooter: Story = {
  parameters: {
    nextjs: {
      appDirectory: true,
      navigation: {
        pathname: `/org/${APP_SHELL_STORY_ORGANIZATION_SLUG}/projects/${APP_SHELL_STORY_PROJECT_ID}/strings`,
      },
    },
  },
  render: () => (
    <AppShellStoryFrame autumnConfigured showBillingLink>
      <p className="text-sm text-muted-foreground">
        Content editor routes enable style guide, glossary, and issue guidance controls in the
        footer.
      </p>
    </AppShellStoryFrame>
  ),
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: "Open style guide" })).toBeInTheDocument();
    await expect(
      canvas.getByRole("button", { name: "Open glossary guidance" }),
    ).toBeInTheDocument();
    await expect(canvas.getByRole("button", { name: "Open board" })).toBeInTheDocument();
    await expect(canvas.getByRole("button", { name: /Open plan usage:/i })).toBeInTheDocument();
  },
};

export const LimitedAdminMenu: Story = {
  render: () => (
    <AppShellStoryFrame showApiKeysLink={false} showBillingLink={false} showMembersLink={false} />
  ),
  play: async ({ canvas, userEvent: storyUserEvent }) => {
    await storyUserEvent.click(canvas.getByRole("button", { name: /Account/i }));

    const menu = within(document.body);
    await expect(menu.getByRole("menuitem", { name: "Account" })).toBeInTheDocument();
    await expect(menu.queryByRole("menuitem", { name: "Members" })).not.toBeInTheDocument();
    await expect(menu.queryByRole("menuitem", { name: "Billing" })).not.toBeInTheDocument();
  },
};

export const AccountMenuOpen: Story = {
  play: async ({ canvas }) => {
    const accountButton = canvas.getByRole("button", { name: /Account/i });
    await userEvent.click(accountButton);

    const menu = within(document.body);
    await expect(menu.getByText(appShellStoryUser.name)).toBeInTheDocument();
    await expect(menu.getByText(appShellStoryUser.email)).toBeInTheDocument();
    await expect(menu.getByRole("radio", { name: "Light" })).toBeInTheDocument();
    await expect(menu.getByRole("radio", { name: "Dark" })).toBeInTheDocument();
    await expect(menu.getByRole("radio", { name: "System" })).toBeInTheDocument();
  },
};

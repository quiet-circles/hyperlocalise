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
// @vitest-environment happy-dom

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import { IntlProvider } from "react-intl";

import { AppShellFooter } from "@/components/app-shell/app-shell-footer";
import { AppShellStoreProvider } from "@/components/app-shell/store/app-shell-store-context";
import {
  CAT_ISSUE_GUIDANCE_OPEN_EVENT,
  EMPTY_CAT_ISSUE_GUIDANCE_STATUS,
  setCatIssueGuidanceStatus,
} from "@/components/content-editor/issues/content-editor-issue-guidance-event";
import { planUsagePrimaryFeatureId } from "@/lib/billing/plan-usage";

const autumnMocks = vi.hoisted(() => ({
  useCustomer: vi.fn(),
  useListPlans: vi.fn(),
}));

vi.mock("autumn-js/react", () => autumnMocks);

vi.mock("next/navigation", () => ({
  usePathname: () => "/org/acme/dashboard",
}));

vi.mock("@/components/content-editor/style-guide/content-editor-style-guide-sheet", () => ({
  ContentEditorStyleGuideSheet: ({ open, projectId }: { open: boolean; projectId: string }) =>
    open ? (
      <div role="dialog" aria-label="Style guide">
        {projectId}
      </div>
    ) : null,
}));

afterEach(() => {
  autumnMocks.useCustomer.mockReset();
  autumnMocks.useListPlans.mockReset();
  setCatIssueGuidanceStatus(EMPTY_CAT_ISSUE_GUIDANCE_STATUS);
});

function renderFooter(
  props: {
    organizationSlug?: string;
    projectId?: string | null;
    showPlan?: boolean;
    withChat?: boolean;
    showIssueGuidance?: boolean;
    showStyleGuide?: boolean;
  } = {},
) {
  const {
    organizationSlug = "acme",
    projectId = null,
    showPlan = true,
    withChat = false,
    showIssueGuidance = false,
    showStyleGuide = false,
  } = props;

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <IntlProvider locale="en" messages={{}}>
        <AppShellStoreProvider defaultNavigationGroups={[]}>
          <AppShellFooter
            organizationSlug={organizationSlug}
            projectId={projectId}
            showPlan={showPlan}
            showIssueGuidance={showIssueGuidance}
            showStyleGuide={showStyleGuide}
            currentUser={
              withChat
                ? {
                    avatarUrl: null,
                    email: "user@example.com",
                    name: "Test User",
                  }
                : undefined
            }
          />
        </AppShellStoreProvider>
      </IntlProvider>
    </QueryClientProvider>,
  );
}

describe("AppShellFooter", () => {
  it("opens plan usage from the fixed footer control", async () => {
    const user = userEvent.setup();
    autumnMocks.useCustomer.mockReturnValue({
      data: {
        subscriptions: [
          {
            planId: "growth",
            status: "active",
            plan: { name: "Growth" },
          },
        ],
        balances: {
          [planUsagePrimaryFeatureId]: {
            usage: 25,
            granted: 100,
            remaining: 75,
          },
        },
      },
      isLoading: false,
      error: null,
    });
    autumnMocks.useListPlans.mockReturnValue({
      data: [{ id: "growth", name: "Growth" }],
      isLoading: false,
      error: null,
    });

    renderFooter({ showPlan: true });

    await user.click(screen.getByRole("button", { name: "Open plan usage: Growth" }));

    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.getByText("Your workspace is on the Growth plan")).toBeTruthy();
    expect(screen.getByText("25 / 100 AI credits used")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Open billing" }).getAttribute("href")).toBe(
      "/org/acme/settings/billing#plan-usage",
    );
  });

  it("keeps support available without billing access", () => {
    renderFooter({ showPlan: false });

    expect(screen.queryByText("Growth")).toBeNull();
    expect(screen.getByRole("link", { name: "Email support" }).getAttribute("href")).toBe(
      "mailto:minh@hyperlocalise.com",
    );
  });

  it("hosts chat tabs on the right of the footer status row with support", async () => {
    const user = userEvent.setup();
    autumnMocks.useCustomer.mockReturnValue({
      data: { id: "cus_1" },
      isLoading: false,
      error: null,
      check: () => ({ allowed: true }),
    });
    autumnMocks.useListPlans.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    });

    renderFooter({ showPlan: false, withChat: true });

    const newChat = screen.getByRole("button", { name: "New request" });
    const support = screen.getByRole("link", { name: "Email support" });
    expect(newChat.closest("footer")).toBeTruthy();
    expect(support.closest("footer")).toBeTruthy();
    expect(
      newChat.compareDocumentPosition(support) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(newChat);
    const tablist = screen.getByRole("tablist", { name: "Chat conversations" });
    expect(tablist.closest("footer")).toBeTruthy();
    expect(screen.getByRole("tab", { name: /New chat/i })).toBeTruthy();
  });

  it("opens contextual issues from the footer and shows the open count", async () => {
    const user = userEvent.setup();
    const openListener = vi.fn();
    window.addEventListener(CAT_ISSUE_GUIDANCE_OPEN_EVENT, openListener);
    setCatIssueGuidanceStatus({ available: true, openIssueCount: 2 });

    renderFooter({ showPlan: false, showIssueGuidance: true });

    const issues = screen.getByRole("button", { name: "Open board, 2 open" });
    expect(issues).toHaveTextContent("Board");
    expect(issues).toHaveTextContent("2");

    try {
      await user.click(issues);
      expect(openListener).toHaveBeenCalledTimes(1);
    } finally {
      window.removeEventListener(CAT_ISSUE_GUIDANCE_OPEN_EVENT, openListener);
    }
  });

  it("opens the style guide sheet from the footer on content editor routes", async () => {
    const user = userEvent.setup();
    renderFooter({
      showPlan: false,
      showStyleGuide: true,
      projectId: "project_1",
    });

    expect(screen.queryByRole("dialog", { name: "Style guide" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Open style guide" }));

    expect(screen.getByRole("dialog", { name: "Style guide" })).toHaveTextContent("project_1");
  });
});

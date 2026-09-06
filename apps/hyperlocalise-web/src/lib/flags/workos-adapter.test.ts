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
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import type { ReadonlyRequestCookies } from "flags";
import type { IntlShape } from "react-intl";

import { buildGlobalNavigationGroups } from "@/components/app-shell/navigation-config";
import {
  WORKSPACE_AUTOMATIONS_FLAG,
  WORKSPACE_KNOWLEDGE_FLAG,
} from "@/lib/flags/workos-flag-entities";
import {
  filterNavigationByWorkspaceFlags,
  annotateNavigationByWorkspaceFlags,
  groupPreviewNavigationGroups,
} from "@/lib/flags/workspace-flag-navigation";
import { getIntlShape } from "@/lib/app-i18n/intl";

const isEnabled = vi.fn();
const waitUntilReady = vi.fn().mockResolvedValue(undefined);

vi.mock("@workos-inc/authkit-nextjs", () => ({
  getFeatureFlagsRuntimeClient: () => ({
    isEnabled,
    waitUntilReady,
  }),
}));

vi.mock("@/lib/workos/config", () => ({
  getWorkosAuthKitConfig: () => ({
    apiKey: "sk_test",
    clientId: "client_test",
    redirectUri: "http://localhost:3000/auth/callback",
    cookiePassword: "test-workos-cookie-password-at-least-32-chars",
  }),
}));

describe("workosAdapter", () => {
  beforeEach(() => {
    vi.resetModules();
    isEnabled.mockReset();
    waitUntilReady.mockClear();
    waitUntilReady.mockResolvedValue(undefined);
  });

  function createMockCookies(): ReadonlyRequestCookies {
    return {
      get: () => undefined,
      getAll: () => [],
      has: () => false,
      size: 0,
      [Symbol.iterator]: () => [][Symbol.iterator](),
    } as unknown as ReadonlyRequestCookies;
  }
  it("passes organization and user ids to the WorkOS runtime client", async () => {
    isEnabled.mockResolvedValue(true);

    const { createWorkosAdapter } = await import("./workos-adapter");
    const adapter = createWorkosAdapter()();
    const enabled = await adapter.decide({
      key: WORKSPACE_AUTOMATIONS_FLAG,
      entities: {
        user: { id: "user_123" },
        organization: { id: "org_456" },
      },
      headers: new Headers(),
      cookies: createMockCookies(),
    });

    expect(enabled).toBe(true);
    expect(waitUntilReady).toHaveBeenCalledTimes(1);
    expect(waitUntilReady).toHaveBeenCalledWith({ timeoutMs: 2_000 });
    expect(isEnabled).toHaveBeenCalledWith(WORKSPACE_AUTOMATIONS_FLAG, {
      organizationId: "org_456",
      userId: "user_123",
    });
  });

  it("waits for feature flag readiness only once per process", async () => {
    isEnabled.mockResolvedValue(false);

    const { createWorkosAdapter } = await import("./workos-adapter");
    const adapter = createWorkosAdapter()();
    const decideArgs = {
      key: WORKSPACE_AUTOMATIONS_FLAG,
      entities: {
        user: { id: "user_123" },
        organization: { id: "org_456" },
      },
      headers: new Headers(),
      cookies: createMockCookies(),
    };

    await adapter.decide(decideArgs);
    await adapter.decide({ ...decideArgs, key: WORKSPACE_KNOWLEDGE_FLAG });

    expect(waitUntilReady).toHaveBeenCalledTimes(1);
    expect(isEnabled).toHaveBeenCalledTimes(2);
  });

  it("returns false when isEnabled rejects", async () => {
    isEnabled.mockRejectedValue(new Error("WorkOS API error"));

    const { createWorkosAdapter } = await import("./workos-adapter");
    const adapter = createWorkosAdapter()();
    const enabled = await adapter.decide({
      key: WORKSPACE_AUTOMATIONS_FLAG,
      entities: {
        user: { id: "user_123" },
        organization: { id: "org_456" },
      },
      headers: new Headers(),
      cookies: createMockCookies(),
    });

    expect(enabled).toBe(false);
  });

  it("returns false when WorkOS is disabled", async () => {
    vi.resetModules();
    vi.doMock("@/lib/workos/config", () => ({
      getWorkosAuthKitConfig: () => null,
    }));

    const { createWorkosAdapter } = await import("./workos-adapter");
    const adapter = createWorkosAdapter()();
    const enabled = await adapter.decide({
      key: WORKSPACE_KNOWLEDGE_FLAG,
      entities: {
        user: { id: "user_123" },
        organization: { id: "org_456" },
      },
      headers: new Headers(),
      cookies: createMockCookies(),
    });

    expect(enabled).toBe(false);
    expect(isEnabled).not.toHaveBeenCalled();

    vi.doUnmock("@/lib/workos/config");
    vi.resetModules();
  });
});

const intl = getIntlShape("en") as IntlShape;

describe("annotateNavigationByWorkspaceFlags", () => {
  it("marks Automations, Guideline, and Domains as preview when workspace flags are disabled", () => {
    const groups = buildGlobalNavigationGroups("acme", intl);
    const annotated = annotateNavigationByWorkspaceFlags(groups, {
      automations: false,
      knowledge: false,
      visualMock: false,
      visualWorkflows: false,
      domains: false,
      glossarySearch: false,
      hyperlab: false,
      reports: false,
    });

    const automationsItem = annotated
      .flatMap((group) => group.items)
      .find((item) => item.label === "Automations");
    const guidelineItem = annotated
      .flatMap((group) => group.items)
      .find((item) => item.label === "Guideline");
    const domainsItem = annotated
      .flatMap((group) => group.items)
      .find((item) => item.label === "Domains");
    const reportsItem = annotated
      .flatMap((group) => group.items)
      .find((item) => item.label === "Reports");

    expect(automationsItem?.preview).toBe(true);
    expect(guidelineItem?.preview).toBe(true);
    expect(domainsItem?.preview).toBe(true);
    expect(reportsItem?.preview).toBe(true);
  });

  it("clears preview badges when workspace flags are enabled", () => {
    const groups = buildGlobalNavigationGroups("acme", intl);
    const annotated = annotateNavigationByWorkspaceFlags(groups, {
      automations: true,
      knowledge: true,
      visualMock: true,
      visualWorkflows: true,
      domains: true,
      glossarySearch: true,
      hyperlab: true,
      reports: true,
    });

    const flaggedItems = annotated
      .flatMap((group) => group.items)
      .filter((item) => item.featureFlagKey);

    expect(flaggedItems.every((item) => item.preview !== true)).toBe(true);
  });
});

describe("filterNavigationByWorkspaceFlags", () => {
  it("removes Automations, Guideline, and Domains when workspace flags are disabled", () => {
    const groups = buildGlobalNavigationGroups("acme", intl);
    const filtered = filterNavigationByWorkspaceFlags(groups, {
      automations: false,
      knowledge: false,
      visualMock: false,
      visualWorkflows: false,
      domains: false,
      glossarySearch: false,
      hyperlab: false,
      reports: false,
    });

    const itemLabels = filtered.flatMap((group) => group.items.map((item) => item.label));

    expect(itemLabels).not.toContain("Automations");
    expect(itemLabels).not.toContain("Guideline");
    expect(itemLabels).toContain("Board");
    expect(itemLabels).toContain("New Request");
    expect(itemLabels).toContain("AI Engine");
    expect(itemLabels).not.toContain("Domains");
    expect(itemLabels).not.toContain("Reports");
    expect(itemLabels).toContain("Projects");
  });

  it("keeps Automations, Guideline, Issues, and Domains when workspace flags are enabled", () => {
    const groups = buildGlobalNavigationGroups("acme", intl);
    const filtered = filterNavigationByWorkspaceFlags(groups, {
      automations: true,
      knowledge: true,
      visualMock: true,
      visualWorkflows: true,
      domains: true,
      glossarySearch: true,
      hyperlab: true,
      reports: true,
    });

    const itemLabels = filtered.flatMap((group) => group.items.map((item) => item.label));

    expect(itemLabels).toContain("Automations");
    expect(itemLabels).toContain("Guideline");
    expect(itemLabels).toContain("Board");
    expect(itemLabels).toContain("Domains");
    expect(itemLabels).toContain("Reports");
    expect(itemLabels).toContain("AI Engine");
  });
});

describe("groupPreviewNavigationGroups", () => {
  it("moves preview items into a Try section and leaves enabled items in place", () => {
    const groups = annotateNavigationByWorkspaceFlags(buildGlobalNavigationGroups("acme", intl), {
      automations: false,
      knowledge: false,
      visualMock: false,
      visualWorkflows: false,
      domains: false,
      glossarySearch: false,
      hyperlab: false,
      reports: false,
    });
    const grouped = groupPreviewNavigationGroups(groups, "Try");

    const tryGroup = grouped.find((group) => group.label === "Try");
    const agentsGroup = grouped.find((group) => group.label === "Agents");
    const workspaceGroup = grouped.find((group) => group.label === "Workspace");
    const promotedLabels = grouped[0]?.items.map((item) => item.label) ?? [];

    expect(tryGroup?.items.map((item) => item.label)).toEqual([
      "Reports",
      "Automations",
      "Domains",
      "Hyperlab",
      "Guideline",
    ]);
    expect(agentsGroup?.items.map((item) => item.label)).toEqual(["New Request", "AI Engine"]);
    expect(workspaceGroup?.items.map((item) => item.label)).not.toContain("Domains");
    expect(workspaceGroup?.items.map((item) => item.label)).toContain("Projects");
    expect(promotedLabels).toContain("Inbox");
    expect(promotedLabels).not.toContain("Reports");
  });

  it("omits the Try section when no items are in preview", () => {
    const groups = annotateNavigationByWorkspaceFlags(buildGlobalNavigationGroups("acme", intl), {
      automations: true,
      knowledge: true,
      visualMock: true,
      visualWorkflows: true,
      domains: true,
      glossarySearch: true,
      hyperlab: true,
      reports: true,
    });
    const grouped = groupPreviewNavigationGroups(groups, "Try");

    expect(grouped.some((group) => group.label === "Try")).toBe(false);
    expect(grouped.flatMap((group) => group.items.map((item) => item.label))).toContain(
      "Automations",
    );
  });
});

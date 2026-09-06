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
import { describe, expect, it } from "vite-plus/test";
import type { IntlShape } from "react-intl";

import { getIntlShape } from "@/lib/app-i18n/intl";

import {
  WORKSPACE_AUTOMATIONS_FLAG,
  WORKSPACE_DOMAINS_FLAG,
  WORKSPACE_HYPERLAB_FLAG,
  WORKSPACE_KNOWLEDGE_FLAG,
} from "@/lib/flags/workos-flag-entities";
import { RELEASE_CAT_ALL_FILES_FLAG } from "@/lib/flags/release-flag-keys";
import { encodeProviderProjectId } from "@/lib/providers/jobs/tms-provider-resource-id";

import {
  buildAutomationsPath,
  buildGlobalNavigationGroups,
  buildOrganizationPath,
  buildProjectNavigationItems,
  buildProjectPath,
  buildTeamPath,
  buildDomainPath,
  isInboxNewRequestPath,
  isNavigationItemActive,
  isOrganizationSettingsPath,
  parseDomainRoute,
  parseProjectRoute,
  parseTeamRoute,
  stripAppLocalePrefix,
} from "./navigation-config";

const intl = getIntlShape("en") as IntlShape;

describe("navigation-config", () => {
  it("encodes external project ids as one path segment and parses them back", () => {
    const projectId = "ext:crowdin:902807";
    const path = buildProjectPath("hyperlocalise", projectId, "files");

    expect(path).toBe("/org/hyperlocalise/projects/ext%3Acrowdin%3A902807/files");
    expect(parseProjectRoute(path)).toEqual({
      organizationSlug: "hyperlocalise",
      projectId,
      section: "files",
    });
  });

  it("parses project routes with a locale prefix", () => {
    const projectId = "proj_1";

    expect(parseProjectRoute("/en/org/acme/projects/proj_1")).toEqual({
      organizationSlug: "acme",
      projectId,
      section: null,
    });
    expect(parseProjectRoute("/en/org/acme/projects/proj_1/files")).toEqual({
      organizationSlug: "acme",
      projectId,
      section: "files",
    });
  });

  it("marks navigation active for locale-prefixed project paths", () => {
    const projectId = "proj_1";
    const filesHref = buildProjectPath("acme", projectId, "files");

    expect(
      isNavigationItemActive("/en/org/acme/projects/proj_1/files", filesHref, {
        organizationSlug: "acme",
        projectId,
      }),
    ).toBe(true);
  });

  it("keeps project overview active state exact for encoded external project ids", () => {
    const projectId = "ext:crowdin:902807";
    const overviewHref = buildProjectPath("hyperlocalise", projectId);
    const filesPath = buildProjectPath("hyperlocalise", projectId, "files");

    expect(
      isNavigationItemActive(filesPath, overviewHref, {
        organizationSlug: "hyperlocalise",
        projectId,
      }),
    ).toBe(false);
    expect(
      isNavigationItemActive(overviewHref, overviewHref, {
        organizationSlug: "hyperlocalise",
        projectId,
      }),
    ).toBe(true);
  });

  it("marks only the matching project subpage active", () => {
    const projectId = "proj_1";
    const filesHref = buildProjectPath("acme", projectId, "files");
    const jobsHref = buildProjectPath("acme", projectId, "jobs");

    expect(
      isNavigationItemActive(filesHref, filesHref, {
        organizationSlug: "acme",
        projectId,
      }),
    ).toBe(true);
    expect(
      isNavigationItemActive(filesHref, jobsHref, {
        organizationSlug: "acme",
        projectId,
      }),
    ).toBe(false);
  });
});

describe("workspace people navigation", () => {
  it("marks teams active for team detail routes", () => {
    const teamsHref = "/org/acme/teams";

    expect(isNavigationItemActive("/org/acme/teams", teamsHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/teams/team_1", teamsHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/members", teamsHref)).toBe(false);
  });

  it("marks members active only on members routes", () => {
    const membersHref = "/org/acme/members";

    expect(isNavigationItemActive("/org/acme/members", membersHref)).toBe(true);
    expect(isNavigationItemActive("/en/org/acme/members", membersHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/members/permissions", membersHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/teams", membersHref)).toBe(false);
    expect(isNavigationItemActive("/org/acme/teams/team_1", membersHref)).toBe(false);
  });

  it("does not mark settings active on members routes", () => {
    const settingsHref = "/org/acme/settings";

    expect(isNavigationItemActive("/org/acme/members", settingsHref)).toBe(false);
    expect(isNavigationItemActive("/org/acme/settings/account", settingsHref)).toBe(true);
  });

  it("handles missing pathnames safely", () => {
    expect(stripAppLocalePrefix(null)).toBe("/");
    expect(stripAppLocalePrefix(undefined)).toBe("/");
    expect(isNavigationItemActive(null, "/org/acme/teams")).toBe(false);
    expect(isNavigationItemActive(undefined, "/org/acme/teams")).toBe(false);
    expect(isNavigationItemActive("", "/org/acme/teams")).toBe(false);
  });
});

describe("path builders", () => {
  it("builds organization paths", () => {
    expect(buildOrganizationPath("acme", "inbox")).toBe("/org/acme/inbox");
  });

  it("builds project paths with and without a section", () => {
    expect(buildProjectPath("acme", "proj_1")).toBe("/org/acme/projects/proj_1");
    expect(buildProjectPath("acme", "proj_1", "files")).toBe("/org/acme/projects/proj_1/files");
  });

  it("builds workspace and project automations paths", () => {
    expect(buildAutomationsPath("acme")).toBe("/org/acme/automations");
    expect(buildAutomationsPath("acme", { section: "new" })).toBe("/org/acme/automations/new");
    expect(buildAutomationsPath("acme", { automationId: "auto_1" })).toBe(
      "/org/acme/automations/auto_1",
    );
    expect(buildAutomationsPath("acme", { projectId: "proj_1" })).toBe(
      "/org/acme/projects/proj_1/automations",
    );
    expect(buildAutomationsPath("acme", { projectId: "proj_1", section: "new" })).toBe(
      "/org/acme/projects/proj_1/automations/new",
    );
    expect(
      buildAutomationsPath("acme", { projectId: "ext:crowdin:1", automationId: "auto_1" }),
    ).toBe("/org/acme/projects/ext%3Acrowdin%3A1/automations/auto_1");
  });

  it("encodes reserved characters in the project id segment", () => {
    expect(buildProjectPath("acme", "ext:crowdin:1", "jobs")).toBe(
      "/org/acme/projects/ext%3Acrowdin%3A1/jobs",
    );
  });

  it("builds global navigation groups scoped to the organization", () => {
    const groups = buildGlobalNavigationGroups("acme", intl);
    const items = groups.flatMap((group) => group.items);
    const byLabel = new Map(items.map((item) => [item.label, item]));

    expect(byLabel.get("Inbox")?.href).toBe("/org/acme/inbox");
    expect(byLabel.get("Projects")?.href).toBe("/org/acme/projects");
    expect(byLabel.get("New Request")).toMatchObject({
      href: "/org/acme/inbox/new",
      exact: true,
    });
    expect(byLabel.get("AI Engine")?.href).toBe("/org/acme/ai-engine");
    expect(byLabel.get("Automations")?.featureFlagKey).toBe(WORKSPACE_AUTOMATIONS_FLAG);
    expect(byLabel.get("Guideline")?.featureFlagKey).toBe(WORKSPACE_KNOWLEDGE_FLAG);
    expect(byLabel.get("Board")?.featureFlagKey).toBeUndefined();
    expect(byLabel.get("Domains")?.featureFlagKey).toBe(WORKSPACE_DOMAINS_FLAG);
    expect(byLabel.get("Hyperlab")?.href).toBe("/org/acme/hyperlab");
    expect(byLabel.get("Hyperlab")?.featureFlagKey).toBe(WORKSPACE_HYPERLAB_FLAG);

    expect(groups.map((group) => group.label)).toEqual([undefined, "Agents", "Workspace"]);
    expect(groups[1]?.items.map((item) => item.label)).toEqual([
      "New Request",
      "Automations",
      "AI Engine",
    ]);
    expect(groups[0]?.items.map((item) => item.label)).toEqual([
      "Inbox",
      "My Jobs",
      "Board",
      "Overview",
      "Reports",
    ]);
  });

  it("builds project navigation items scoped to the project", () => {
    const items = buildProjectNavigationItems("acme", "proj_1", intl);

    expect(items.map((item) => [item.label, item.href])).toEqual([
      ["Overview", "/org/acme/projects/proj_1"],
      ["Reports", "/org/acme/projects/proj_1/reports"],
      ["Files", "/org/acme/projects/proj_1/files"],
      ["Content Editor", "/org/acme/projects/proj_1/strings"],
      ["Jobs", "/org/acme/projects/proj_1/jobs"],
      ["Board", "/org/acme/projects/proj_1/issue-sheet"],
      ["Automations", "/org/acme/projects/proj_1/automations"],
      ["Guideline", "/org/acme/projects/proj_1/knowledge"],
      ["Settings", "/org/acme/projects/proj_1/settings"],
    ]);
    expect(items.find((item) => item.label === "Board")?.featureFlagKey).toBeUndefined();
    expect(items.find((item) => item.label === "Automations")?.featureFlagKey).toBe(
      WORKSPACE_AUTOMATIONS_FLAG,
    );
    expect(items.find((item) => item.label === "Guideline")?.featureFlagKey).toBe(
      WORKSPACE_KNOWLEDGE_FLAG,
    );
  });
});

describe("stripAppLocalePrefix", () => {
  it("removes a supported locale prefix", () => {
    expect(stripAppLocalePrefix("/en/org/acme/inbox")).toBe("/org/acme/inbox");
  });

  it("collapses a locale-only path to root", () => {
    expect(stripAppLocalePrefix("/en")).toBe("/");
    expect(stripAppLocalePrefix("/en/")).toBe("/");
  });

  it("strips trailing slashes after removing the locale", () => {
    expect(stripAppLocalePrefix("/en/org/acme/inbox/")).toBe("/org/acme/inbox");
  });

  it("leaves paths without a locale prefix untouched", () => {
    expect(stripAppLocalePrefix("/org/acme/inbox")).toBe("/org/acme/inbox");
    expect(stripAppLocalePrefix("/fr/org/acme/inbox")).toBe("/fr/org/acme/inbox");
  });

  it("removes newly supported locale prefixes", () => {
    expect(stripAppLocalePrefix("/fr-FR/org/acme/inbox")).toBe("/org/acme/inbox");
    expect(stripAppLocalePrefix("/zh-CN/blog")).toBe("/blog");
  });
});

describe("parseProjectRoute", () => {
  it("returns null for empty and non-project routes", () => {
    expect(parseProjectRoute(null)).toBeNull();
    expect(parseProjectRoute("")).toBeNull();
    expect(parseProjectRoute("/org/acme/inbox")).toBeNull();
    expect(parseProjectRoute("/org/acme/projects")).toBeNull();
  });

  it("returns a null section for the project overview route", () => {
    expect(parseProjectRoute("/org/acme/projects/proj_1")).toEqual({
      organizationSlug: "acme",
      projectId: "proj_1",
      section: null,
    });
  });

  it("only reports the first section segment for nested routes", () => {
    expect(parseProjectRoute("/org/acme/projects/proj_1/jobs/job_1/strings")).toEqual({
      organizationSlug: "acme",
      projectId: "proj_1",
      section: "jobs",
    });
  });
});

describe("parseTeamRoute", () => {
  it("returns null for empty and non-team routes", () => {
    expect(parseTeamRoute(null)).toBeNull();
    expect(parseTeamRoute("/org/acme/teams")).toBeNull();
    expect(parseTeamRoute("/org/acme/projects/proj_1")).toBeNull();
  });

  it("parses team detail routes", () => {
    expect(parseTeamRoute("/org/acme/teams/team_1")).toEqual({
      organizationSlug: "acme",
      teamId: "team_1",
    });
    expect(parseTeamRoute("/en/org/acme/teams/team_1")).toEqual({
      organizationSlug: "acme",
      teamId: "team_1",
    });
  });
});

describe("parseDomainRoute", () => {
  it("returns null for empty and non-domain routes", () => {
    expect(parseDomainRoute(null)).toBeNull();
    expect(parseDomainRoute("/org/acme/domains")).toBeNull();
    expect(parseDomainRoute("/org/acme/projects/proj_1")).toBeNull();
  });

  it("parses domain detail routes", () => {
    expect(parseDomainRoute("/org/acme/domains/ld_1")).toEqual({
      organizationSlug: "acme",
      linkedDomainId: "ld_1",
    });
    expect(parseDomainRoute("/en/org/acme/domains/ld_1")).toEqual({
      organizationSlug: "acme",
      linkedDomainId: "ld_1",
    });
  });
});

describe("buildTeamPath", () => {
  it("encodes team ids in the path", () => {
    expect(buildTeamPath("acme", "team_1")).toBe("/org/acme/teams/team_1");
  });
});

describe("buildDomainPath", () => {
  it("encodes linked domain ids in the path", () => {
    expect(buildDomainPath("acme", "ld_1")).toBe("/org/acme/domains/ld_1");
  });
});

describe("buildProjectNavigationItems", () => {
  it("includes a Content Editor item for native projects", () => {
    const items = buildProjectNavigationItems("acme", "proj_1", intl);
    const contentEditorItem = items.find((item) => item.label === "Content Editor");
    expect(contentEditorItem?.href).toBe("/org/acme/projects/proj_1/strings");
    expect(contentEditorItem?.featureFlagKey).toBe(RELEASE_CAT_ALL_FILES_FLAG);
  });

  it("shows the Content Editor for Crowdin projects", () => {
    const projectId = encodeProviderProjectId({
      providerKind: "crowdin",
      externalProjectId: "902807",
    });
    const items = buildProjectNavigationItems("acme", projectId, intl);
    expect(items.find((item) => item.label === "Content Editor")?.href).toBe(
      `/org/acme/projects/${encodeURIComponent(projectId)}/strings`,
    );
  });

  it("hides the Content Editor for unsupported TMS providers", () => {
    const projectId = encodeProviderProjectId({
      providerKind: "phrase",
      externalProjectId: "42",
    });
    const items = buildProjectNavigationItems("acme", projectId, intl);
    expect(items.find((item) => item.label === "Content Editor")).toBeUndefined();
  });

  it("includes an Automations item gated by the workspace automations flag", () => {
    const items = buildProjectNavigationItems("acme", "proj_1", intl);
    const automationsItem = items.find((item) => item.label === "Automations");
    expect(automationsItem?.href).toBe("/org/acme/projects/proj_1/automations");
    expect(automationsItem?.featureFlagKey).toBe(WORKSPACE_AUTOMATIONS_FLAG);
  });
});

describe("isNavigationItemActive", () => {
  it("matches exactly when the exact option is set", () => {
    expect(isNavigationItemActive("/org/acme/inbox", "/org/acme/inbox", { exact: true })).toBe(
      true,
    );
    expect(
      isNavigationItemActive("/org/acme/inbox/thread_1", "/org/acme/inbox", { exact: true }),
    ).toBe(false);
  });

  it("matches nested subpaths for non-exact items", () => {
    expect(isNavigationItemActive("/org/acme/inbox/thread_1", "/org/acme/inbox")).toBe(true);
  });

  it("keeps Inbox inactive on the dedicated New Request compose route", () => {
    expect(isNavigationItemActive("/org/acme/inbox/new", "/org/acme/inbox")).toBe(false);
    expect(isNavigationItemActive("/en/org/acme/inbox/new", "/org/acme/inbox")).toBe(false);
    expect(
      isNavigationItemActive("/org/acme/inbox/new", "/org/acme/inbox/new", { exact: true }),
    ).toBe(true);
  });

  it("ignores the hash fragment on the item href", () => {
    expect(isNavigationItemActive("/org/acme/settings", "/org/acme/settings#profile")).toBe(true);
  });

  it("keeps the projects list inactive on project detail routes", () => {
    expect(isNavigationItemActive("/org/acme/projects", "/org/acme/projects")).toBe(true);
    expect(isNavigationItemActive("/org/acme/projects/proj_1", "/org/acme/projects")).toBe(false);
    expect(isNavigationItemActive("/org/acme/projects/proj_1/files", "/org/acme/projects")).toBe(
      false,
    );
  });

  it("treats my-work and my-jobs as the same destination", () => {
    const myWorkHref = "/org/acme/my-work";

    expect(isNavigationItemActive("/org/acme/my-work", myWorkHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/my-jobs", myWorkHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/my-jobs/job_1", myWorkHref)).toBe(true);
    expect(isNavigationItemActive("/en/org/acme/my-jobs", myWorkHref)).toBe(true);
    expect(isNavigationItemActive("/org/acme/inbox", myWorkHref)).toBe(false);
  });
});

describe("isInboxNewRequestPath", () => {
  it("matches locale-prefixed and bare inbox compose paths", () => {
    expect(isInboxNewRequestPath("/org/acme/inbox/new")).toBe(true);
    expect(isInboxNewRequestPath("/en/org/acme/inbox/new")).toBe(true);
    expect(isInboxNewRequestPath("/org/acme/inbox")).toBe(false);
    expect(isInboxNewRequestPath("/org/acme/inbox/thread_1")).toBe(false);
  });
});

describe("isOrganizationSettingsPath", () => {
  it("matches org settings routes, including locale prefixes", () => {
    expect(isOrganizationSettingsPath("/org/acme/settings")).toBe(true);
    expect(isOrganizationSettingsPath("/org/acme/settings/billing")).toBe(true);
    expect(isOrganizationSettingsPath("/en/org/acme/settings/account")).toBe(true);
  });

  it("does not match project settings or other org routes", () => {
    expect(isOrganizationSettingsPath("/org/acme/projects/proj_1/settings")).toBe(false);
    expect(isOrganizationSettingsPath("/org/acme/members")).toBe(false);
    expect(isOrganizationSettingsPath(null)).toBe(false);
  });
});

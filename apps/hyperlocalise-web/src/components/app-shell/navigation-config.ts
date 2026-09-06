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
import type { ComponentProps } from "react";
import type { IntlShape } from "react-intl";

import { normalizeAppLocale } from "@/lib/app-i18n/locales";
import { RELEASE_CAT_ALL_FILES_FLAG } from "@/lib/flags/release-flag-keys";
import {
  WORKSPACE_AUTOMATIONS_FLAG,
  WORKSPACE_DOMAINS_FLAG,
  WORKSPACE_HYPERLAB_FLAG,
  WORKSPACE_KNOWLEDGE_FLAG,
  WORKSPACE_REPORTS_FLAG,
} from "@/lib/flags/workos-flag-entities";
import { supportsContentEditorAllFilesProvider } from "@/lib/projects/content-editor-all-files";
import { parseProviderProjectId } from "@/lib/providers/jobs/tms-provider-resource-id";
import {
  BookOpenTextIcon,
  Bookmark01Icon,
  CenterFocusIcon,
  ChartHistogramIcon,
  Copy01Icon,
  CubeIcon,
  DashboardSquare01Icon,
  Database01Icon,
  File01Icon,
  FlashIcon,
  FlaskConicalIcon,
  Globe02Icon,
  InboxIcon,
  LanguageCircleIcon,
  PuzzleIcon,
  SentIcon,
  Settings01Icon,
  SparklesIcon,
  UserMultiple02Icon,
} from "@hugeicons/core-free-icons";
import type { HugeiconsIcon } from "@hugeicons/react";

export type NavigationIcon = ComponentProps<typeof HugeiconsIcon>["icon"];

export type NavigationItem = {
  label: string;
  href: string;
  icon: NavigationIcon;
  description?: string;
  badge?: string;
  /** When true, only the exact href is active (not nested paths). */
  exact?: boolean;
  featureFlagKey?:
    | typeof WORKSPACE_AUTOMATIONS_FLAG
    | typeof WORKSPACE_KNOWLEDGE_FLAG
    | typeof WORKSPACE_DOMAINS_FLAG
    | typeof WORKSPACE_HYPERLAB_FLAG
    | typeof WORKSPACE_REPORTS_FLAG
    | typeof RELEASE_CAT_ALL_FILES_FLAG;
  /** When true, the feature flag is off and the nav item links to a teaser page. */
  preview?: boolean;
};

export type NavigationGroup = {
  label?: string;
  items: readonly NavigationItem[];
};

export function buildOrganizationPath(organizationSlug: string, section: string) {
  return `/org/${organizationSlug}/${section}`;
}

export function buildProjectPath(organizationSlug: string, projectId: string, section?: string) {
  const base = `/org/${organizationSlug}/projects/${encodeURIComponent(projectId)}`;
  return section ? `${base}/${section}` : base;
}

export function buildTeamPath(organizationSlug: string, teamId: string) {
  return `/org/${organizationSlug}/teams/${encodeURIComponent(teamId)}`;
}

export function buildDomainPath(organizationSlug: string, linkedDomainId: string) {
  return `/org/${organizationSlug}/domains/${encodeURIComponent(linkedDomainId)}`;
}

export function buildAutomationsPath(
  organizationSlug: string,
  options?: {
    automationId?: string;
    projectId?: string;
    section?: "new";
  },
) {
  const suffix = options?.section === "new" ? "new" : options?.automationId;
  if (options?.projectId) {
    return suffix
      ? buildProjectPath(organizationSlug, options.projectId, `automations/${suffix}`)
      : buildProjectPath(organizationSlug, options.projectId, "automations");
  }

  return suffix
    ? buildOrganizationPath(organizationSlug, `automations/${suffix}`)
    : buildOrganizationPath(organizationSlug, "automations");
}

export function buildGlobalNavigationGroups(
  organizationSlug: string,
  intl: IntlShape,
): readonly NavigationGroup[] {
  const org = (section: string) => buildOrganizationPath(organizationSlug, section);

  return [
    {
      items: [
        {
          label: intl.formatMessage({
            defaultMessage: "Inbox",
            id: "qYH/VTnW7r",
            description: "Sidebar navigation item for the workspace inbox",
          }),
          href: org("inbox"),
          icon: InboxIcon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "My Jobs",
            id: "7VRqQJwWUI",
            description: "Sidebar navigation item for the current user’s jobs",
          }),
          href: org("my-work"),
          icon: CenterFocusIcon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Board",
            id: "XswXu+UFpy",
            description: "Sidebar navigation item for the workspace board",
          }),
          href: org("issues"),
          icon: Copy01Icon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Overview",
            id: "M1acCMedpF",
            description: "Sidebar navigation item for the workspace dashboard overview",
          }),
          href: org("dashboard"),
          icon: DashboardSquare01Icon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Reports",
            id: "0wXT++q3xO",
            description: "Workspace translation reports",
          }),
          href: org("reports"),
          icon: ChartHistogramIcon,
          featureFlagKey: WORKSPACE_REPORTS_FLAG,
        },
      ],
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Agents",
        id: "/EOfWYVF9T",
        description: "Sidebar group label for agent navigation items",
      }),
      items: [
        {
          label: intl.formatMessage({
            defaultMessage: "New Request",
            id: "VtO24sqmBM",
            description: "Sidebar navigation item to start a new localisation request",
          }),
          href: org("inbox/new"),
          exact: true,
          icon: SentIcon,
          description: intl.formatMessage({
            defaultMessage: "Ask the localisation agent to prepare work",
            id: "z45OPLD254",
            description: "Sidebar description for the New Request navigation item",
          }),
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Automations",
            id: "87mk4HgY5S",
            description: "Sidebar navigation item for workspace automations",
          }),
          href: org("automations"),
          icon: FlashIcon,
          description: intl.formatMessage({
            defaultMessage: "Scheduled and GitHub-triggered deterministic workflows",
            id: "TBagRGINiT",
            description: "Sidebar description for the Automations navigation item",
          }),
          badge: intl.formatMessage({
            defaultMessage: "Beta",
            id: "+WwLLR9+vz",
            description: "Badge shown next to the Automations navigation item",
          }),
          featureFlagKey: WORKSPACE_AUTOMATIONS_FLAG,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "AI Engine",
            id: "Q8QL+zifeT",
            description: "Sidebar navigation item for workspace AI model providers",
          }),
          href: org("ai-engine"),
          icon: SparklesIcon,
          description: intl.formatMessage({
            defaultMessage: "Choose the model provider agents use",
            id: "ZnnVLUSfjR",
            description: "Sidebar description for the AI Engine navigation item",
          }),
        },
      ],
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Workspace",
        id: "VMLVh0fGup",
        description: "Sidebar group label for workspace-level navigation items",
      }),
      items: [
        {
          label: intl.formatMessage({
            defaultMessage: "Projects",
            id: "WXz3UNteSC",
            description: "Sidebar navigation item for the projects list",
          }),
          href: org("projects"),
          icon: CubeIcon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Domains",
            id: "sHQ6RKFJ37",
            description: "Sidebar navigation item for linked domains",
          }),
          href: org("domains"),
          icon: Globe02Icon,
          description: intl.formatMessage({
            defaultMessage: "Claimed sites and localisation audit reports",
            id: "B3yFCBQLDF",
            description: "Sidebar description for the Domains navigation item",
          }),
          featureFlagKey: WORKSPACE_DOMAINS_FLAG,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Hyperlab",
            id: "e1WibRfkfv",
            description: "Sidebar navigation item for Hyperlab experiments",
          }),
          href: org("hyperlab"),
          icon: FlaskConicalIcon,
          description: intl.formatMessage({
            defaultMessage: "Flags and experiments for your apps",
            id: "4/gpJsFcCP",
            description: "Sidebar description for the Hyperlab navigation item",
          }),
          featureFlagKey: WORKSPACE_HYPERLAB_FLAG,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Guideline",
            id: "D6HJDahz6H",
            description: "Sidebar navigation item for workspace guideline",
          }),
          href: org("knowledge"),
          icon: Bookmark01Icon,
          description: intl.formatMessage({
            defaultMessage: "Shared guidance for agents and teams",
            id: "dEzuHMWHq4",
            description: "Sidebar description for the Guideline navigation item",
          }),
          featureFlagKey: WORKSPACE_KNOWLEDGE_FLAG,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Glossaries",
            id: "p2ZEW4INMa",
            description: "Sidebar navigation item for glossaries",
          }),
          href: org("glossaries"),
          icon: BookOpenTextIcon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Translation Memories",
            id: "x6PqOisw0g",
            description: "Sidebar navigation item for translation memories",
          }),
          href: org("translation-memories"),
          icon: Database01Icon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Integrations",
            id: "lq1y6qqiDK",
            description: "Sidebar navigation item for integrations",
          }),
          href: org("integrations"),
          icon: PuzzleIcon,
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Members",
            id: "1/YUf106Rt",
            description: "Sidebar navigation item for workspace members",
          }),
          href: org("members"),
          icon: UserMultiple02Icon,
          description: intl.formatMessage({
            defaultMessage: "Invite people and manage workspace roles",
            id: "blLcFpSkB4",
            description: "Sidebar description for the Members navigation item",
          }),
        },
        {
          label: intl.formatMessage({
            defaultMessage: "Settings",
            id: "3cDDnXngWu",
            description: "Sidebar navigation item for workspace settings",
          }),
          href: org("settings"),
          icon: Settings01Icon,
        },
      ],
    },
  ] as const;
}

export function buildProjectNavigationItems(
  organizationSlug: string,
  projectId: string,
  intl: IntlShape,
): readonly NavigationItem[] {
  const project = (section: string) => buildProjectPath(organizationSlug, projectId, section);
  const providerKind = parseProviderProjectId(projectId)?.providerKind ?? null;
  const showContentEditor = supportsContentEditorAllFilesProvider(providerKind);

  const items: NavigationItem[] = [
    {
      label: intl.formatMessage({
        defaultMessage: "Overview",
        id: "w6stmLL+C3",
        description: "Project sidebar navigation item for the project overview",
      }),
      href: buildProjectPath(organizationSlug, projectId),
      icon: CubeIcon,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Reports",
        id: "MDRyeVzjZ2",
        description: "Project translation reports",
      }),
      href: project("reports"),
      icon: ChartHistogramIcon,
      featureFlagKey: WORKSPACE_REPORTS_FLAG,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Files",
        id: "IMr6sfD/7/",
        description: "Project sidebar navigation item for project files",
      }),
      href: project("files"),
      icon: File01Icon,
    },
  ];

  if (showContentEditor) {
    items.push({
      label: intl.formatMessage({
        defaultMessage: "Content Editor",
        id: "1XfD1U3TWk",
        description: "Project sidebar navigation item for the Content Editor",
      }),
      href: project("strings"),
      icon: LanguageCircleIcon,
      featureFlagKey: RELEASE_CAT_ALL_FILES_FLAG,
    });
  }

  items.push(
    {
      label: intl.formatMessage({
        defaultMessage: "Jobs",
        id: "8HNfmSDv7C",
        description: "Project sidebar navigation item for project jobs",
      }),
      href: project("jobs"),
      icon: CenterFocusIcon,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Board",
        id: "r7gtZsn8Qh",
        description: "Project sidebar navigation item for the project board",
      }),
      href: project("issue-sheet"),
      icon: Copy01Icon,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Automations",
        id: "Weqt7PXrnL",
        description: "Project sidebar navigation item for project automations",
      }),
      href: project("automations"),
      icon: FlashIcon,
      featureFlagKey: WORKSPACE_AUTOMATIONS_FLAG,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Guideline",
        id: "6dq9jGfioL",
        description: "Project sidebar navigation item for project guideline",
      }),
      href: project("knowledge"),
      icon: Bookmark01Icon,
      description: intl.formatMessage({
        defaultMessage: "Project-specific guidance for agents and teams",
        id: "tMOiEvbyBd",
        description: "Sidebar description for the project Guideline navigation item",
      }),
      featureFlagKey: WORKSPACE_KNOWLEDGE_FLAG,
    },
    {
      label: intl.formatMessage({
        defaultMessage: "Settings",
        id: "Ly3jSjXVvC",
        description: "Project sidebar navigation item for project settings",
      }),
      href: project("settings"),
      icon: Settings01Icon,
    },
  );

  return items;
}

export function stripAppLocalePrefix(pathname: string | null | undefined) {
  if (!pathname) {
    return "/";
  }

  const [, firstSegment, ...rest] = pathname.split("/");
  const locale = firstSegment ? normalizeAppLocale(firstSegment) : null;

  if (!locale) {
    return pathname;
  }

  return `/${rest.join("/")}`.replace(/\/+$/, "") || "/";
}

export function isOrganizationSettingsPath(pathname: string | null | undefined) {
  return /^\/org\/[^/]+\/settings(?:\/|$)/.test(stripAppLocalePrefix(pathname));
}

export function parseProjectRoute(pathname: string | null) {
  if (!pathname) return null;

  const match = stripAppLocalePrefix(pathname).match(
    /^\/org\/([^/]+)\/projects\/([^/]+)(?:\/(.*))?$/,
  );
  if (!match) return null;

  const [, organizationSlug, projectIdSegment, remainder] = match;
  const section = remainder?.split("/").filter(Boolean)[0] ?? null;

  return {
    organizationSlug,
    projectId: decodePathSegment(projectIdSegment),
    section,
  };
}

export function parseTeamRoute(pathname: string | null) {
  if (!pathname) return null;

  const match = stripAppLocalePrefix(pathname).match(/^\/org\/([^/]+)\/teams\/([^/]+)$/);
  if (!match) return null;

  const [, organizationSlug, teamIdSegment] = match;

  return {
    organizationSlug,
    teamId: decodePathSegment(teamIdSegment),
  };
}

export function parseDomainRoute(pathname: string | null) {
  if (!pathname) return null;

  const match = stripAppLocalePrefix(pathname).match(/^\/org\/([^/]+)\/domains\/([^/]+)$/);
  if (!match) return null;

  const [, organizationSlug, linkedDomainIdSegment] = match;

  return {
    organizationSlug,
    linkedDomainId: decodePathSegment(linkedDomainIdSegment),
  };
}

function decodePathSegment(value: string) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function isInboxNewRequestPath(pathname: string | null | undefined, inboxHref?: string) {
  const normalizedPathname = stripAppLocalePrefix(pathname ?? "");
  if (inboxHref) {
    const inboxNewHref = `${inboxHref.replace(/\/+$/, "")}/new`;
    return normalizedPathname === inboxNewHref || normalizedPathname.startsWith(`${inboxNewHref}/`);
  }

  return /\/org\/[^/]+\/inbox\/new\/?$/.test(normalizedPathname);
}

export function isNavigationItemActive(
  pathname: string | null | undefined,
  href: string,
  options?: {
    exact?: boolean;
    projectId?: string;
    organizationSlug?: string;
  },
) {
  if (!pathname) {
    return false;
  }

  const normalizedPathname = stripAppLocalePrefix(pathname);
  const itemPathname = href.split("#", 1)[0];

  if (options?.exact) {
    return normalizedPathname === itemPathname;
  }

  if (options?.projectId && options.organizationSlug) {
    const overviewHref = buildProjectPath(options.organizationSlug, options.projectId);
    if (itemPathname === overviewHref) {
      return normalizedPathname === overviewHref;
    }
  }

  if (normalizedPathname === itemPathname) {
    return true;
  }

  // New Request lives under /inbox/new; keep the Inbox item inactive there.
  if (itemPathname.endsWith("/inbox") && isInboxNewRequestPath(normalizedPathname, itemPathname)) {
    return false;
  }

  if (itemPathname.endsWith("/projects")) {
    return normalizedPathname === itemPathname;
  }

  if (normalizedPathname.startsWith(`${itemPathname}/`)) {
    return true;
  }

  if (
    itemPathname.endsWith("/my-work") &&
    normalizedPathname.startsWith(itemPathname.replace("my-work", "my-jobs"))
  ) {
    return true;
  }

  return false;
}

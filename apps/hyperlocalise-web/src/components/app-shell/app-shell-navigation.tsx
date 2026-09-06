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
import { usePathname } from "next/navigation";
import { ArrowDown01Icon, ArrowLeft01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import { observer } from "mobx-react-lite";
import { FormattedMessage, useIntl } from "react-intl";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { apiClient } from "@/lib/api-client-instance";
import { cn } from "@/lib/primitives/cn";
import {
  createInboxNotificationsApi,
  notificationsUnreadCountQueryKey,
} from "@/app/[lang]/(authenticated)/org/[organizationSlug]/inbox/_components/inbox-notifications-api";

import { appShellNavigationMessages } from "./app-shell-navigation.messages";
import { OrgNavLink } from "./org-nav-link";
import {
  annotateNavigationItemsWithWorkspaceFlags,
  groupPreviewNavigationGroups,
} from "@/lib/flags/workspace-flag-navigation";
import { formatInboxUnreadBadgeLabel, inboxUnreadBadgeClassName } from "./inbox-unread-badge";

import {
  buildOrganizationPath,
  buildProjectNavigationItems,
  isNavigationItemActive,
  parseProjectRoute,
  type NavigationGroup,
  type NavigationItem,
} from "./navigation-config";
import { useAppShellStore } from "./store/app-shell-store-context";

const inboxNotificationsApi = createInboxNotificationsApi(apiClient);

type AppShellNavigationProps = {
  organizationSlug: string;
};

export const AppShellNavigation = observer(function AppShellNavigation({
  organizationSlug,
}: AppShellNavigationProps) {
  const store = useAppShellStore();
  const pathname = usePathname();
  const projectRoute = parseProjectRoute(pathname);

  if (store.navigation.mode === "custom" && store.navigation.customState) {
    const customState = store.navigation.customState;

    if (customState.projectContext) {
      return (
        <ProjectNavigation
          organizationSlug={customState.projectContext.organizationSlug}
          projectId={customState.projectContext.projectId}
          pathname={pathname}
          projectName={customState.projectContext.projectName}
          items={customState.groups.flatMap((group) => group.items)}
        />
      );
    }

    return (
      <GlobalNavigation
        organizationSlug={organizationSlug}
        groups={customState.groups}
        pathname={pathname}
      />
    );
  }

  if (projectRoute?.organizationSlug === organizationSlug) {
    return (
      <ProjectNavigation
        organizationSlug={organizationSlug}
        projectId={projectRoute.projectId}
        pathname={pathname}
      />
    );
  }

  return (
    <GlobalNavigation
      organizationSlug={organizationSlug}
      groups={store.navigation.defaultNavigationGroups}
      pathname={pathname}
    />
  );
});

function GlobalNavigation({
  organizationSlug,
  groups,
  pathname,
}: {
  organizationSlug: string;
  groups: readonly NavigationGroup[];
  pathname: string;
}) {
  const intl = useIntl();
  const grouped = groupPreviewNavigationGroups(
    groups,
    intl.formatMessage(appShellNavigationMessages.trySection),
  );

  return (
    <div className="flex flex-col gap-3">
      {grouped.map((group, groupIndex) => {
        const content = (
          <NavigationGroupItems
            group={group}
            pathname={pathname}
            organizationSlug={organizationSlug}
          />
        );

        if (!group.label) {
          return (
            <SidebarGroup key={`promoted-${groupIndex}`} className="p-0">
              {content}
            </SidebarGroup>
          );
        }

        return (
          <LabeledNavigationSection key={group.label} label={group.label} offset={groupIndex > 0}>
            {content}
          </LabeledNavigationSection>
        );
      })}
    </div>
  );
}

function ProjectNavigation({
  organizationSlug,
  projectId,
  pathname,
  projectName,
  items,
}: {
  organizationSlug: string;
  projectId: string;
  pathname: string;
  projectName?: string;
  items?: readonly NavigationItem[];
}) {
  const intl = useIntl();
  const projectQuery = useQuery({
    queryKey: ["translation-project", organizationSlug, projectId],
    enabled: !projectName && !items,
    queryFn: async () => {
      const response = await apiClient.api.orgs[":organizationSlug"].projects[":projectId"].$get({
        param: { organizationSlug, projectId },
      });
      if (response.status !== 200) {
        throw new Error(`Failed to load project (${response.status})`);
      }
      const body = await response.json();
      return body.project;
    },
  });

  const store = useAppShellStore();
  const resolvedItems = annotateNavigationItemsWithWorkspaceFlags(
    items ?? buildProjectNavigationItems(organizationSlug, projectId, intl),
    store.workspaceFeatureFlags,
  );
  const tryLabel = intl.formatMessage(appShellNavigationMessages.trySection);
  const grouped = groupPreviewNavigationGroups([{ items: resolvedItems }], tryLabel);
  const tryGroup = grouped.find((group) => group.label === tryLabel);
  const projectItems = grouped.find((group) => group.label !== tryLabel)?.items ?? [];
  const resolvedProjectName =
    projectName ??
    projectQuery.data?.name ??
    intl.formatMessage(appShellNavigationMessages.projectFallbackName);
  const projectsHref = buildOrganizationPath(organizationSlug, "projects");
  const allProjectsLabel = intl.formatMessage(appShellNavigationMessages.allProjects);

  return (
    <div className="flex flex-col gap-3">
      <SidebarGroup className="p-0">
        <SidebarGroupContent>
          <SidebarMenu className="gap-1">
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<OrgNavLink href={projectsHref} />}
                tooltip={allProjectsLabel}
                className="h-8 rounded-md px-2.5 text-sm font-medium text-muted-foreground hover:text-sidebar-foreground group-data-[collapsible=icon]:size-8!"
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} strokeWidth={2} className="size-4" />
                <span>
                  <FormattedMessage {...appShellNavigationMessages.allProjects} />
                </span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <SidebarGroup className="gap-1 p-0">
        <SidebarGroupLabel className="h-auto px-3 py-1 text-xs font-medium tracking-wide text-muted-foreground uppercase group-data-[collapsible=icon]:hidden">
          <FormattedMessage {...appShellNavigationMessages.projectSection} />
        </SidebarGroupLabel>
        <div className="px-3 pb-1 group-data-[collapsible=icon]:hidden">
          {!projectName && projectQuery.isLoading ? (
            <Skeleton className="h-5 w-4/5" />
          ) : (
            <p className="truncate text-sm font-medium text-sidebar-foreground">
              {resolvedProjectName}
            </p>
          )}
        </div>
        <NavigationGroupItems
          group={{ items: projectItems }}
          pathname={pathname}
          organizationSlug={organizationSlug}
          projectId={projectId}
        />
      </SidebarGroup>

      {tryGroup ? (
        <LabeledNavigationSection label={tryLabel} offset>
          <NavigationGroupItems
            group={tryGroup}
            pathname={pathname}
            organizationSlug={organizationSlug}
            projectId={projectId}
          />
        </LabeledNavigationSection>
      ) : null}
    </div>
  );
}

function LabeledNavigationSection({
  label,
  offset = false,
  children,
}: {
  label: string;
  offset?: boolean;
  children: ReactNode;
}) {
  return (
    <Collapsible defaultOpen className={cn(offset && "mt-2")}>
      <SidebarGroup className="p-0">
        <CollapsibleTrigger className="group/collapsible-trigger flex h-7 w-full items-center gap-2 rounded-md px-3 text-left text-xs font-medium tracking-wide text-muted-foreground uppercase outline-hidden transition-[margin,opacity,color] duration-200 hover:text-sidebar-foreground focus-visible:text-sidebar-foreground group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0">
          <span className="min-w-0 flex-1 truncate">{label}</span>
          <HugeiconsIcon
            icon={ArrowDown01Icon}
            strokeWidth={1.8}
            className="size-3.5 shrink-0 transition-transform group-data-panel-open/collapsible-trigger:rotate-180"
          />
        </CollapsibleTrigger>
        <CollapsibleContent hiddenUntilFound>{children}</CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}

function NavigationGroupItems({
  group,
  pathname,
  organizationSlug,
  projectId,
}: {
  group: { items: readonly NavigationItem[] };
  pathname: string;
  organizationSlug: string;
  projectId?: string;
}) {
  const intl = useIntl();
  const inboxHref = buildOrganizationPath(organizationSlug, "inbox");
  const unreadCountQuery = useQuery({
    queryKey: notificationsUnreadCountQueryKey(organizationSlug),
    queryFn: () => inboxNotificationsApi.unreadCount(organizationSlug),
    enabled: Boolean(organizationSlug),
    refetchInterval: 45_000,
  });
  const unreadCount = unreadCountQuery.data ?? 0;
  const unreadBadgeLabel = formatInboxUnreadBadgeLabel(unreadCount);

  return (
    <SidebarGroupContent>
      <SidebarMenu className="gap-1">
        {group.items.map((item) => {
          const isActive = isNavigationItemActive(pathname, item.href, {
            exact: item.exact,
            projectId,
            organizationSlug,
          });
          const isInboxItem = item.href === inboxHref;
          const previewBadgeLabel = item.preview
            ? intl.formatMessage(appShellNavigationMessages.previewBadge)
            : null;
          const dynamicBadge = isInboxItem ? unreadBadgeLabel : null;
          const badge = dynamicBadge ?? previewBadgeLabel ?? item.badge;
          const tooltip = badge
            ? intl.formatMessage(appShellNavigationMessages.badgeSeparator, {
                label: item.label,
                badge,
              })
            : item.label;

          return (
            <SidebarMenuItem key={item.href}>
              <SidebarMenuButton
                render={<OrgNavLink href={item.href} />}
                isActive={isActive}
                tooltip={tooltip}
                className={navigationButtonClass(isActive)}
              >
                <HugeiconsIcon icon={item.icon} strokeWidth={2} className="size-4" />
                <span className="min-w-0 flex-1 truncate">{item.label}</span>
                {badge && !dynamicBadge ? (
                  <span className="ms-auto inline-flex shrink-0 items-center rounded-full border border-sidebar-border bg-sidebar px-1.5 py-0.5 text-[0.625rem] leading-none font-medium tracking-normal text-muted-foreground group-data-[collapsible=icon]:hidden">
                    {badge}
                  </span>
                ) : null}
              </SidebarMenuButton>
              {dynamicBadge ? (
                <SidebarMenuBadge className={inboxUnreadBadgeClassName}>
                  {dynamicBadge}
                </SidebarMenuBadge>
              ) : null}
            </SidebarMenuItem>
          );
        })}
      </SidebarMenu>
    </SidebarGroupContent>
  );
}

function navigationButtonClass(isActive: boolean) {
  return cn(
    "h-8 rounded-md px-2.5 text-sm font-medium text-muted-foreground hover:text-sidebar-foreground group-data-[collapsible=icon]:size-8!",
    isActive && "bg-sidebar-accent font-semibold text-sidebar-accent-foreground",
  );
}

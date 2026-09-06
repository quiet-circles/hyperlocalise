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
import { type CSSProperties, type ReactNode } from "react";
import Image from "next/image";
import { FormattedMessage, useIntl, type IntlShape } from "react-intl";

import { Button } from "@/components/ui/button";
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";
import { TypographyP } from "@/components/ui/typography";
import { getIntlShape } from "@/lib/app-i18n/intl";
import { annotateNavigationByWorkspaceFlags } from "@/lib/flags/workspace-flag-navigation";
import type { WorkspaceFeatureFlagState } from "@/lib/flags/workos-flag-entities";
import type { TmsUserConnectCta } from "@/lib/providers/credentials/tms-user-connection-shared";

import { appShellClientMessages } from "./app-shell-client.messages";
import { AppShellClient } from "./app-shell-client";
import { AppShellBreadcrumb } from "./app-shell-breadcrumb";
import { AppShellNavigation } from "./app-shell-navigation";
import { buildGlobalNavigationGroups, buildProjectNavigationItems } from "./navigation-config";
import { NavUser } from "./nav-user";
import { AppShellHeaderActions } from "./store/app-shell-header-actions";
import { AppShellStoreProvider } from "./store/app-shell-store-context";
import { SidebarStoreBridge } from "./store/sidebar-store-bridge";
import { useAppShellHeaderAction } from "./store/use-app-shell-header-action";
import { useAppShellNavigationCustom } from "./store/use-app-shell-navigation";
import { TmsUserConnectButton } from "./tms-user-connect-button";

export const APP_SHELL_STORY_ORGANIZATION_SLUG = "acme";
export const APP_SHELL_STORY_PROJECT_ID = "project_website";

export const appShellStoryUser = {
  name: "Minh Cung",
  email: "minh.cung@example.com",
  avatarUrl: undefined,
};

export const appShellStoryOrganizations = [
  { name: "Acme Localization", slug: APP_SHELL_STORY_ORGANIZATION_SLUG },
  { name: "Beta Workspace", slug: "beta" },
] as const;

export const appShellStoryTmsConnectCta: TmsUserConnectCta = {
  showConnectCta: true,
  providerKind: "crowdin",
  providerDisplayName: "Crowdin",
  connectMethod: "oauth",
};

export const ALL_WORKSPACE_FEATURE_FLAGS: WorkspaceFeatureFlagState = {
  automations: true,
  knowledge: true,
  visualMock: true,
  visualWorkflows: true,
  domains: true,
  glossarySearch: true,
  hyperlab: true,
  reports: true,
};

export function buildAppShellStoryNavigationGroups(locale: string = "en") {
  const intl = getIntlShape(locale) as IntlShape;
  return annotateNavigationByWorkspaceFlags(
    buildGlobalNavigationGroups(APP_SHELL_STORY_ORGANIZATION_SLUG, intl),
    ALL_WORKSPACE_FEATURE_FLAGS,
  );
}

export function buildAppShellStoryProjectNavigationGroups(locale: string = "en") {
  const intl = getIntlShape(locale) as IntlShape;
  return [
    {
      items: buildProjectNavigationItems(
        APP_SHELL_STORY_ORGANIZATION_SLUG,
        APP_SHELL_STORY_PROJECT_ID,
        intl,
      ),
    },
  ];
}

const appShellStoryLayoutStyle = {
  "--app-shell-content-height":
    "calc(100svh - var(--app-shell-header-height) - var(--app-shell-footer-height))",
  "--app-shell-plan-footer-height": "calc(3rem + env(safe-area-inset-bottom))",
  "--app-shell-footer-height":
    "calc(var(--app-shell-plan-footer-height) + var(--app-shell-dock-height, 0px))",
  "--sidebar-width": "15rem",
} as CSSProperties;

export function AppShellStoryPlaceholder() {
  return (
    <div className="rounded-2xl border border-dashed border-border bg-muted/20 p-8">
      <p className="text-sm text-muted-foreground">
        Page content renders in this area. Use the shell chrome around it to preview navigation,
        header actions, and footer states.
      </p>
    </div>
  );
}

type AppShellStoryFrameProps = {
  children?: ReactNode;
  navigationGroups?: ReturnType<typeof buildAppShellStoryNavigationGroups>;
  showApiKeysLink?: boolean;
  showBillingLink?: boolean;
  showMembersLink?: boolean;
  autumnConfigured?: boolean;
  tmsUserConnectCta?: TmsUserConnectCta;
};

export function AppShellStoryFrame({
  children,
  navigationGroups = buildAppShellStoryNavigationGroups(),
  showApiKeysLink = true,
  showBillingLink = true,
  showMembersLink = true,
  autumnConfigured = false,
  tmsUserConnectCta = { showConnectCta: false },
}: AppShellStoryFrameProps) {
  return (
    <AppShellClient
      activeOrganization={{
        name: "Acme Localization",
        slug: APP_SHELL_STORY_ORGANIZATION_SLUG,
      }}
      navigationGroups={navigationGroups}
      organizations={[...appShellStoryOrganizations]}
      showApiKeysLink={showApiKeysLink}
      showBillingLink={showBillingLink}
      showMembersLink={showMembersLink}
      autumnConfigured={autumnConfigured}
      tmsUserConnectCta={tmsUserConnectCta}
      user={appShellStoryUser}
      workspaceFeatureFlags={ALL_WORKSPACE_FEATURE_FLAGS}
    >
      {children ?? <AppShellStoryPlaceholder />}
    </AppShellClient>
  );
}

function AppShellHeaderStoryBar({
  showTmsConnect = false,
  showAdminLinks = true,
}: {
  showTmsConnect?: boolean;
  showAdminLinks?: boolean;
}) {
  return (
    <div className="sticky top-0 z-20 shrink-0 border-b border-border bg-background/96 backdrop-blur">
      <div className="flex h-(--app-shell-header-height) items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <SidebarTrigger className="-ms-1" />
          <Separator
            orientation="vertical"
            className="me-2 data-vertical:h-4 data-vertical:self-auto"
          />
          <AppShellBreadcrumb organizationSlug={APP_SHELL_STORY_ORGANIZATION_SLUG} />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <AppShellHeaderActions />
          {showTmsConnect ? (
            <TmsUserConnectButton
              organizationSlug={APP_SHELL_STORY_ORGANIZATION_SLUG}
              providerKind="crowdin"
              providerDisplayName="Crowdin"
              connectMethod="oauth"
            />
          ) : null}
          <NavUser
            organizationSlug={APP_SHELL_STORY_ORGANIZATION_SLUG}
            organizations={[...appShellStoryOrganizations]}
            showApiKeysLink={showAdminLinks}
            showBillingLink={showAdminLinks}
            showMembersLink={showAdminLinks}
            user={{
              name: appShellStoryUser.name,
              email: appShellStoryUser.email,
              avatar: appShellStoryUser.avatarUrl ?? "",
            }}
          />
        </div>
      </div>
    </div>
  );
}

export function AppShellHeaderStoryFrame({
  children,
  showTmsConnect = false,
  showAdminLinks = true,
}: {
  children?: ReactNode;
  showTmsConnect?: boolean;
  showAdminLinks?: boolean;
}) {
  return (
    <AppShellStoreProvider defaultNavigationGroups={buildAppShellStoryNavigationGroups()}>
      <SidebarProvider
        defaultOpen
        style={appShellStoryLayoutStyle}
        className="min-h-[16rem] bg-background"
      >
        <SidebarStoreBridge />
        <Sidebar variant="sidebar" collapsible="icon" className="hidden" />
        <SidebarInset className="min-h-[16rem] bg-background">
          <AppShellHeaderStoryBar showTmsConnect={showTmsConnect} showAdminLinks={showAdminLinks} />
          <div className="px-4 py-5 sm:px-6 lg:px-8">
            {children ?? <AppShellStoryPlaceholder />}
          </div>
        </SidebarInset>
      </SidebarProvider>
    </AppShellStoreProvider>
  );
}

function ProjectSidebarStorySetup() {
  useAppShellNavigationCustom({
    groups: buildAppShellStoryProjectNavigationGroups(),
    projectContext: {
      organizationSlug: APP_SHELL_STORY_ORGANIZATION_SLUG,
      projectId: APP_SHELL_STORY_PROJECT_ID,
      projectName: "Website localization",
    },
  });

  return null;
}

export function AppShellSidebarStoryFrame({
  collapsed = false,
  variant = "global",
}: {
  collapsed?: boolean;
  variant?: "global" | "project";
}) {
  const intl = useIntl();

  return (
    <AppShellStoreProvider defaultNavigationGroups={buildAppShellStoryNavigationGroups()}>
      <SidebarProvider
        defaultOpen={!collapsed}
        style={appShellStoryLayoutStyle}
        className="min-h-[36rem] rounded-2xl border bg-background text-foreground"
      >
        <SidebarStoreBridge />
        {variant === "project" ? <ProjectSidebarStorySetup /> : null}
        <Sidebar variant="sidebar" collapsible="icon">
          <SidebarHeader className="gap-3 border-b border-sidebar-border px-3 py-3 group-data-[collapsible=icon]:items-center group-data-[collapsible=icon]:px-0">
            <div className="flex items-center gap-2.5 rounded-xl px-1 py-1 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
              <Image
                src="/images/logo.png"
                width={28}
                height={28}
                sizes="28px"
                alt={intl.formatMessage(appShellClientMessages.logoAlt)}
                className="size-7 shrink-0 rounded-lg"
              />
              <div className="min-w-0 group-data-[collapsible=icon]:hidden">
                <TypographyP
                  className="text-sidebar-foreground"
                  lineClamp={1}
                  size="small"
                  weight="medium"
                >
                  <FormattedMessage {...appShellClientMessages.brandName} />
                </TypographyP>
              </div>
            </div>
          </SidebarHeader>
          <SidebarContent className="gap-0 px-2 pt-2">
            <AppShellNavigation organizationSlug={APP_SHELL_STORY_ORGANIZATION_SLUG} />
          </SidebarContent>
          <SidebarRail />
        </Sidebar>
        <SidebarInset className="p-6">
          <SidebarTrigger />
          <p className="mt-4 text-sm text-muted-foreground">
            Sidebar preview. Toggle collapse with the trigger above or the rail on the left.
          </p>
        </SidebarInset>
      </SidebarProvider>
    </AppShellStoreProvider>
  );
}

export function AppShellHeaderActionDemo() {
  useAppShellHeaderAction({
    id: "storybook-save",
    render: () => (
      <Button size="sm" type="button">
        Save changes
      </Button>
    ),
  });

  return null;
}

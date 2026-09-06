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
import type { CSSProperties, ReactNode } from "react";
import Image from "next/image";
import { usePathname } from "next/navigation";
import { FormattedMessage, useIntl } from "react-intl";

import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { cn } from "@/lib/primitives/cn";
import { AppShellBreadcrumb } from "./app-shell-breadcrumb";
import { AppShellNavigation } from "./app-shell-navigation";
import { TmsUserConnectButton } from "./tms-user-connect-button";
import { TmsUserOAuthErrorToast } from "./tms-user-oauth-error-toast";
import {
  isOrganizationSettingsPath,
  parseProjectRoute,
  type NavigationGroup,
} from "./navigation-config";
import { AppShellHeaderActions } from "./store/app-shell-header-actions";
import { AppShellStoreProvider } from "./store/app-shell-store-context";
import { SidebarStoreBridge } from "./store/sidebar-store-bridge";
import type { WorkspaceFeatureFlagState } from "@/lib/flags/workos-flag-entities";
import type { TmsUserConnectCta } from "@/lib/providers/credentials/tms-user-connection-shared";
import { useTmsUserConnectCta } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/_hooks/use-tms-user-connect-cta";
import { NavUser } from "./nav-user";
import { Separator } from "@/components/ui/separator";
import { TypographyP } from "@/components/ui/typography";
import { AppShellFooter } from "./app-shell-footer";

import { appShellClientMessages } from "./app-shell-client.messages";

type AppShellClientProps = {
  autumnConfigured?: boolean;
  children: ReactNode;
  navigationGroups: readonly NavigationGroup[];
  workspaceFeatureFlags: WorkspaceFeatureFlagState;
  activeOrganization: {
    name: string;
    slug?: string | null;
  };
  organizations: Array<{
    name: string;
    slug?: string | null;
  }>;
  tmsUserConnectCta?: TmsUserConnectCta;
  showApiKeysLink?: boolean;
  showBillingLink?: boolean;
  showMembersLink?: boolean;
  user: {
    name: string;
    email: string;
    avatarUrl?: string;
  };
};

export function AppShellClient({
  autumnConfigured = false,
  children,
  navigationGroups,
  workspaceFeatureFlags,
  activeOrganization,
  organizations,
  tmsUserConnectCta = { showConnectCta: false },
  showApiKeysLink = false,
  showBillingLink = false,
  showMembersLink = false,
  user,
}: AppShellClientProps) {
  const intl = useIntl();
  const pathname = usePathname();
  const organizationSlug = activeOrganization.slug ?? "";
  const projectRoute = parseProjectRoute(pathname);
  const isContentEditorWorkspaceRoute =
    pathname.includes("/strings") || pathname.includes("/files/content-editor");
  const isOrgSettingsRoute = isOrganizationSettingsPath(pathname);
  const tmsUserConnectQuery = useTmsUserConnectCta(organizationSlug, {
    enabled: Boolean(organizationSlug),
    initialData: tmsUserConnectCta,
  });
  const resolvedTmsUserConnectCta = tmsUserConnectQuery.data ?? tmsUserConnectCta;

  return (
    <AppShellStoreProvider
      defaultNavigationGroups={navigationGroups}
      workspaceFeatureFlags={workspaceFeatureFlags}
    >
      <TmsUserOAuthErrorToast />
      <SidebarProvider
        defaultOpen
        style={
          {
            "--app-shell-content-height":
              "calc(100svh - var(--app-shell-header-height) - var(--app-shell-footer-height))",
            "--app-shell-plan-footer-height": "calc(3rem + env(safe-area-inset-bottom))",
            "--app-shell-footer-height":
              "calc(var(--app-shell-plan-footer-height) + var(--app-shell-dock-height, 0px))",
            "--sidebar-width": "15rem",
          } as CSSProperties
        }
        className="min-h-svh bg-background text-foreground"
      >
        <SidebarStoreBridge />
        <Sidebar
          variant="sidebar"
          collapsible="icon"
          style={{ viewTransitionName: "app-shell-sidebar" }}
        >
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

          <SidebarContent className="gap-0 px-2 pt-2 pb-[var(--app-shell-footer-height)]">
            <AppShellNavigation organizationSlug={organizationSlug} />
          </SidebarContent>

          <SidebarRail />
        </Sidebar>

        <SidebarInset className="h-svh max-h-svh min-h-0 overflow-hidden bg-background pb-[var(--app-shell-footer-height)]">
          <div
            className="sticky top-0 z-20 shrink-0 border-b border-border bg-background/96 backdrop-blur"
            style={{ viewTransitionName: "app-shell-header" }}
          >
            <div className="flex h-(--app-shell-header-height) items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
              <div className="flex min-w-0 items-center gap-3">
                <SidebarTrigger className="-ms-1" />
                <Separator
                  orientation="vertical"
                  className="me-2 data-vertical:h-4 data-vertical:self-auto"
                />
                <AppShellBreadcrumb organizationSlug={organizationSlug} />
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <AppShellHeaderActions />
                {resolvedTmsUserConnectCta.showConnectCta && organizationSlug ? (
                  <TmsUserConnectButton
                    organizationSlug={organizationSlug}
                    providerKind={resolvedTmsUserConnectCta.providerKind}
                    providerDisplayName={resolvedTmsUserConnectCta.providerDisplayName}
                    connectMethod={resolvedTmsUserConnectCta.connectMethod}
                  />
                ) : null}
                <NavUser
                  organizationSlug={activeOrganization.slug ?? ""}
                  organizations={organizations}
                  showApiKeysLink={showApiKeysLink}
                  showBillingLink={showBillingLink}
                  showMembersLink={showMembersLink}
                  user={{
                    name: user.name,
                    email: user.email,
                    avatar: user.avatarUrl ?? "",
                  }}
                />
              </div>
            </div>
          </div>

          <div
            className={cn(
              "flex min-h-0 flex-1 flex-col",
              isOrgSettingsRoute ? "overflow-hidden" : "overflow-y-auto px-4 py-5 sm:px-6 lg:px-8",
            )}
          >
            {children}
          </div>
        </SidebarInset>

        <AppShellFooter
          organizationSlug={organizationSlug}
          projectId={projectRoute?.projectId ?? null}
          showPlan={showBillingLink && autumnConfigured}
          showGlossaryGuidance={isContentEditorWorkspaceRoute}
          showIssueGuidance={isContentEditorWorkspaceRoute}
          showStyleGuide={isContentEditorWorkspaceRoute}
          currentUser={
            organizationSlug
              ? {
                  avatarUrl: user.avatarUrl ?? null,
                  email: user.email,
                  name: user.name,
                }
              : undefined
          }
        />
      </SidebarProvider>
    </AppShellStoreProvider>
  );
}

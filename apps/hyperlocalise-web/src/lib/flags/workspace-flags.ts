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
import { redirect } from "next/navigation";
import { flag, type Flag } from "flags/next";
import { and, eq } from "drizzle-orm";

import { db, schema } from "@/lib/database/client";
import type { AppAuthContext } from "@/lib/workos/app-auth";

import { createWorkosIdentify } from "./identify-workos-context";
import { workosAdapter } from "./workos-adapter";
import {
  WORKSPACE_AUTOMATIONS_FLAG,
  WORKSPACE_DOMAINS_FLAG,
  WORKSPACE_FEATURE_UNAVAILABLE_REASON,
  WORKSPACE_GLOSSARY_SEARCH_FLAG,
  WORKSPACE_HYPERLAB_FLAG,
  WORKSPACE_KNOWLEDGE_FLAG,
  WORKSPACE_REPORTS_FLAG,
  WORKSPACE_VISUAL_MOCK_FLAG,
  WORKSPACE_VISUAL_WORKFLOWS_FLAG,
  type WorkosFlagEntities,
  type WorkspaceFeatureFlagState,
} from "./workos-flag-entities";

export {
  annotateNavigationByWorkspaceFlags,
  annotateNavigationItemsWithWorkspaceFlags,
  filterNavigationByWorkspaceFlags,
  filterNavigationItemsByWorkspaceFlags,
  groupPreviewNavigationGroups,
} from "./workspace-flag-navigation";

export const workspaceAutomationsFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_AUTOMATIONS_FLAG,
  defaultValue: false,
  description: "Workspace automations for scheduled and GitHub-triggered workflows.",
  adapter: workosAdapter(),
});

export const workspaceKnowledgeFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_KNOWLEDGE_FLAG,
  defaultValue: false,
  description: "Workspace knowledge memory for agents and teams.",
  adapter: workosAdapter(),
});

export const workspaceVisualMockFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_VISUAL_MOCK_FLAG,
  defaultValue: false,
  description: "Visual mock skill for repository-backed Hyperlocalise agent previews.",
  adapter: workosAdapter(),
});

export const workspaceVisualWorkflowsFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_VISUAL_WORKFLOWS_FLAG,
  defaultValue: false,
  description: "Advanced visual workflow editor for deterministic automation graphs.",
  adapter: workosAdapter(),
});

export const workspaceDomainsFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_DOMAINS_FLAG,
  defaultValue: false,
  description: "Workspace domains for claimed sites and localisation audit reports.",
  adapter: workosAdapter(),
});

export const workspaceGlossarySearchFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_GLOSSARY_SEARCH_FLAG,
  defaultValue: false,
  description:
    "Conversational agent glossary search (native and Crowdin) for terminology-safe advice.",
  adapter: workosAdapter(),
});

export const workspaceHyperlabFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_HYPERLAB_FLAG,
  defaultValue: false,
  description: "Hyperlab experiments and feature flags for customer applications.",
  adapter: workosAdapter(),
});

export const workspaceReportsFlag = flag<boolean, WorkosFlagEntities>({
  key: WORKSPACE_REPORTS_FLAG,
  defaultValue: false,
  description: "Workspace translation reports for word counts, time, and cost.",
  adapter: workosAdapter(),
});

export async function evaluateWorkspaceFeatureFlags(
  auth: Pick<AppAuthContext, "activeOrganization" | "user">,
): Promise<WorkspaceFeatureFlagState> {
  const identify = () => createWorkosIdentify(auth);

  const [
    automations,
    knowledge,
    visualMock,
    visualWorkflows,
    domains,
    glossarySearch,
    hyperlab,
    reports,
  ] = await Promise.all([
    workspaceAutomationsFlag.run({ identify }),
    workspaceKnowledgeFlag.run({ identify }),
    workspaceVisualMockFlag.run({ identify }),
    workspaceVisualWorkflowsFlag.run({ identify }),
    workspaceDomainsFlag.run({ identify }),
    workspaceGlossarySearchFlag.run({ identify }),
    workspaceHyperlabFlag.run({ identify }),
    workspaceReportsFlag.run({ identify }),
  ]);

  return {
    automations,
    knowledge,
    visualMock,
    visualWorkflows,
    domains,
    glossarySearch,
    hyperlab,
    reports,
  };
}

export async function resolveWorkspaceVisualMockFlag(input: {
  organizationId: string;
  localUserId: string;
  dbClient?: Pick<typeof db, "select">;
}) {
  const dbClient = input.dbClient ?? db;
  if (typeof dbClient.select !== "function") {
    return false;
  }

  try {
    const [identity] = await dbClient
      .select({
        workosOrganizationId: schema.organizations.workosOrganizationId,
        workosUserId: schema.users.workosUserId,
      })
      .from(schema.organizationMemberships)
      .innerJoin(
        schema.organizations,
        eq(schema.organizations.id, schema.organizationMemberships.organizationId),
      )
      .innerJoin(schema.users, eq(schema.users.id, schema.organizationMemberships.userId))
      .where(
        and(
          eq(schema.organizationMemberships.organizationId, input.organizationId),
          eq(schema.organizationMemberships.userId, input.localUserId),
        ),
      )
      .limit(1);

    if (!identity) {
      return false;
    }

    return workspaceVisualMockFlag.run({
      identify: () => ({
        organization: { id: identity.workosOrganizationId },
        user: { id: identity.workosUserId },
      }),
    });
  } catch {
    return false;
  }
}

/**
 * Resolves workspaceKnowledgeFlag from just an internal organizationId, for callers with no live
 * HTTP auth context to build a WorkOS identify() from — background jobs, queue workers, and
 * workspace-automation tool calls. Mirrors resolveWorkspaceVisualMockFlag's DB-join-then-run-flag
 * shape, minus the user lookup those callers don't have either.
 *
 * knowledge-memory.route.ts already rejects every request when this flag is off, but that check
 * lives only on the human-facing HTTP route — nothing stopped an automation's stored toolConfig
 * (set once, possibly before the flag was disabled, or via direct API access that skips UI
 * validation) from reaching save_memory/recall_memory and mutating or reading Memory.md on a
 * schedule regardless of the flag. Call this at the point of use, not just at config-save time, so
 * disabling the flag actually stops in-flight automations too.
 */
export async function resolveWorkspaceKnowledgeFlag(input: {
  organizationId: string;
  dbClient?: Pick<typeof db, "select">;
}): Promise<boolean> {
  const dbClient = input.dbClient ?? db;
  if (typeof dbClient.select !== "function") {
    return false;
  }

  try {
    const [organization] = await dbClient
      .select({ workosOrganizationId: schema.organizations.workosOrganizationId })
      .from(schema.organizations)
      .where(eq(schema.organizations.id, input.organizationId))
      .limit(1);

    if (!organization) {
      return false;
    }

    return (
      (await workspaceKnowledgeFlag.run({
        identify: () => ({ organization: { id: organization.workosOrganizationId } }),
      })) === true
    );
  } catch {
    return false;
  }
}

export async function getWorkspaceFeatureFlagEnabled(
  workspaceFlag: Flag<boolean, WorkosFlagEntities>,
  auth: Pick<AppAuthContext, "activeOrganization" | "user">,
): Promise<boolean> {
  try {
    return (await workspaceFlag.run({ identify: () => createWorkosIdentify(auth) })) === true;
  } catch {
    return false;
  }
}

export async function requireWorkspaceFeatureFlag(
  workspaceFlag: Flag<boolean, WorkosFlagEntities>,
  auth: Pick<AppAuthContext, "activeOrganization" | "user">,
) {
  const enabled = await workspaceFlag.run({ identify: () => createWorkosIdentify(auth) });

  if (enabled) {
    return;
  }

  const organizationSlug = auth.activeOrganization.slug;
  if (organizationSlug) {
    redirect(`/org/${organizationSlug}/dashboard?reason=${WORKSPACE_FEATURE_UNAVAILABLE_REASON}`);
  }

  redirect("/auth/select-organization");
}

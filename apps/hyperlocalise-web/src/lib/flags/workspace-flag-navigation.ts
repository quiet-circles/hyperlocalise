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
import type { NavigationGroup, NavigationItem } from "@/components/app-shell/navigation-config";

import {
  WORKSPACE_AUTOMATIONS_FLAG,
  WORKSPACE_DOMAINS_FLAG,
  WORKSPACE_GLOSSARY_SEARCH_FLAG,
  WORKSPACE_HYPERLAB_FLAG,
  WORKSPACE_KNOWLEDGE_FLAG,
  WORKSPACE_REPORTS_FLAG,
  WORKSPACE_VISUAL_MOCK_FLAG,
  WORKSPACE_VISUAL_WORKFLOWS_FLAG,
  type WorkspaceFeatureFlagState,
} from "./workos-flag-entities";

function workspaceFlagEnabledByKey(flags: WorkspaceFeatureFlagState): Record<string, boolean> {
  return {
    [WORKSPACE_AUTOMATIONS_FLAG]: flags.automations,
    [WORKSPACE_KNOWLEDGE_FLAG]: flags.knowledge,
    [WORKSPACE_VISUAL_MOCK_FLAG]: flags.visualMock,
    [WORKSPACE_VISUAL_WORKFLOWS_FLAG]: flags.visualWorkflows,
    [WORKSPACE_DOMAINS_FLAG]: flags.domains,
    [WORKSPACE_GLOSSARY_SEARCH_FLAG]: flags.glossarySearch,
    [WORKSPACE_HYPERLAB_FLAG]: flags.hyperlab,
    [WORKSPACE_REPORTS_FLAG]: flags.reports,
  };
}

function annotateNavigationItemWithWorkspaceFlags(
  item: NavigationItem,
  enabledByKey: Record<string, boolean>,
): NavigationItem {
  if (!item.featureFlagKey || !(item.featureFlagKey in enabledByKey)) {
    return item;
  }

  const enabled = enabledByKey[item.featureFlagKey] ?? false;
  if (enabled) {
    return { ...item, preview: false };
  }

  return { ...item, preview: true };
}

export function annotateNavigationItemsWithWorkspaceFlags(
  items: readonly NavigationItem[],
  flags: WorkspaceFeatureFlagState,
): readonly NavigationItem[] {
  const enabledByKey = workspaceFlagEnabledByKey(flags);
  return items.map((item) => annotateNavigationItemWithWorkspaceFlags(item, enabledByKey));
}

export function annotateNavigationByWorkspaceFlags(
  groups: readonly NavigationGroup[],
  flags: WorkspaceFeatureFlagState,
): readonly NavigationGroup[] {
  const enabledByKey = workspaceFlagEnabledByKey(flags);

  return groups.map((group) => ({
    ...group,
    items: group.items.map((item) => annotateNavigationItemWithWorkspaceFlags(item, enabledByKey)),
  }));
}

export function groupPreviewNavigationGroups(
  groups: readonly NavigationGroup[],
  tryLabel: string,
): readonly NavigationGroup[] {
  const previewItems: NavigationItem[] = [];
  const remainingGroups = groups
    .map((group) => {
      const items = group.items.filter((item) => {
        if (item.preview) {
          previewItems.push(item);
          return false;
        }

        return true;
      });

      return { ...group, items };
    })
    .filter((group) => group.items.length > 0);

  if (previewItems.length === 0) {
    return remainingGroups;
  }

  return [...remainingGroups, { label: tryLabel, items: previewItems }];
}

/** @deprecated Use annotateNavigationItemsWithWorkspaceFlags — kept for tests during migration. */
export function filterNavigationItemsByWorkspaceFlags(
  items: readonly NavigationItem[],
  flags: WorkspaceFeatureFlagState,
): readonly NavigationItem[] {
  const enabledByKey = workspaceFlagEnabledByKey(flags);
  return items.filter((item) => {
    if (!item.featureFlagKey) {
      return true;
    }

    if (!(item.featureFlagKey in enabledByKey)) {
      return true;
    }

    return enabledByKey[item.featureFlagKey] ?? false;
  });
}

/** @deprecated Use annotateNavigationByWorkspaceFlags — kept for tests during migration. */
export function filterNavigationByWorkspaceFlags(
  groups: readonly NavigationGroup[],
  flags: WorkspaceFeatureFlagState,
): readonly NavigationGroup[] {
  const enabledByKey = workspaceFlagEnabledByKey(flags);

  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (!item.featureFlagKey) {
          return true;
        }

        if (!(item.featureFlagKey in enabledByKey)) {
          return true;
        }

        return enabledByKey[item.featureFlagKey] ?? false;
      }),
    }))
    .filter((group) => group.items.length > 0);
}

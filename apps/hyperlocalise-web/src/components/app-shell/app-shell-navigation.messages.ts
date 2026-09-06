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
import { defineMessages } from "react-intl";

export const appShellNavigationMessages = defineMessages({
  allProjects: {
    defaultMessage: "All projects",
    id: "SUXI1fXBfn",
    description: "Sidebar link to return from a project to the projects list",
  },
  projectSection: {
    defaultMessage: "Project",
    id: "73pQzYGwei",
    description: "Sidebar section label above the current project name and project nav items",
  },
  projectFallbackName: {
    defaultMessage: "Project",
    id: "E2040akAix",
    description: "Fallback project name in the sidebar while the project is loading",
  },
  badgeSeparator: {
    defaultMessage: "{label} · {badge}",
    id: "90L8I4Y/Gl",
    description: "Sidebar tooltip combining a navigation item label and its badge",
  },
  previewBadge: {
    defaultMessage: "Preview",
    id: "qiPX0q5vvf",
    description: "Sidebar badge for gated features that link to teaser pages",
  },
  trySection: {
    defaultMessage: "Try",
    id: "cCjEeqCM8J",
    description: "Sidebar group label for gated features that are not yet enabled",
  },
});

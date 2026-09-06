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

export const projectSettingsPageContentMessages = defineMessages({
  emptyValue: {
    defaultMessage: "—",
    id: "WUYpu2cx7T",
    description: "Placeholder when a project settings detail value is missing",
  },
  sourceConnectionTitle: {
    defaultMessage: "Source connection",
    id: "s0zzQZDyx9",
    description: "Heading for the external TMS source connection section",
  },
  sourceConnectionDescription: {
    defaultMessage:
      "External TMS projects inherit source data and locales from the connected provider.",
    id: "P8FDaqATCf",
    description: "Description for the external TMS source connection section",
  },
  openInProvider: {
    defaultMessage: "Open in provider",
    id: "pk2cfeU9zV",
    description: "Button to open the connected TMS project in the provider",
  },
  saving: {
    defaultMessage: "Saving...",
    id: "r8NvhQi+Bk",
    description: "Save button label while project settings are saving",
  },
  saveSettings: {
    defaultMessage: "Save settings",
    id: "fEG/yqFpT9",
    description: "Save button label for project settings",
  },
  loading: {
    defaultMessage: "Loading project settings...",
    id: "YwhxAul31F",
    description: "Loading state for the project settings page",
  },
  loadError: {
    defaultMessage: "Failed to load project settings.",
    id: "OxbEfuCKvL",
    description: "Error state when project settings fail to load",
  },
  generalTitle: {
    defaultMessage: "General",
    id: "riG8529T+y",
    description: "Heading for the general project settings section",
  },
  generalDescription: {
    defaultMessage: "Name the project and capture operational notes for the team.",
    id: "anx6vfTB7A",
    description: "Description for the general project settings section",
  },
  readOnly: {
    defaultMessage: "Read-only",
    id: "N14/CQLyYb",
    description: "Badge shown when a project settings section cannot be edited",
  },
  nameLabel: {
    defaultMessage: "Name",
    id: "6Lz1ZPrrNj",
    description: "Label for the project name field",
  },
  identifierLabel: {
    defaultMessage: "Identifier",
    id: "WqXZ03MFeR",
    description: "Label for the project issue ID prefix field",
  },
  identifierHelp: {
    defaultMessage:
      "Used as the prefix for new issue IDs (e.g. HL-12). Must be unique among projects in this workspace. Existing issue IDs are kept.",
    id: "o/5bDmAGTd",
    description: "Help text under the project issue identifier field",
  },
  descriptionLabel: {
    defaultMessage: "Description",
    id: "Vd2zLJN2br",
    description: "Label for the project description field",
  },
  descriptionHelp: {
    defaultMessage: "Use this for project scope, release, and ownership notes.",
    id: "gCJJUH5CEa",
    description: "Help text under the project description field",
  },
  styleGuideTitle: {
    defaultMessage: "Style guide",
    id: "iL9/D33IcM",
    description: "Heading for the project style guide settings section",
  },
  styleGuideDescription: {
    defaultMessage:
      "Tone, terminology, and formatting rules for translators and agents. Write this as markdown.",
    id: "DR8B4GRFYT",
    description: "Description for the project style guide settings section",
  },
  styleGuideLabel: {
    defaultMessage: "Style guide",
    id: "/uFRS+73Kw",
    description: "Accessible label for the project style guide editor",
  },
  styleGuidePlaceholder: {
    defaultMessage:
      "Keep product names in English. Use concise UI copy. Prefer an informal tone for marketing pages.",
    id: "k2nQ8sL4pR",
    description: "Placeholder shown in the empty project style guide editor",
  },
  localesTitle: {
    defaultMessage: "Locales",
    id: "gpourmIR9e",
    description: "Heading for the project locales settings section",
  },
  localesEditableDescription: {
    defaultMessage: "Edit the source locale and target locales for this native project.",
    id: "advxgHaDz+",
    description: "Description when project locales can be edited",
  },
  localesReadOnlyDescription: {
    defaultMessage: "Locales are managed by the connected TMS provider.",
    id: "5r0yAGU35E",
    description: "Description when project locales are provider-managed",
  },
});

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

export const contentEditorStyleGuideSheetMessages = defineMessages({
  title: {
    defaultMessage: "Style guide",
    id: "yKFLGCc18p",
    description: "Title for the content editor style guide sheet",
  },
  description: {
    defaultMessage: "Project tone, terminology, and formatting for translators and agents.",
    id: "MTwF8eOJic",
    description: "Description for the content editor style guide sheet",
  },
  loading: {
    defaultMessage: "Loading style guide...",
    id: "pqhSI7L7NV",
    description: "Loading state for the content editor style guide sheet",
  },
  loadError: {
    defaultMessage: "Unable to load the style guide.",
    id: "AJdf6PzoHW",
    description: "Error state when the content editor style guide fails to load",
  },
  empty: {
    defaultMessage: "No style guide yet. Add one in project settings.",
    id: "UlnuVlbp4t",
    description: "Empty state when the project has no style guide",
  },
  editInSettings: {
    defaultMessage: "Edit in settings",
    id: "HCwou8cplr",
    description: "Link from the style guide sheet to project settings",
  },
});

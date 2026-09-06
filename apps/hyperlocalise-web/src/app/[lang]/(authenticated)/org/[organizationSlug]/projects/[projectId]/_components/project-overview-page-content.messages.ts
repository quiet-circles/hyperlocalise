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

export const projectOverviewPageContentMessages = defineMessages({
  createJob: {
    defaultMessage: "Create job",
    id: "7WTIzTDLbF",
    description: "Button on project overview to open the create job dialog",
  },
  openEditor: {
    defaultMessage: "Open Editor",
    id: "LPgS5mUnwV",
    description: "Button on project overview linking to the project Content Editor page",
  },
  projectOverviewFallbackTitle: {
    defaultMessage: "Project overview",
    id: "mehTxJuesM",
    description: "Fallback project overview heading when project details fail to load",
  },
  projectFallbackName: {
    defaultMessage: "Project",
    id: "NhoTfXzaHI",
    description: "Fallback project name when the project title is missing",
  },
  loadProjectError: {
    defaultMessage: "Unable to load project details. Refresh the page or try again in a moment.",
    id: "KjJZbmRFki",
    description: "Error message when project overview details fail to load",
  },
  defaultProjectDescription: {
    defaultMessage: "Today’s queue for this project.",
    id: "6lH73749rG",
    description: "Fallback project description on the project overview page",
  },
  needsYouNowTitle: {
    defaultMessage: "Needs you now",
    id: "dGhvlRGto0",
    description: "Heading for the project overview triage band",
  },
  needsYouNowCount: {
    defaultMessage: "{count, plural, one {# item} other {# items}}",
    id: "obkqu+QKmi",
    description: "Count badge for triage items on project overview",
  },
  triageEmptyTitle: {
    defaultMessage: "No reviews waiting",
    id: "TVxvj2B1bk",
    description: "Title when the triage band has no urgent items",
  },
  triageEmptyDescription: {
    defaultMessage: "Open Files for coverage, or create a job when you are ready.",
    id: "85veKzf/Q7",
    description: "Description when the triage band has no urgent items",
  },
  reviewCta: {
    defaultMessage: "Review",
    id: "v0Xmk6UQW5",
    description: "CTA for a job waiting for review on project overview",
  },
  openJobCta: {
    defaultMessage: "Open job",
    id: "wdhrNg8UHz",
    description: "CTA for a failed or active job on project overview",
  },
  addGuidanceCta: {
    defaultMessage: "Add style guide",
    id: "U+ZJP/RMin",
    description: "CTA when the project style guide is missing on project overview",
  },
  triageReviewTitle: {
    defaultMessage: "Waiting for review",
    id: "t4mhc63bGr",
    description: "Status label for review triage items",
  },
  triageFailedTitle: {
    defaultMessage: "Job failed",
    id: "piWs1xE1NT",
    description: "Status label for failed job triage items",
  },
  triageGuidanceTitle: {
    defaultMessage: "Add a style guide",
    id: "DGx/LlMkDv",
    description: "Title when the native project style guide is missing",
  },
  triageGuidanceDescription: {
    defaultMessage: "Shared tone and terminology so agents stay consistent.",
    id: "+y5oYiRfPw",
    description: "Description when the native project style guide is missing",
  },
  triageJobRunning: {
    defaultMessage: "In progress",
    id: "KYKSoue+Dr",
    description: "Status label for queued or running jobs in triage",
  },
  viewAllJobs: {
    defaultMessage: "View all jobs",
    id: "QaBpv8qa4h",
    description: "Link from triage band to the project jobs page",
  },
  signalsTitle: {
    defaultMessage: "Project",
    id: "RZNls0g+WM",
    description: "Section heading for lightweight project signals on overview",
  },
  signalsLocales: {
    defaultMessage: "Locales",
    id: "wQOKFwmzrC",
    description: "Label for locale route on project overview signals",
  },
  signalsNoLocales: {
    defaultMessage: "No target locales yet",
    id: "Pw5huaM2vN",
    description: "Shown when the project has no target locales configured",
  },
  guidanceTitle: {
    defaultMessage: "Style guide",
    id: "6KrEdJWOHm",
    description: "Section heading for the style guide preview on project overview",
  },
  guidanceEdit: {
    defaultMessage: "Edit",
    id: "Hvpl2DhEOB",
    description: "Link to edit the style guide in project settings",
  },
  shipTitle: {
    defaultMessage: "Sync",
    id: "cE0bfQDgYq",
    description: "Section heading for native sync status on project overview",
  },
  shipLastSynced: {
    defaultMessage: "Last synced {when}",
    id: "I28Vekk8WA",
    description: "Last sync timestamp on project overview ship section",
  },
  shipNeverSynced: {
    defaultMessage: "Not synced yet",
    id: "wk5x4r43TH",
    description: "Shown when a native project has never synced",
  },
  shipCliHint: {
    defaultMessage: "Download translations from Files or run <code>sync pull</code>.",
    id: "1x7rivVz4j",
    description: "CLI hint in the sync section on project overview",
  },
  shipConnectCli: {
    defaultMessage: "Connect CLI & CI",
    id: "NQq3ItGutD",
    description: "Link to project settings for CLI and CI setup",
  },
  viewSettings: {
    defaultMessage: "View settings",
    id: "PpiEEboJdd",
    description: "Call-to-action linking to project settings",
  },
  viewFiles: {
    defaultMessage: "View files",
    id: "9GZvF1K90w",
    description: "Button linking to the project files page",
  },
  viewJobs: {
    defaultMessage: "View jobs",
    id: "2c+GvUZLgD",
    description: "Button linking to the project jobs page",
  },
  jobsUnavailable: {
    defaultMessage: "Jobs unavailable",
    id: "M4sYVolrI+",
    description: "Empty-state title when project jobs fail to load",
  },
  jobsUnavailableDescription: {
    defaultMessage: "We could not load jobs for this project.",
    id: "CvYNnC7KYY",
    description: "Empty-state description when project jobs fail to load",
  },
});

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

export const dashboardPageViewMessages = defineMessages({
  pageLabel: {
    defaultMessage: "Workspace",
    id: "mRWwmBY6N9",
    description: "Dashboard page header eyebrow label",
  },
  pageTitle: {
    defaultMessage: "Overview",
    id: "6xnpnIBioo",
    description: "Dashboard page heading",
  },
  pageDescription: {
    defaultMessage: "A top-down view of localization operations in this workspace.",
    id: "AIaaCgpWdC",
    description: "Dashboard page description under the heading",
  },
  newRequest: {
    defaultMessage: "New request",
    id: "vKq2ly7dxU",
    description: "Button to start a new localization request from the dashboard",
  },
  loadingWorkspaceOverview: {
    defaultMessage: "Loading workspace overview",
    id: "nhcw8sOpqz",
    description: "Accessible label while the dashboard overview is loading",
  },
  overviewLoadError: {
    defaultMessage: "Workspace overview could not be loaded.",
    id: "LpAm1eoUb6",
    description: "Error when the overview snapshot fails to load",
  },
  jobsMetric: {
    defaultMessage: "Jobs",
    id: "WpT4MqV5Ti",
    description: "Overview metric card title for jobs created in the last 7 days",
  },
  translationsMetric: {
    defaultMessage: "Translations",
    id: "XrHgNqTVDc",
    description: "Overview metric card title for translations updated in the last 7 days",
  },
  automationsMetric: {
    defaultMessage: "Automations",
    id: "EQDi5wnuJu",
    description: "Overview metric card title for workspace automations",
  },
  issuesMetric: {
    defaultMessage: "Open issues",
    id: "cgDZYF+DDY",
    description: "Overview metric card title for open issues",
  },
  lastSevenDays: {
    defaultMessage: "Last 7 days",
    id: "21rpYl9weA",
    description: "Metric card subtitle for a 7-day lookback",
  },
  sparklineBarTooltip: {
    defaultMessage: "{date}: {count}",
    id: "3aG6RMsKle",
    description: "Tooltip for a daily count bar on an overview metric chart",
  },
  sparklineChartAriaLabel: {
    defaultMessage: "{metric} by day: {values}",
    id: "kF6mTL+t+j",
    description: "Accessible summary of the daily count bars on an overview metric card",
  },
  pausedCount: {
    defaultMessage: "{count} paused",
    id: "iN7gragIFi",
    description: "Automations metric subtitle showing how many automations are paused",
  },
  p1Count: {
    defaultMessage: "{count} P1 on Board",
    id: "51tdxkVdu+",
    description: "Open issues metric subtitle showing P1 issue count on the board",
  },
  automationActivityKind: {
    defaultMessage: "Automation",
    id: "f01aYrKnGZ",
    description: "Activity subtitle for an automation run on the overview",
  },
  nativeProjectSource: {
    defaultMessage: "Native",
    id: "JNJTudsDqz",
    description: "Project source label when the project is a native Hyperlocalise project",
  },
  activityLabel: {
    defaultMessage: "Activity",
    id: "2Z3HVQXs5m",
    description: "Overview section label for recent jobs and automation runs",
  },
  projectsLabel: {
    defaultMessage: "Projects",
    id: "VYzt977Ihg",
    description: "Overview section label for workspace projects",
  },
  boardLabel: {
    defaultMessage: "Board",
    id: "MQYav5C7q6",
    description: "Overview section label for open issues",
  },
  automationsLabel: {
    defaultMessage: "Automations",
    id: "+0IukqAGBP",
    description: "Overview section label for recent automation runs",
  },
  viewJobs: {
    defaultMessage: "View jobs",
    id: "S9c8QG8bF2",
    description: "Link from overview activity to the jobs page",
  },
  viewAll: {
    defaultMessage: "View all",
    id: "ZCcOS0o7o1",
    description: "Link from overview projects to the projects page",
  },
  viewBoard: {
    defaultMessage: "View Board",
    id: "M+PhLBJ7kK",
    description: "Link from overview board to the issues page",
  },
  viewAutomations: {
    defaultMessage: "View automations",
    id: "vynGfdRurY",
    description: "Button to open the automations page from the dashboard",
  },
  activityEmpty: {
    defaultMessage: "No recent activity yet.",
    id: "X4Hw6VR6Ws",
    description: "Empty state for overview activity",
  },
  projectsEmpty: {
    defaultMessage: "No projects yet. Create a project to get started.",
    id: "6NGN+mn8CD",
    description: "Empty state when there are no workspace projects",
  },
  boardEmpty: {
    defaultMessage: "No open issues.",
    id: "XNK40wY8Cc",
    description: "Empty state for overview board issues",
  },
  automationsEmpty: {
    defaultMessage: "No automation runs yet.",
    id: "yF+d/IbGFc",
    description: "Empty state when there are no automation runs",
  },
  openAction: {
    defaultMessage: "Open",
    id: "Wm5TT1di5J",
    description: "Link to open a failed activity item",
  },
  projectOpenCount: {
    defaultMessage: "{count} open",
    id: "ho4WUcnmEe",
    description: "Badge showing how many open jobs a project has",
  },
  projectFailedCount: {
    defaultMessage: "{count} failed",
    id: "18aFSqdbKb",
    description: "Badge showing how many failed jobs a project has",
  },
  workspaceFallbackProject: {
    defaultMessage: "Workspace",
    id: "0bdrX6gjre",
    description: "Fallback project name when a job has no project",
  },
  runStatusQueued: {
    defaultMessage: "Queued",
    id: "Khk1oUs6TP",
    description: "Automation run status badge for queued runs",
  },
  runStatusRunning: {
    defaultMessage: "Running",
    id: "D7NvZBBM/6",
    description: "Automation run status badge for running runs",
  },
  runStatusSucceeded: {
    defaultMessage: "Succeeded",
    id: "B+9mu1WlLc",
    description: "Automation run status badge for succeeded runs",
  },
  runStatusFailed: {
    defaultMessage: "Failed",
    id: "jgrmHo9SE2",
    description: "Automation run status badge for failed runs",
  },
  runStatusCancelled: {
    defaultMessage: "Cancelled",
    id: "oKp30YYGZb",
    description: "Automation run status badge for cancelled runs",
  },
  runStatusSkipped: {
    defaultMessage: "Skipped",
    id: "Qklvn9azHF",
    description: "Automation run status badge for skipped runs",
  },
  jobStatusWaiting: {
    defaultMessage: "Waiting",
    id: "NcIdsG0LSX",
    description: "Status pill for jobs waiting for review on the overview",
  },
  triggerManual: {
    defaultMessage: "Manual",
    id: "MpshM2ubvT",
    description: "Automation run trigger source label for manual runs",
  },
  triggerScheduled: {
    defaultMessage: "Scheduled",
    id: "t+TLW1HZ59",
    description: "Automation run trigger source label for scheduled runs",
  },
  triggerGithub: {
    defaultMessage: "GitHub",
    id: "SYEyQ9llcS",
    description: "Automation run trigger source label for GitHub-triggered runs",
  },
  triggerContentful: {
    defaultMessage: "Contentful",
    id: "GdufIv1RQV",
    description: "Automation run trigger source label for Contentful-triggered runs",
  },
  triggerSourceUpload: {
    defaultMessage: "Source upload",
    id: "h3/Sx/1Xgp",
    description: "Automation run trigger source label for source-upload-triggered runs",
  },
  triggerWebChat: {
    defaultMessage: "Web chat",
    id: "KCMz//p+DR",
    description: "Automation run trigger source label for web-chat-triggered runs",
  },
});

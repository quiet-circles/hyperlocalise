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
"use client";
import { defineMessages } from "react-intl";
export const reportMessages = defineMessages({
  title: {
    id: "uUKIpz/nd7",
    defaultMessage: "Reports",
    description: "Translation reporting label",
  },
  overview: {
    id: "ceDFVw/CRC",
    defaultMessage: "Overview",
    description: "Translation reporting label",
  },
  words: {
    id: "nfT64/qfUE",
    defaultMessage: "Word counts",
    description: "Translation reporting label",
  },
  time: {
    id: "+h67bEnfK7",
    defaultMessage: "Time",
    description: "Translation reporting label",
  },
  costs: {
    id: "idVdv+g+Al",
    defaultMessage: "Costs",
    description: "Translation reporting label",
  },
  from: {
    id: "HizH8Sur70",
    defaultMessage: "From",
    description: "Translation reporting label",
  },
  to: { id: "G13YyZNkrb", defaultMessage: "To", description: "Translation reporting label" },
  projectId: {
    id: "Il0gK7uLCw",
    defaultMessage: "Project",
    description: "Translation reporting label",
  },
  jobId: {
    id: "fiPdUBwOKp",
    defaultMessage: "Task ID",
    description: "Translation reporting label",
  },
  targetLocale: {
    id: "wFAehxKEKp",
    defaultMessage: "Target language",
    description: "Translation reporting label",
  },
  sourceLocale: {
    id: "VpriA1wu00",
    defaultMessage: "Source language",
    description: "Translation reporting label",
  },
  step: {
    id: "yJkKJLXXDK",
    defaultMessage: "Step",
    description: "Translation reporting label",
  },
  interval: {
    id: "8hy7KRery5",
    defaultMessage: "Group by",
    description: "Translation reporting label",
  },
  day: {
    id: "oZKIJnBlXM",
    defaultMessage: "Daily",
    description: "Translation reporting label",
  },
  week: {
    id: "GygH5eMQXV",
    defaultMessage: "Weekly",
    description: "Translation reporting label",
  },
  all: {
    id: "X1JobycgOR",
    defaultMessage: "All",
    description: "Translation reporting label",
  },
  export: {
    id: "oWD/trTqPz",
    defaultMessage: "Export CSV",
    description: "Translation reporting label",
  },
  loading: {
    id: "eoeCTUBKbh",
    defaultMessage: "Loading reports\u2026",
    description: "Translation reporting label",
  },
  error: {
    id: "DCD7XCdOZZ",
    defaultMessage: "Unable to load reports. Please try again.",
    description: "Translation reporting label",
  },
  retry: {
    id: "VoN3hfznUR",
    defaultMessage: "Try again",
    description: "Translation reporting label",
  },
  empty: {
    id: "F7WfKl5UxS",
    defaultMessage:
      "No activity recorded for these filters. Run a translation or log time to start reporting.",
    description: "Translation reporting label",
  },
  unallocated: {
    id: "jCFx+R25Yk",
    defaultMessage: "Unallocated",
    description: "Cost that cannot be attributed to a single target language",
  },
  allocationHelp: {
    id: "iC2gG2o0Em",
    defaultMessage:
      "Language filters also show unallocated costs. These shared costs are not attributed to the selected language.",
    description: "Explains shared costs in language-filtered reports",
  },
  taskRateLocked: {
    id: "1OhnBb85Zc",
    defaultMessage:
      "Task rates can only change before work starts. Recorded costs keep their original rates.",
    description: "Explains why an existing task rate cannot be changed",
  },
  prospective: {
    id: "4g0qoebzmh",
    defaultMessage:
      "Reporting collects new activity only. Earlier activity is unavailable, not zero. Dates and weeks use UTC.",
    description: "Translation reporting label",
  },
  started: {
    id: "dIjV9bElOk",
    defaultMessage: "Reporting started: {date}",
    description: "Translation reporting label",
  },
  notStarted: {
    id: "BXQNGYUADX",
    defaultMessage: "Reporting has not started yet.",
    description: "Translation reporting label",
  },
  period: {
    id: "jNAmRjZwLm",
    defaultMessage: "Period (UTC)",
    description: "Translation reporting label",
  },
  bucket: {
    id: "og0hJhgZ7f",
    defaultMessage: "Match",
    description: "Translation reporting label",
  },
  kind: {
    id: "KvqvzlXjFp",
    defaultMessage: "Activity",
    description: "Translation reporting label",
  },
  minutes: {
    id: "fVkQmhbtrN",
    defaultMessage: "Minutes",
    description: "Translation reporting label",
  },
  durationMs: {
    id: "vKTyOSIgVr",
    defaultMessage: "Execution time (ms)",
    description: "Translation reporting label",
  },
  amountUsd: {
    id: "g6yC/xE58y",
    defaultMessage: "Cost (USD)",
    description: "Translation reporting label",
  },
  inputTokens: {
    id: "NUXNiXqQOj",
    defaultMessage: "Input tokens",
    description: "Translation reporting label",
  },
  outputTokens: {
    id: "qcuUGO7z6v",
    defaultMessage: "Output tokens",
    description: "Translation reporting label",
  },
  unavailable: {
    id: "jdk6igRynG",
    defaultMessage: "Unavailable",
    description: "Translation reporting label",
  },
  count: {
    id: "uiUTzHtrEP",
    defaultMessage: "Events",
    description: "Translation reporting label",
  },
  translation: {
    id: "XmcWHG0JFP",
    defaultMessage: "Translation",
    description: "Translation reporting label",
  },
  review: {
    id: "1vsHtnlJ64",
    defaultMessage: "Review",
    description: "Translation reporting label",
  },
  name: {
    id: "r01vthnyrV",
    defaultMessage: "Rate card name",
    description: "Translation reporting label",
  },
  basis: {
    id: "iMQKhwz7gh",
    defaultMessage: "Pricing basis",
    description: "Translation reporting label",
  },
  word: {
    id: "ymcbvyHwtl",
    defaultMessage: "Per word",
    description: "Translation reporting label",
  },
  hour: {
    id: "oKhMD/kdj3",
    defaultMessage: "Per hour",
    description: "Translation reporting label",
  },
  rate: {
    id: "j+CBv2iqBK",
    defaultMessage: "Rate (USD)",
    description: "Translation reporting label",
  },
  budget: {
    id: "3+I4HMb5MN",
    defaultMessage: "Budget (USD)",
    description: "Translation reporting label",
  },
  rateCardName: {
    id: "5n8TzfHe24",
    defaultMessage: "Default rate card name",
    description: "Translation reporting label",
  },
  estimatedMinutes: {
    id: "UFGE6jBR7S",
    defaultMessage: "Estimated minutes",
    description: "Translation reporting label",
  },
  overrideUsd: {
    id: "MVijhDEjv/",
    defaultMessage: "Human cost override (USD)",
    description: "Translation reporting label",
  },
  rateId: {
    id: "0iQPMSh0Vj",
    defaultMessage: "Task rate",
    description: "Translation reporting label",
  },
  workDate: {
    id: "KbRaMGCw9d",
    defaultMessage: "Work date (UTC)",
    description: "Translation reporting label",
  },
  note: {
    id: "57UDNEMoCB",
    defaultMessage: "Note",
    description: "Translation reporting label",
  },
  save: {
    id: "qhnooZhwIt",
    defaultMessage: "Save",
    description: "Translation reporting label",
  },
  saved: {
    id: "Y+2g7ZpTUk",
    defaultMessage: "Saved",
    description: "Translation reporting label",
  },
  saveError: {
    id: "YNiI/7MZFy",
    defaultMessage: "Unable to save. Check the values and try again.",
    description: "Translation reporting label",
  },
  rateForm: {
    id: "0ZcBNFxj2u",
    defaultMessage: "Add rate version",
    description: "Translation reporting label",
  },
  budgetForm: {
    id: "Jlg9FCraTi",
    defaultMessage: "Set project budget",
    description: "Translation reporting label",
  },
  timeForm: {
    id: "ULXNCw9YAZ",
    defaultMessage: "Log time",
    description: "Translation reporting label",
  },
  expenseForm: {
    id: "FHXR08HCFK",
    defaultMessage: "Record expense",
    description: "Translation reporting label",
  },
  taskRateForm: {
    id: "tNhXqAYLEr",
    defaultMessage: "Set task pricing",
    description: "Translation reporting label",
  },
  accruedUsd: {
    id: "Wyrjwu8fRi",
    defaultMessage: "Accrued (USD)",
    description: "Translation reporting label",
  },
  outstandingUsd: {
    id: "Y68HG1V9vz",
    defaultMessage: "Outstanding (USD)",
    description: "Translation reporting label",
  },
  forecastUsd: {
    id: "TictSeLm8D",
    defaultMessage: "Forecast (USD)",
    description: "Translation reporting label",
  },
  remainingUsd: {
    id: "kE7O20HfQk",
    defaultMessage: "Remaining (USD)",
    description: "Translation reporting label",
  },
  incomplete: {
    id: "IROF4KmxXQ",
    defaultMessage: "Incomplete estimate",
    description: "Translation reporting label",
  },
  spendWarning: {
    id: "bu/fF+Rrb5",
    defaultMessage: "Spend warning",
    description: "Translation reporting label",
  },
  forecastWarning: {
    id: "fi/3cd111E",
    defaultMessage: "Forecast warning",
    description: "Translation reporting label",
  },
  none: {
    id: "2MhyTTF5nR",
    defaultMessage: "Within budget",
    description: "Translation reporting label",
  },
  approaching: {
    id: "wfWyTH0EQs",
    defaultMessage: "Approaching budget",
    description: "Translation reporting label",
  },
  exceeded: {
    id: "uK/UF8qzfF",
    defaultMessage: "Budget exceeded",
    description: "Translation reporting label",
  },
  analysis: {
    id: "5X1hpw6ahf",
    defaultMessage: "Task analysis",
    description: "Translation reporting label",
  },
  timeEntries: {
    id: "AFoch1/QFD",
    defaultMessage: "Time entries",
    description: "Translation reporting label",
  },
  void: {
    id: "I+Kbw5ZnI0",
    defaultMessage: "Remove entry",
    description: "Translation reporting label",
  },
  confirmVoid: {
    id: "oKXCvcM8O7",
    defaultMessage: "Remove this time entry?",
    description: "Translation reporting label",
  },
  confirmVoidDescription: {
    id: "8JhGQ1Ny08",
    defaultMessage:
      "The entry and its calculated cost will be removed from totals. An audit record is retained.",
    description: "Translation reporting label",
  },
  cancel: {
    id: "Edot+nksiq",
    defaultMessage: "Cancel",
    description: "Translation reporting label",
  },
  sourceWords: {
    id: "5qbTrwpH1i",
    defaultMessage: "Source words",
    description: "Translation reporting label",
  },
  workloadWords: {
    id: "Q19ddI8gEn",
    defaultMessage: "Completed workload words",
    description: "Translation reporting label",
  },
  hours: {
    id: "GzgrtCYOAU",
    defaultMessage: "Logged hours",
    description: "Translation reporting label",
  },
  tokens: {
    id: "psFFq8CuYE",
    defaultMessage: "AI tokens",
    description: "Translation reporting label",
  },
  percentages: {
    id: "pEdVdVrZIy",
    defaultMessage: "Match payable percentages",
    description: "Translation reporting label",
  },
  repetition: {
    id: "TRakT3+hGy",
    defaultMessage: "Repetitions",
    description: "Translation reporting label",
  },
  "100": {
    id: "ULeYUNSVim",
    defaultMessage: "100%",
    description: "Translation reporting label",
  },
  "95-99": {
    id: "EFds7VbW/w",
    defaultMessage: "95\u201399%",
    description: "Translation reporting label",
  },
  "85-94": {
    id: "seTM+j07h8",
    defaultMessage: "85\u201394%",
    description: "Translation reporting label",
  },
  "75-84": {
    id: "kRxU5ry3a+",
    defaultMessage: "75\u201384%",
    description: "Translation reporting label",
  },
  "50-74": {
    id: "PpTSCPxwk9",
    defaultMessage: "50\u201374%",
    description: "Translation reporting label",
  },
  new: {
    id: "XhmzurY1tl",
    defaultMessage: "Below 50% / new",
    description: "Translation reporting label",
  },
  segmentId: {
    id: "iDCSMhBSZF",
    defaultMessage: "Segment",
    description: "Translation reporting label",
  },
  analysisUnavailable: {
    id: "HLUpd07Vd7",
    defaultMessage: "Analysis unavailable",
    description: "Translation reporting label",
  },
  completion: {
    id: "wJ0RllAJ9H",
    defaultMessage: "Completed",
    description: "Translation reporting label",
  },
  status: {
    id: "ilFwBwzxTP",
    defaultMessage: "Status change",
    description: "Translation reporting label",
  },
  workflow: {
    id: "6emsPwQUkU",
    defaultMessage: "Workflow",
    description: "Translation reporting label",
  },
  human: {
    id: "lVdc+gO4U4",
    defaultMessage: "Human",
    description: "Translation reporting label",
  },
  ai: { id: "0jU9ov00EY", defaultMessage: "AI", description: "Translation reporting label" },
  expense: {
    id: "cSx8PhiZkw",
    defaultMessage: "Expense",
    description: "Translation reporting label",
  },
  noBudget: {
    id: "3yl8ZKATF0",
    defaultMessage: "No project budgets configured. Set a budget below to begin tracking.",
    description: "Translation reporting label",
  },
  pricingHelp: {
    id: "44rQmsHaoJ",
    defaultMessage:
      "Saving a rate creates a new version. Existing snapshots keep their applied rates.",
    description: "Translation reporting label",
  },
  unknownCost: {
    id: "bo2HxpLRUX",
    defaultMessage: "Unpriced",
    description: "Translation reporting label",
  },
  reviewReport: {
    id: "5a+z7ec3fj",
    defaultMessage: "View report",
    description: "Translation reporting label",
  },
  rateHelp: {
    id: "APe9cqw+Vb",
    defaultMessage:
      "Choose a rate matching the task language and step. Leave blank to use the project default.",
    description: "Translation reporting label",
  },
});

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
import type { Edge } from "@xyflow/react";

import { createDefaultConfig } from "@/lib/visual-workflows/catalog/node-catalog";
import { fromVisualWorkflowDefinition } from "@/lib/visual-workflows/schema/serializers";
import type {
  MockNodeRunStatus,
  VisualCatalogType,
  VisualNodeConfig,
  VisualWorkflowDefinition,
  VisualWorkflowRfNode,
} from "@/lib/visual-workflows/schema/types";
import { VISUAL_WORKFLOW_SCHEMA_VERSION } from "@/lib/visual-workflows/schema/types";

export type ContentOpsFlowTemplateId = "brief" | "campaign";

export type ContentOpsFlowGraph = {
  nodes: VisualWorkflowRfNode[];
  edges: Edge[];
  order: string[];
};

function httpPostConfig(url: string): VisualNodeConfig {
  const config = createDefaultConfig("action.http");
  if (config.kind !== "action.http") {
    return config;
  }

  return { ...config, method: "POST", url };
}

function aiConfig(prompt: string): VisualNodeConfig {
  return { kind: "ai.agent", prompt, onError: "stop" };
}

function slackConfig(channelId: string, message: string): VisualNodeConfig {
  return { kind: "action.notify_slack", channelId, message, onError: "stop" };
}

function scheduledConfig(): VisualNodeConfig {
  return {
    kind: "trigger.scheduled",
    schedule: { cadence: "weekly", hourUtc: 9, timezone: "UTC" },
  };
}

const BRIEF_ORDER = [
  "schedule",
  "keywords",
  "draft",
  "localise",
  "review",
  "cms",
  "slack",
] as const;

const CAMPAIGN_ORDER = ["brief", "localise", "review", "staging", "slack"] as const;

const BRIEF_DEFINITION: VisualWorkflowDefinition = {
  schemaVersion: VISUAL_WORKFLOW_SCHEMA_VERSION,
  name: "Multilingual blog",
  nodes: [
    { id: "schedule", type: "trigger.scheduled", config: scheduledConfig() },
    { id: "keywords", type: "ai.agent", config: aiConfig("Research keywords") },
    { id: "draft", type: "ai.agent", config: aiConfig("Create content draft") },
    { id: "localise", type: "ai.agent", config: aiConfig("Localise") },
    { id: "review", type: "logic.if", config: { kind: "logic.if", condition: "review.passed" } },
    {
      id: "cms",
      type: "action.http",
      config: httpPostConfig("https://cms.acme.com/posts"),
    },
    {
      id: "slack",
      type: "action.notify_slack",
      config: slackConfig("#content", "Blog draft is live"),
    },
  ],
  edges: [
    {
      id: "brief-e-0",
      source: "schedule",
      target: "keywords",
      sourceHandle: null,
      targetHandle: null,
    },
    {
      id: "brief-e-1",
      source: "keywords",
      target: "draft",
      sourceHandle: null,
      targetHandle: null,
    },
    {
      id: "brief-e-2",
      source: "draft",
      target: "localise",
      sourceHandle: null,
      targetHandle: null,
    },
    {
      id: "brief-e-3",
      source: "localise",
      target: "review",
      sourceHandle: null,
      targetHandle: null,
    },
    { id: "brief-e-4", source: "review", target: "cms", sourceHandle: "true", targetHandle: null },
    {
      id: "brief-e-5",
      source: "review",
      target: "slack",
      sourceHandle: "true",
      targetHandle: null,
    },
  ],
  editor: {
    positions: {
      schedule: { x: 40, y: 160 },
      keywords: { x: 300, y: 160 },
      draft: { x: 560, y: 160 },
      localise: { x: 820, y: 160 },
      review: { x: 1080, y: 160 },
      cms: { x: 1340, y: 40 },
      slack: { x: 1340, y: 280 },
    },
  },
};

const CAMPAIGN_DEFINITION: VisualWorkflowDefinition = {
  schemaVersion: VISUAL_WORKFLOW_SCHEMA_VERSION,
  name: "Campaign",
  nodes: [
    { id: "brief", type: "trigger.manual", config: createDefaultConfig("trigger.manual") },
    { id: "localise", type: "ai.agent", config: aiConfig("Localise") },
    { id: "review", type: "logic.if", config: { kind: "logic.if", condition: "review.passed" } },
    {
      id: "staging",
      type: "action.http",
      config: httpPostConfig("https://staging.acme.com/launch"),
    },
    {
      id: "slack",
      type: "action.notify_slack",
      config: slackConfig("#launch", "Campaign is on staging"),
    },
  ],
  edges: [
    {
      id: "campaign-e-0",
      source: "brief",
      target: "localise",
      sourceHandle: null,
      targetHandle: null,
    },
    {
      id: "campaign-e-1",
      source: "localise",
      target: "review",
      sourceHandle: null,
      targetHandle: null,
    },
    {
      id: "campaign-e-2",
      source: "review",
      target: "staging",
      sourceHandle: "true",
      targetHandle: null,
    },
    {
      id: "campaign-e-3",
      source: "review",
      target: "slack",
      sourceHandle: "true",
      targetHandle: null,
    },
  ],
  editor: {
    positions: {
      brief: { x: 40, y: 160 },
      localise: { x: 300, y: 160 },
      review: { x: 560, y: 160 },
      staging: { x: 820, y: 40 },
      slack: { x: 820, y: 280 },
    },
  },
};

const DEFINITIONS: Record<ContentOpsFlowTemplateId, VisualWorkflowDefinition> = {
  brief: BRIEF_DEFINITION,
  campaign: CAMPAIGN_DEFINITION,
};

const ORDERS: Record<ContentOpsFlowTemplateId, readonly string[]> = {
  brief: BRIEF_ORDER,
  campaign: CAMPAIGN_ORDER,
};

function runStatusForIndex(index: number, activeIndex: number): MockNodeRunStatus {
  if (index < 0) {
    return "idle";
  }
  if (index < activeIndex) {
    return "succeeded";
  }
  if (index === activeIndex) {
    return "running";
  }
  return "idle";
}

export function contentOpsFlowNodeOrder(templateId: ContentOpsFlowTemplateId): readonly string[] {
  return ORDERS[templateId];
}

export function contentOpsFlowCatalogTypes(
  templateId: ContentOpsFlowTemplateId,
): VisualCatalogType[] {
  return DEFINITIONS[templateId].nodes.map((node) => node.type);
}

export function styleContentOpsFlowEdges(
  edges: Edge[],
  order: readonly string[],
  activeIndex: number,
): Edge[] {
  const activeId = order[activeIndex];
  const completed = new Set(order.slice(0, activeIndex));

  return edges.map((edge) => {
    const isActivePath = edge.source === activeId;
    const isCompletePath = completed.has(edge.source);

    return {
      ...edge,
      animated: isActivePath,
      style: {
        strokeWidth: isCompletePath || isActivePath ? 2 : 1.5,
        stroke: isCompletePath || isActivePath ? "var(--primary)" : "var(--border)",
      },
    };
  });
}

export function buildContentOpsFlowGraph(
  templateId: ContentOpsFlowTemplateId,
  subtitles: Readonly<Record<string, string>>,
  activeIndex: number,
): ContentOpsFlowGraph {
  const definition = DEFINITIONS[templateId];
  const order = ORDERS[templateId];
  const draft = fromVisualWorkflowDefinition(definition);

  const nodes = draft.nodes.map((node) => {
    const index = order.indexOf(node.id);

    return {
      ...node,
      draggable: true,
      data: {
        ...node.data,
        previewSubtitle: subtitles[node.id] ?? node.data.previewSubtitle,
        hideAddAction: true,
        runStatus: runStatusForIndex(index, activeIndex),
      },
    };
  });

  return {
    nodes,
    edges: styleContentOpsFlowEdges(draft.edges, order, activeIndex),
    order: [...order],
  };
}

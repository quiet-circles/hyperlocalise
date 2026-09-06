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
import { describe, expect, it } from "vite-plus/test";

import {
  buildContentOpsFlowGraph,
  contentOpsFlowCatalogTypes,
  styleContentOpsFlowEdges,
} from "./content-ops-flow-graph";

describe("content-ops-flow-graph", () => {
  it("builds the multilingual blog graph from visual workflow catalog types", () => {
    expect(contentOpsFlowCatalogTypes("brief")).toEqual([
      "trigger.scheduled",
      "ai.agent",
      "ai.agent",
      "ai.agent",
      "logic.if",
      "action.http",
      "action.notify_slack",
    ]);

    const graph = buildContentOpsFlowGraph(
      "brief",
      {
        schedule: "Scheduled run",
        keywords: "Keyword research",
        draft: "Create content draft",
        localise: "Localise",
        review: "Review",
        cms: "CMS publish",
        slack: "Slack notify",
      },
      1,
    );

    expect(graph.nodes.map((node) => node.type)).toEqual([
      "trigger.scheduled",
      "ai.agent",
      "ai.agent",
      "ai.agent",
      "logic.if",
      "action.http",
      "action.notify_slack",
    ]);
    expect(graph.nodes[1]?.data.runStatus).toBe("running");
    expect(graph.nodes[1]?.data.hideAddAction).toBe(true);
    expect(graph.nodes[1]?.data.previewSubtitle).toBe("Keyword research");
    expect(
      graph.edges.filter((edge) => edge.source === "review").map((edge) => edge.sourceHandle),
    ).toEqual(["true", "true"]);
  });

  it("builds the campaign graph with a manual trigger and review branch", () => {
    const graph = buildContentOpsFlowGraph(
      "campaign",
      {
        brief: "GTM brief",
        localise: "Localise",
        review: "Review",
        staging: "Staging",
        slack: "Slack notify",
      },
      0,
    );

    expect(graph.nodes[0]?.type).toBe("trigger.manual");
    expect(graph.nodes[2]?.type).toBe("logic.if");
    expect(graph.nodes[3]?.type).toBe("action.http");
    expect(graph.nodes[4]?.type).toBe("action.notify_slack");
    expect(graph.edges.some((edge) => edge.source === "review" && edge.target === "staging")).toBe(
      true,
    );
  });

  it("animates edges leaving the active node", () => {
    const graph = buildContentOpsFlowGraph("brief", {}, 4);
    const styled = styleContentOpsFlowEdges(graph.edges, graph.order, 4);
    const reviewEdges = styled.filter((edge) => edge.source === "review");

    expect(reviewEdges.length).toBe(2);
    expect(reviewEdges.every((edge) => edge.animated)).toBe(true);
  });
});

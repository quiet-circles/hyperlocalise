// @vitest-environment happy-dom

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
import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { IntlProvider } from "react-intl";
import { describe, expect, it, vi } from "vite-plus/test";

import type { VisualWorkflowRfNode } from "@/lib/visual-workflows/schema/types";

import { ContentOpsFlowPanel } from "./content-ops-flow-panel";

vi.mock("@/components/ai-elements/canvas", () => ({
  Canvas: ({ nodes, children }: { nodes: VisualWorkflowRfNode[]; children?: ReactNode }) => (
    <div data-testid="visual-workflow-canvas">
      {nodes.map((node) => (
        <div key={node.id}>
          {node.type}:{node.data.previewSubtitle}
        </div>
      ))}
      {children}
    </div>
  ),
}));

vi.mock("@/components/ai-elements/controls", () => ({
  Controls: () => <div>Canvas controls</div>,
}));

describe("ContentOpsFlowPanel", () => {
  it("renders visual workflow catalog nodes for the multilingual blog template", () => {
    render(
      <IntlProvider locale="en" messages={{}}>
        <ContentOpsFlowPanel pauseAutoplay />
      </IntlProvider>,
    );

    expect(screen.getByTestId("visual-workflow-canvas")).toBeInTheDocument();
    expect(screen.getByText("trigger.scheduled:Scheduled run")).toBeInTheDocument();
    expect(screen.getByText("ai.agent:Keyword research")).toBeInTheDocument();
    expect(screen.getByText("logic.if:Review")).toBeInTheDocument();
    expect(screen.getByText("action.http:CMS publish")).toBeInTheDocument();
    expect(screen.getByText("action.notify_slack:Slack notify")).toBeInTheDocument();
    expect(screen.getByText("Preview")).toBeInTheDocument();
  });

  it("switches to campaign catalog nodes", async () => {
    const user = userEvent.setup();

    render(
      <IntlProvider locale="en" messages={{}}>
        <ContentOpsFlowPanel pauseAutoplay />
      </IntlProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Campaign" }));

    expect(screen.getByText("trigger.manual:GTM brief")).toBeInTheDocument();
    expect(screen.getByText("action.http:Staging")).toBeInTheDocument();
    expect(screen.queryByText("trigger.scheduled:Scheduled run")).not.toBeInTheDocument();
  });
});

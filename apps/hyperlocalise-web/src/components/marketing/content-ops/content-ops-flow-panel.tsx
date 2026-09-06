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
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  MarkerType,
  useEdgesState,
  useNodesState,
  type OnEdgesChange,
  type OnNodesChange,
} from "@xyflow/react";
import { FormattedMessage, useIntl } from "react-intl";

import { visualWorkflowEditorMessages as editorMessages } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/automations/_components/visual-workflow-editor/visual-workflow-editor.messages";
import { VisualWorkflowCanvasActionsProvider } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/automations/_components/visual-workflow-editor/visual-workflow-canvas-actions";
import { VISUAL_WORKFLOW_NODE_TYPES } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/automations/_components/visual-workflow-editor/visual-workflow-canvas";
import { Canvas } from "@/components/ai-elements/canvas";
import { Controls } from "@/components/ai-elements/controls";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/primitives/cn";
import type { VisualWorkflowRfNode } from "@/lib/visual-workflows/schema/types";

import {
  buildContentOpsFlowGraph,
  contentOpsFlowNodeOrder,
  styleContentOpsFlowEdges,
  type ContentOpsFlowTemplateId,
} from "./content-ops-flow-graph";
import { CONTENT_OPS_MOCK_INNER_CLASSNAME } from "./content-ops-mock-stage.constants";
import { contentOpsMockStageMessages } from "./content-ops-mock-stage.messages";

function buildFlowState(
  templateId: ContentOpsFlowTemplateId,
  subtitles: Readonly<Record<string, string>>,
  activeIndex: number,
) {
  return buildContentOpsFlowGraph(templateId, subtitles, activeIndex);
}

export function ContentOpsFlowPanel({
  pauseAutoplay = false,
  onActiveNodeChange,
}: {
  pauseAutoplay?: boolean;
  onActiveNodeChange?: (nodeIndex: number) => void;
}) {
  const intl = useIntl();
  const [templateId, setTemplateId] = useState<ContentOpsFlowTemplateId>("brief");
  const [activeIndex, setActiveIndex] = useState(0);

  const subtitles = useMemo((): Record<string, string> => {
    if (templateId === "campaign") {
      return {
        brief: intl.formatMessage(contentOpsMockStageMessages.flowNodeBrief),
        localise: intl.formatMessage(contentOpsMockStageMessages.flowNodeLocalise),
        review: intl.formatMessage(contentOpsMockStageMessages.flowNodeReview),
        staging: intl.formatMessage(contentOpsMockStageMessages.flowNodeStaging),
        slack: intl.formatMessage(contentOpsMockStageMessages.flowNodeSlack),
      };
    }

    return {
      schedule: intl.formatMessage(contentOpsMockStageMessages.flowNodeSchedule),
      keywords: intl.formatMessage(contentOpsMockStageMessages.flowNodeKeywords),
      draft: intl.formatMessage(contentOpsMockStageMessages.flowNodeCreateContent),
      localise: intl.formatMessage(contentOpsMockStageMessages.flowNodeLocalise),
      review: intl.formatMessage(contentOpsMockStageMessages.flowNodeReview),
      cms: intl.formatMessage(contentOpsMockStageMessages.flowNodeCms),
      slack: intl.formatMessage(contentOpsMockStageMessages.flowNodeSlack),
    };
  }, [intl, templateId]);

  const order = contentOpsFlowNodeOrder(templateId);
  const initialFlow = useMemo(
    () => buildFlowState(templateId, subtitles, 0),
    [subtitles, templateId],
  );
  const [nodes, setNodes, onNodesChange] = useNodesState(initialFlow.nodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialFlow.edges);

  const handleTemplateSelect = useCallback((nextTemplateId: ContentOpsFlowTemplateId) => {
    setTemplateId(nextTemplateId);
    setActiveIndex(0);
  }, []);

  useEffect(() => {
    const nextFlow = buildFlowState(templateId, subtitles, 0);
    setNodes(nextFlow.nodes);
    setEdges(nextFlow.edges);
  }, [setEdges, setNodes, subtitles, templateId]);

  useEffect(() => {
    setNodes((current) =>
      current.map((node) => {
        const index = order.indexOf(node.id);

        return {
          ...node,
          data: {
            ...node.data,
            runStatus:
              index < 0
                ? "idle"
                : index < activeIndex
                  ? "succeeded"
                  : index === activeIndex
                    ? "running"
                    : "idle",
          },
        };
      }),
    );
    setEdges((current) => styleContentOpsFlowEdges(current, order, activeIndex));
  }, [activeIndex, order, setEdges, setNodes]);

  useEffect(() => {
    onActiveNodeChange?.(activeIndex);
  }, [activeIndex, onActiveNodeChange]);

  useEffect(() => {
    if (pauseAutoplay) {
      return;
    }

    const timer = setInterval(() => {
      setActiveIndex((index) => (index + 1) % order.length);
    }, 1400);

    return () => clearInterval(timer);
  }, [order.length, pauseAutoplay]);

  const templates: {
    id: ContentOpsFlowTemplateId;
    label: typeof contentOpsMockStageMessages.flowTemplateBrief;
  }[] = [
    { id: "brief", label: contentOpsMockStageMessages.flowTemplateBrief },
    { id: "campaign", label: contentOpsMockStageMessages.flowTemplateCampaign },
  ];

  const meta =
    templateId === "brief"
      ? {
          title: contentOpsMockStageMessages.flowTemplateBrief,
          description: contentOpsMockStageMessages.flowBriefDescription,
        }
      : {
          title: contentOpsMockStageMessages.flowTemplateCampaign,
          description: contentOpsMockStageMessages.flowCampaignDescription,
        };

  return (
    <div className={CONTENT_OPS_MOCK_INNER_CLASSNAME}>
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-5 py-4">
        <div className="min-w-0 space-y-0.5">
          <div className="flex items-center gap-2">
            <p className="text-sm font-semibold tracking-[-0.02em] text-foreground">
              <FormattedMessage {...meta.title} />
            </p>
            <Badge variant="outline" className="rounded-full">
              <FormattedMessage {...editorMessages.previewBadge} />
            </Badge>
          </div>
          <p className="max-w-md text-xs leading-relaxed text-muted-foreground">
            <FormattedMessage {...meta.description} />
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="hidden text-[10px] text-muted-foreground sm:inline">
            <FormattedMessage {...contentOpsMockStageMessages.flowDragHint} />
          </span>
          <div className="flex flex-wrap gap-1.5">
            {templates.map((template) => (
              <button
                key={template.id}
                type="button"
                onClick={() => handleTemplateSelect(template.id)}
                className={cn(
                  "cursor-pointer rounded-full border px-2.5 py-1 text-[10px] font-medium transition-colors",
                  templateId === template.id
                    ? "border-primary/40 bg-primary/10 text-foreground"
                    : "border-border/60 text-muted-foreground hover:border-border hover:text-foreground",
                )}
              >
                <FormattedMessage {...template.label} />
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="relative min-h-0 min-h-[28rem] w-full flex-1">
        <VisualWorkflowCanvasActionsProvider onAddFromNode={() => undefined}>
          <Canvas
            className="h-full w-full"
            defaultEdgeOptions={{
              type: "smoothstep",
              markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
              style: { strokeWidth: 1.5, stroke: "var(--border)" },
            }}
            edges={edges}
            elementsSelectable
            fitView
            fitViewOptions={{ padding: 0.2, minZoom: 0.55, maxZoom: 1.05 }}
            maxZoom={1.25}
            minZoom={0.5}
            nodeTypes={VISUAL_WORKFLOW_NODE_TYPES}
            nodes={nodes as VisualWorkflowRfNode[]}
            nodesConnectable={false}
            nodesDraggable
            onEdgesChange={onEdgesChange as OnEdgesChange}
            onInit={(reactFlow) => {
              void reactFlow.fitView({ padding: 0.2 });
            }}
            onNodesChange={onNodesChange as OnNodesChange}
            panOnDrag
            panOnScroll
            preventScrolling={false}
            proOptions={{ hideAttribution: true }}
            selectionOnDrag={false}
            zoomOnScroll={false}
          >
            <Controls showInteractive={false} position="bottom-left" />
          </Canvas>
        </VisualWorkflowCanvasActionsProvider>
      </div>
    </div>
  );
}

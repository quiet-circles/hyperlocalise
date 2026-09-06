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
import Link from "next/link";
import { FormattedMessage, useIntl } from "react-intl";

import { useProjectPageQuery } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/projects/[projectId]/_components/project-page-shell";
import { buildProjectPath } from "@/components/app-shell/navigation-config";
import { MarkdownPreview } from "@/components/markdown-editor/markdown-editor";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { TypographyP } from "@/components/ui/typography";

import { contentEditorStyleGuideSheetMessages as messages } from "./content-editor-style-guide-sheet.messages";

export function ContentEditorStyleGuideSheet({
  organizationSlug,
  projectId,
  open,
  onOpenChange,
}: {
  organizationSlug: string;
  projectId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const intl = useIntl();
  const projectQuery = useProjectPageQuery(organizationSlug, projectId, { enabled: open });
  const project = projectQuery.data;
  const content = project?.translationContextValue ?? "";
  const canEdit = project?.source === "native";
  const settingsHref = buildProjectPath(organizationSlug, projectId, "settings");

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-xl" aria-busy={projectQuery.isFetching}>
        <SheetHeader>
          <SheetTitle>
            <FormattedMessage {...messages.title} />
          </SheetTitle>
          <SheetDescription>
            <FormattedMessage {...messages.description} />
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-6">
          {projectQuery.isLoading ? (
            <TypographyP size="small" tone="subtle">
              <FormattedMessage {...messages.loading} />
            </TypographyP>
          ) : null}

          {projectQuery.isError ? (
            <TypographyP className="text-flame-100" size="small">
              <FormattedMessage {...messages.loadError} />
            </TypographyP>
          ) : null}

          {project && !projectQuery.isLoading ? (
            <MarkdownPreview
              value={content}
              emptyMessage={intl.formatMessage(messages.empty)}
              className="border-border bg-transparent"
            />
          ) : null}
        </div>

        {canEdit ? (
          <SheetFooter>
            <Button
              nativeButton={false}
              variant="outline"
              render={<Link href={settingsHref} />}
              onClick={() => onOpenChange(false)}
            >
              <FormattedMessage {...messages.editInSettings} />
            </Button>
          </SheetFooter>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

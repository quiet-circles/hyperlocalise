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
import { useState, type ReactNode } from "react";
import { Add01Icon, ArrowRight01Icon, LanguageCircleIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { FormattedMessage, useIntl } from "react-intl";

import { buildProjectPath } from "@/components/app-shell/navigation-config";
import { MarkdownPreview } from "@/components/markdown-editor/markdown-editor";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { TypographyH1, TypographyP } from "@/components/ui/typography";
import { supportsContentEditorAllFilesProvider } from "@/lib/projects/content-editor-all-files";
import { parseProviderProjectId } from "@/lib/providers/jobs/tms-provider-resource-id";

import { OverviewSectionHeader } from "../../../_components/overview/overview-section-header";
import { CreateJobDialog } from "../../../jobs/_components/create-job-dialog";
import {
  getJobName,
  taskDetailSummary,
  type ApiJob,
} from "../../../jobs/_components/jobs-page-view";
import type { ProjectListRow } from "../../_components/project-list";
import { ProjectOverviewMeshStage } from "./project-overview-mesh-stage";
import { projectOverviewPageContentMessages as messages } from "./project-overview-page-content.messages";
import { ProjectPageShell, useProjectPageQuery } from "./project-page-shell";
import { useProjectOverviewJobsQuery } from "./use-project-overview-jobs";
import {
  buildProjectOverviewTriageItems,
  formatProjectLocaleRoute,
  projectOverviewMeshTone,
  type ProjectOverviewTriageItem,
} from "./project-overview-view-model";

function buildProjectJobHref(organizationSlug: string, projectId: string, jobId: string) {
  return `/org/${organizationSlug}/projects/${encodeURIComponent(projectId)}/jobs/${encodeURIComponent(jobId)}`;
}

function resolveTriageJobMeta(
  job: ApiJob | undefined,
  intl: ReturnType<typeof useIntl>,
): string | null {
  if (!job) {
    return null;
  }

  const hasLocales = Boolean(job.externalTargetLocales?.length || job.reviewTargetLocale);
  const hasAssignees = Boolean(job.externalAssignedUsers?.length);
  if (!hasLocales && !hasAssignees) {
    return null;
  }

  return taskDetailSummary(job, intl);
}

function resolveTriageCopy(
  item: ProjectOverviewTriageItem,
  intl: ReturnType<typeof useIntl>,
): { title: string; description: string; meta: string | null; cta: string } {
  switch (item.kind) {
    case "review":
      return {
        title: getJobName(item.job!, intl),
        description: intl.formatMessage(messages.triageReviewTitle),
        meta: resolveTriageJobMeta(item.job, intl),
        cta: intl.formatMessage(messages.reviewCta),
      };
    case "failed":
      return {
        title: getJobName(item.job!, intl),
        description: intl.formatMessage(messages.triageFailedTitle),
        meta: resolveTriageJobMeta(item.job, intl),
        cta: intl.formatMessage(messages.openJobCta),
      };
    case "job":
      return {
        title: getJobName(item.job!, intl),
        description: intl.formatMessage(messages.triageJobRunning),
        meta: resolveTriageJobMeta(item.job, intl),
        cta: intl.formatMessage(messages.openJobCta),
      };
    case "guidance":
      return {
        title: intl.formatMessage(messages.triageGuidanceTitle),
        description: intl.formatMessage(messages.triageGuidanceDescription),
        meta: null,
        cta: intl.formatMessage(messages.addGuidanceCta),
      };
    default: {
      const _exhaustive: never = item.kind;
      return _exhaustive;
    }
  }
}

function TriageRow({
  href,
  title,
  description,
  meta,
  cta,
}: {
  href: string;
  title: string;
  description: string;
  meta: string | null;
  cta: string;
}) {
  return (
    <Link
      href={href}
      className="group flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background/70 px-4 py-3 transition-colors hover:border-border hover:bg-background/90"
    >
      <div className="min-w-0">
        <TypographyP lineClamp={1} size="small" weight="medium" tone="content">
          {title}
        </TypographyP>
        <TypographyP className="mt-0.5" size="xsmall" tone="subtle">
          {description}
        </TypographyP>
        {meta ? (
          <TypographyP className="mt-0.5" lineClamp={1} size="xsmall" tone="subtle">
            {meta}
          </TypographyP>
        ) : null}
      </div>
      <span className="inline-flex shrink-0 items-center gap-1 text-sm font-medium text-foreground">
        {cta}
        <HugeiconsIcon
          icon={ArrowRight01Icon}
          strokeWidth={1.8}
          className="size-4 transition-transform group-hover:translate-x-0.5"
        />
      </span>
    </Link>
  );
}

export type ProjectOverviewPageContentViewProps = {
  organizationSlug: string;
  projectId: string;
  project: ProjectListRow | null;
  isProjectLoading: boolean;
  isProjectError: boolean;
  jobs: readonly ApiJob[];
  isJobsLoading: boolean;
  isJobsError: boolean;
  onCreateJob?: () => void;
};

export function ProjectOverviewPageContentView({
  organizationSlug,
  projectId,
  project,
  isProjectLoading,
  isProjectError,
  jobs,
  isJobsLoading,
  isJobsError,
  onCreateJob,
}: ProjectOverviewPageContentViewProps) {
  const intl = useIntl();
  const isNative = project?.source === "native";
  const hasTranslationGuidance = Boolean(project?.translationContextValue?.trim());
  const showViewStrings = supportsContentEditorAllFilesProvider(
    parseProviderProjectId(projectId)?.providerKind,
  );

  const triageItems = project
    ? buildProjectOverviewTriageItems({
        jobs,
        isNative: isNative ?? false,
        hasTranslationGuidance,
      })
    : [];
  const meshTone = projectOverviewMeshTone(triageItems.length);

  const projectDescription =
    project?.descriptionValue || intl.formatMessage(messages.defaultProjectDescription);

  const settingsHref = buildProjectPath(organizationSlug, projectId, "settings");
  const filesHref = buildProjectPath(organizationSlug, projectId, "files");
  const jobsHref = buildProjectPath(organizationSlug, projectId, "jobs");
  const showHeaderActions = Boolean(project) && !isProjectLoading && !isProjectError;
  const localeRoute = project
    ? formatProjectLocaleRoute(project.sourceLocale, project.targetLocales)
    : "";

  return (
    <ProjectPageShell className="gap-8">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-2">
          {isProjectLoading ? (
            <>
              <Skeleton className="h-8 w-64" />
              <Skeleton className="h-4 w-full max-w-xl" />
            </>
          ) : isProjectError ? (
            <>
              <TypographyH1 className="text-2xl" weight="medium" tone="content">
                <FormattedMessage {...messages.projectOverviewFallbackTitle} />
              </TypographyH1>
              <TypographyP size="small" tone="subtle">
                <FormattedMessage {...messages.loadProjectError} />
              </TypographyP>
            </>
          ) : (
            <>
              <TypographyH1 className="text-2xl" weight="medium" tone="content">
                {project?.name ?? intl.formatMessage(messages.projectFallbackName)}
              </TypographyH1>
              <TypographyP className="max-w-2xl leading-6" size="small" tone="subtle">
                {projectDescription}
              </TypographyP>
            </>
          )}
        </div>

        {showHeaderActions ? (
          <div className="flex shrink-0 flex-wrap gap-2">
            {showViewStrings ? (
              <Button
                nativeButton={false}
                render={<Link href={buildProjectPath(organizationSlug, projectId, "strings")} />}
                size="sm"
                variant="outline"
              >
                <HugeiconsIcon icon={LanguageCircleIcon} strokeWidth={1.8} />
                <FormattedMessage {...messages.openEditor} />
              </Button>
            ) : null}
            <Button
              nativeButton={false}
              render={<Link href={filesHref} />}
              size="sm"
              variant="outline"
            >
              <FormattedMessage {...messages.viewFiles} />
            </Button>
            <Button type="button" size="sm" onClick={onCreateJob}>
              <HugeiconsIcon icon={Add01Icon} strokeWidth={1.8} />
              <FormattedMessage {...messages.createJob} />
            </Button>
          </div>
        ) : null}
      </header>

      {isProjectLoading || isJobsLoading ? (
        <Skeleton className="min-h-56 rounded-2xl" />
      ) : project ? (
        <ProjectOverviewMeshStage tone={meshTone}>
          <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <TypographyP className="font-heading" size="xlarge" weight="medium" tone="content">
                <FormattedMessage {...messages.needsYouNowTitle} />
              </TypographyP>
              {triageItems.length > 0 ? (
                <TypographyP size="small" tone="subtle">
                  <FormattedMessage
                    {...messages.needsYouNowCount}
                    values={{ count: triageItems.length }}
                  />
                </TypographyP>
              ) : null}
            </div>

            {triageItems.length > 0 ? (
              <div className="flex flex-col gap-2">
                {triageItems.map((item) => {
                  const copy = resolveTriageCopy(item, intl);
                  const href =
                    item.kind === "guidance"
                      ? settingsHref
                      : item.job
                        ? buildProjectJobHref(organizationSlug, projectId, item.job.id)
                        : jobsHref;

                  return (
                    <TriageRow
                      key={item.id}
                      href={href}
                      title={copy.title}
                      description={copy.description}
                      meta={copy.meta}
                      cta={copy.cta}
                    />
                  );
                })}
                <div className="pt-1">
                  <Button
                    nativeButton={false}
                    render={<Link href={jobsHref} />}
                    variant="ghost"
                    size="sm"
                    className="w-fit px-0"
                  >
                    <FormattedMessage {...messages.viewAllJobs} />
                    <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={1.8} />
                  </Button>
                </div>
              </div>
            ) : isJobsError ? (
              <div className="rounded-xl border border-dashed border-border bg-background/60 px-4 py-4">
                <TypographyP size="small" weight="medium" tone="content">
                  <FormattedMessage {...messages.jobsUnavailable} />
                </TypographyP>
                <TypographyP className="mt-1" size="small" tone="subtle">
                  <FormattedMessage {...messages.jobsUnavailableDescription} />
                </TypographyP>
                <Button
                  nativeButton={false}
                  render={<Link href={jobsHref} />}
                  variant="outline"
                  size="sm"
                  className="mt-3 w-fit"
                >
                  <FormattedMessage {...messages.viewJobs} />
                </Button>
              </div>
            ) : (
              <div className="space-y-2">
                <TypographyP size="small" weight="medium" tone="content">
                  <FormattedMessage {...messages.triageEmptyTitle} />
                </TypographyP>
                <TypographyP size="small" tone="subtle">
                  <FormattedMessage {...messages.triageEmptyDescription} />
                </TypographyP>
              </div>
            )}
          </div>
        </ProjectOverviewMeshStage>
      ) : null}

      {project && !isProjectLoading ? (
        <section className="space-y-4">
          <OverviewSectionHeader title={intl.formatMessage(messages.signalsTitle)} />
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-2xl border border-border px-5 py-4">
              <TypographyP size="xsmall" weight="medium" tone="subtle" capitalization="uppercase">
                <FormattedMessage {...messages.signalsLocales} />
              </TypographyP>
              <TypographyP className="mt-2 font-mono" size="small" tone="content">
                {project.targetLocales.length > 0 ? (
                  localeRoute
                ) : (
                  <FormattedMessage {...messages.signalsNoLocales} />
                )}
              </TypographyP>
              {project.targetLocales.length === 0 ? (
                <Button
                  nativeButton={false}
                  render={<Link href={settingsHref} />}
                  variant="ghost"
                  size="sm"
                  className="mt-2 px-0"
                >
                  <FormattedMessage {...messages.viewSettings} />
                </Button>
              ) : null}
            </div>

            {isNative ? (
              <div className="rounded-2xl border border-border px-5 py-4">
                <TypographyP size="xsmall" weight="medium" tone="subtle" capitalization="uppercase">
                  <FormattedMessage {...messages.shipTitle} />
                </TypographyP>
                <TypographyP className="mt-2" size="small" tone="content">
                  {project.lastSyncedAt ? (
                    <FormattedMessage
                      {...messages.shipLastSynced}
                      values={{ when: project.lastSyncedAt }}
                    />
                  ) : (
                    <FormattedMessage {...messages.shipNeverSynced} />
                  )}
                </TypographyP>
                <TypographyP className="mt-1" size="small" tone="subtle">
                  <FormattedMessage
                    {...messages.shipCliHint}
                    values={{
                      code: (chunks: ReactNode) => (
                        <span className="font-mono text-foreground">{chunks}</span>
                      ),
                    }}
                  />
                </TypographyP>
                <Button
                  nativeButton={false}
                  render={<Link href={settingsHref} />}
                  variant="ghost"
                  size="sm"
                  className="mt-2 px-0"
                >
                  <FormattedMessage {...messages.shipConnectCli} />
                </Button>
              </div>
            ) : null}
          </div>
        </section>
      ) : null}

      {isNative && hasTranslationGuidance ? (
        <section className="space-y-3 rounded-2xl border border-border bg-muted/30 px-5 py-5">
          <div className="flex items-center justify-between gap-3">
            <TypographyP size="small" weight="medium" tone="content">
              <FormattedMessage {...messages.guidanceTitle} />
            </TypographyP>
            <Button
              nativeButton={false}
              render={<Link href={settingsHref} />}
              variant="ghost"
              size="sm"
              className="shrink-0"
            >
              <FormattedMessage {...messages.guidanceEdit} />
            </Button>
          </div>
          <MarkdownPreview
            value={project?.translationContextValue ?? ""}
            chrome="minimal"
            className="line-clamp-6"
            contentClassName="text-sm leading-6 text-subtle-foreground"
          />
        </section>
      ) : null}
    </ProjectPageShell>
  );
}

export function ProjectOverviewPageContent({
  organizationSlug,
  projectId,
}: {
  organizationSlug: string;
  projectId: string;
}) {
  const [createJobOpen, setCreateJobOpen] = useState(false);
  const projectQuery = useProjectPageQuery(organizationSlug, projectId);
  const jobsQuery = useProjectOverviewJobsQuery(organizationSlug, projectId, {
    enabled: projectQuery.isSuccess,
  });

  const sourceLocale = projectQuery.data?.sourceLocale?.trim() || "en";
  const targetLocales = projectQuery.data?.targetLocales ?? [];

  return (
    <>
      <ProjectOverviewPageContentView
        organizationSlug={organizationSlug}
        projectId={projectId}
        project={projectQuery.data ?? null}
        isProjectLoading={projectQuery.isLoading}
        isProjectError={projectQuery.isError}
        jobs={jobsQuery.data ?? []}
        isJobsLoading={jobsQuery.isLoading}
        isJobsError={jobsQuery.isError}
        onCreateJob={() => setCreateJobOpen(true)}
      />
      <CreateJobDialog
        open={createJobOpen}
        onOpenChange={setCreateJobOpen}
        organizationSlug={organizationSlug}
        projectId={projectId}
        sourceLocale={sourceLocale}
        targetLocales={targetLocales}
        onCreated={async () => {
          await jobsQuery.refetch();
        }}
      />
    </>
  );
}

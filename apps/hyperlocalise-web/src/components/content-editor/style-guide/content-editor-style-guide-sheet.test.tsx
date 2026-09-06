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
// @vitest-environment happy-dom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { IntlProvider } from "react-intl";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";

import type { ProjectListRow } from "@/app/[lang]/(authenticated)/org/[organizationSlug]/projects/_components/project-list";

const { useProjectPageQueryMock } = vi.hoisted(() => ({
  useProjectPageQueryMock: vi.fn(),
}));

vi.mock(
  "@/app/[lang]/(authenticated)/org/[organizationSlug]/projects/[projectId]/_components/project-page-shell",
  () => ({
    useProjectPageQuery: (...args: unknown[]) => useProjectPageQueryMock(...args),
  }),
);

vi.mock("@/components/markdown-editor/markdown-editor", () => ({
  MarkdownPreview: ({ value, emptyMessage }: { value: string; emptyMessage?: string }) => (
    <div>{value.trim() ? value : emptyMessage}</div>
  ),
}));

import { ContentEditorStyleGuideSheet } from "./content-editor-style-guide-sheet";

function createProject(overrides: Partial<ProjectListRow> = {}): ProjectListRow {
  return {
    id: "project_1",
    name: "Tourmatic",
    key: "TM",
    identifier: "TM",
    description: "No description",
    descriptionValue: "",
    translationContext: "Keep product names in English.",
    translationContextValue: "Keep product names in English.",
    created: "Apr 29, 2026",
    updated: "Apr 30, 2026",
    source: "native",
    externalProviderKind: null,
    externalProjectId: null,
    sourceLocale: "en-US",
    targetLocales: ["fr-FR"],
    externalProjectUrl: null,
    isActive: true,
    logoUrl: null,
    lastActivityAt: null,
    lastSyncedAt: null,
    lastSyncErrorAt: null,
    lastSyncErrorMessage: null,
    openJobCount: 0,
    ...overrides,
  };
}

function renderSheet(project: ProjectListRow | undefined, options?: { isLoading?: boolean; isError?: boolean }) {
  useProjectPageQueryMock.mockReturnValue({
    isLoading: options?.isLoading ?? false,
    isFetching: options?.isLoading ?? false,
    isError: options?.isError ?? false,
    isSuccess: Boolean(project),
    data: project,
    error: null,
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={new QueryClient()}>
        <IntlProvider locale="en" messages={{}}>
          {children}
        </IntlProvider>
      </QueryClientProvider>
    );
  }

  return render(
    <ContentEditorStyleGuideSheet
      organizationSlug="acme"
      projectId="project_1"
      open
      onOpenChange={vi.fn()}
    />,
    { wrapper: Wrapper },
  );
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("ContentEditorStyleGuideSheet", () => {
  it("renders markdown style guide content and a settings link for native projects", () => {
    renderSheet(createProject());

    expect(screen.getByRole("dialog", { name: "Style guide" })).toBeTruthy();
    expect(screen.getByText("Keep product names in English.")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Edit in settings" })).toHaveAttribute(
      "href",
      "/org/acme/projects/project_1/settings",
    );
  });

  it("shows an empty state when the project has no style guide", () => {
    renderSheet(
      createProject({
        translationContext: "No translation context",
        translationContextValue: "",
      }),
    );

    expect(screen.getByText("No style guide yet. Add one in project settings.")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Edit in settings" })).toBeTruthy();
  });

  it("hides the settings link for provider-managed projects", () => {
    renderSheet(createProject({ source: "external_tms" }));

    expect(screen.getByText("Keep product names in English.")).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Edit in settings" })).toBeNull();
  });

  it("shows a loading state while the project is fetching", () => {
    renderSheet(undefined, { isLoading: true });

    expect(screen.getByText("Loading style guide...")).toBeTruthy();
  });

  it("shows an error when the project fails to load", () => {
    renderSheet(undefined, { isError: true });

    expect(screen.getByText("Unable to load the style guide.")).toBeTruthy();
  });
});

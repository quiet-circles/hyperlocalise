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
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { IntlProvider } from "react-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";

import { AppShellHeaderActions } from "@/components/app-shell/store/app-shell-header-actions";
import { AppShellStoreProvider } from "@/components/app-shell/store/app-shell-store-context";

import type { ProjectListRow } from "../../../_components/project-list";

const { useProjectPageQueryMock, patchMock, toastErrorMock, toastSuccessMock } = vi.hoisted(() => ({
  useProjectPageQueryMock: vi.fn(),
  patchMock: vi.fn(),
  toastErrorMock: vi.fn(),
  toastSuccessMock: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  usePathname: () => "/en/org/acme/projects/project_1/settings",
}));

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccessMock(...args),
    error: (...args: unknown[]) => toastErrorMock(...args),
  },
}));

vi.mock("../../_components/project-page-shell", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../_components/project-page-shell")>();
  return {
    ...actual,
    useProjectPageQuery: (...args: unknown[]) => useProjectPageQueryMock(...args),
  };
});

vi.mock("@/lib/api-client-instance", () => ({
  apiClient: {
    api: {
      orgs: {
        ":organizationSlug": {
          projects: {
            ":projectId": {
              $patch: (...args: unknown[]) => patchMock(...args),
            },
          },
        },
      },
    },
  },
}));

vi.mock("./project-issue-templates-panel", () => ({
  ProjectIssueTemplatesPanel: () => null,
}));

vi.mock("./project-native-connect-cli-panel", () => ({
  ProjectNativeConnectCliPanel: () => null,
}));

vi.mock("./project-content-editor-behavior-settings", () => ({
  ProjectContentEditorBehaviorSettings: () => null,
}));

vi.mock("./project-issue-columns-settings", () => ({
  ProjectIssueColumnsSettings: () => null,
}));

vi.mock("@/components/markdown-editor/markdown-editor", () => ({
  MarkdownEditor: ({
    id,
    value,
    onChange,
    ariaLabel,
    disabled,
  }: {
    id?: string;
    value: string;
    onChange: (next: string) => void;
    ariaLabel?: string;
    disabled?: boolean;
  }) => (
    <textarea
      id={id}
      aria-label={ariaLabel}
      value={value}
      disabled={disabled}
      onChange={(event) => onChange(event.currentTarget.value)}
    />
  ),
}));

import { ProjectSettingsPageContent } from "./project-settings-page-content";

function createProject(overrides: Partial<ProjectListRow> = {}): ProjectListRow {
  return {
    id: "project_1",
    name: "Tourmatic",
    key: "TM",
    identifier: "TM",
    description: "No description",
    descriptionValue: "",
    translationContext: "No translation context",
    translationContextValue: "",
    created: "Apr 29, 2026",
    updated: "Apr 30, 2026",
    source: "native",
    externalProviderKind: null,
    externalProjectId: null,
    sourceLocale: "en-US",
    targetLocales: ["fr-FR", "de-DE"],
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

function mockProjectQuery(project: ProjectListRow | undefined, options?: { isLoading?: boolean }) {
  useProjectPageQueryMock.mockReturnValue({
    isLoading: options?.isLoading ?? false,
    isError: false,
    isSuccess: Boolean(project),
    data: project,
    error: null,
  });
}

function renderSettings(project: ProjectListRow = createProject()) {
  mockProjectQuery(project);
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <IntlProvider locale="en" messages={{}}>
          <AppShellStoreProvider defaultNavigationGroups={[]}>
            <div data-testid="header-actions">
              <AppShellHeaderActions />
            </div>
            {children}
          </AppShellStoreProvider>
        </IntlProvider>
      </QueryClientProvider>
    );
  }

  return render(
    <ProjectSettingsPageContent
      organizationSlug="acme"
      projectId={project.id}
      canManageCatBehavior
    />,
    { wrapper: Wrapper },
  );
}

afterEach(() => {
  vi.clearAllMocks();
});

beforeEach(() => {
  patchMock.mockResolvedValue({
    ok: true,
    json: async () => ({ project: createProject({ identifier: "NEW" }) }),
  });
});

describe("ProjectSettingsPageContent", () => {
  it("saves an updated identifier via the header Save settings action", async () => {
    const user = userEvent.setup();
    renderSettings();

    const identifier = await screen.findByLabelText("Identifier");
    await user.clear(identifier);
    await user.type(identifier, "new");

    await user.click(screen.getByRole("button", { name: "Save settings" }));

    await waitFor(() => expect(patchMock).toHaveBeenCalled());
    expect(patchMock.mock.calls[0]?.[0]).toMatchObject({
      param: { organizationSlug: "acme", projectId: "project_1" },
      json: expect.objectContaining({ identifier: "NEW" }),
    });
    await waitFor(() => expect(toastSuccessMock).toHaveBeenCalledWith("Project settings saved"));
  });

  it("shows a toast and skips PATCH when the identifier is invalid", async () => {
    const user = userEvent.setup();
    renderSettings();

    const identifier = await screen.findByLabelText("Identifier");
    await user.clear(identifier);
    await user.type(identifier, "123");

    await user.click(screen.getByRole("button", { name: "Save settings" }));

    await waitFor(() =>
      expect(toastErrorMock).toHaveBeenCalledWith(
        "Use 1–10 letters or numbers, starting with a letter (e.g. HL).",
      ),
    );
    expect(patchMock).not.toHaveBeenCalled();
    expect(
      screen.getByText("Use 1–10 letters or numbers, starting with a letter (e.g. HL)."),
    ).toBeInTheDocument();
  });

  it("does not wipe in-progress identifier edits when the same project data is refetched", async () => {
    const user = userEvent.setup();
    const project = createProject();
    const view = renderSettings(project);

    const identifier = await screen.findByLabelText("Identifier");
    await user.clear(identifier);
    await user.type(identifier, "edit");
    expect(identifier).toHaveValue("EDIT");

    mockProjectQuery({ ...project });
    view.rerender(
      <ProjectSettingsPageContent
        organizationSlug="acme"
        projectId={project.id}
        canManageCatBehavior
      />,
    );

    expect(await screen.findByLabelText("Identifier")).toHaveValue("EDIT");
  });

  it("hides Save settings while the project is still loading", () => {
    mockProjectQuery(undefined, { isLoading: true });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <IntlProvider locale="en" messages={{}}>
          <AppShellStoreProvider defaultNavigationGroups={[]}>
            <AppShellHeaderActions />
            <ProjectSettingsPageContent
              organizationSlug="acme"
              projectId="project_1"
              canManageCatBehavior
            />
          </AppShellStoreProvider>
        </IntlProvider>
      </QueryClientProvider>,
    );

    expect(screen.queryByRole("button", { name: "Save settings" })).not.toBeInTheDocument();
    expect(screen.getByText("Loading project settings...")).toBeInTheDocument();
  });

  it("saves markdown style guide content as translation context", async () => {
    const user = userEvent.setup();
    renderSettings();

    const styleGuide = await screen.findByLabelText("Style guide");
    expect(styleGuide).toHaveAttribute("id", "translation-context");
    await user.clear(styleGuide);
    await user.type(styleGuide, "Keep product names in English.");

    await user.click(screen.getByRole("button", { name: "Save settings" }));

    await waitFor(() => expect(patchMock).toHaveBeenCalled());
    expect(patchMock.mock.calls[0]?.[0]).toMatchObject({
      json: expect.objectContaining({
        translationContext: "Keep product names in English.",
      }),
    });
  });
});

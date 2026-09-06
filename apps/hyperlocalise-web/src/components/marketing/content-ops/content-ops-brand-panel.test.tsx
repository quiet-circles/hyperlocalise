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
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { IntlProvider } from "react-intl";
import { describe, expect, it, vi } from "vite-plus/test";

import { ContentOpsBrandPanel } from "./content-ops-brand-panel";

describe("ContentOpsBrandPanel", () => {
  it("plays the brand review transcript after send", async () => {
    vi.useFakeTimers();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(
      <IntlProvider locale="en" messages={{}}>
        <ContentOpsBrandPanel autoStart={false} />
      </IntlProvider>,
    );

    expect(screen.getByText("Ask about brand voice")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Send" }));

    expect(
      screen.getByText(
        'Does this German CTA follow our brand guidelines? "Nutzen Sie unsere innovative Plattform"',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("Ask about brand voice")).not.toBeInTheDocument();

    act(() => {
      vi.runAllTimers();
    });

    expect(
      screen.getByText("Off-brand — too formal and jargon-heavy for DE checkout."),
    ).toBeInTheDocument();
    expect(screen.getByText("Jetzt starten")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Replay" })).toBeInTheDocument();

    vi.useRealTimers();
  });

  it("starts playback from the empty-state suggestion", async () => {
    vi.useFakeTimers();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(
      <IntlProvider locale="en" messages={{}}>
        <ContentOpsBrandPanel autoStart={false} />
      </IntlProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Review this German CTA" }));

    expect(screen.queryByText("Ask about brand voice")).not.toBeInTheDocument();
    expect(screen.queryByText("Jetzt starten")).not.toBeInTheDocument();

    act(() => {
      vi.runAllTimers();
    });

    expect(screen.getByText("Jetzt starten")).toBeInTheDocument();

    vi.useRealTimers();
  });
});

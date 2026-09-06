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
import { render, screen } from "@testing-library/react";
import { IntlProvider } from "react-intl";
import { describe, expect, it } from "vite-plus/test";

import { ConnectedCampaign } from "./connected-campaign";

describe("ConnectedCampaign", () => {
  it("renders harbour-coloured capability cards in a carousel", () => {
    render(
      <IntlProvider locale="en" messages={{}}>
        <ConnectedCampaign />
      </IntlProvider>,
    );

    expect(
      screen.getByRole("region", { name: "Ask Hyperlocalise to take your content global." }),
    ).toBeInTheDocument();
    expect(screen.getByText("What can Hyperlocalise do?")).toBeInTheDocument();
    expect(screen.getByText("Launch this release everywhere")).toBeInTheDocument();
    expect(screen.getByText("Take this campaign global")).toBeInTheDocument();
    expect(screen.getByText("Grow our presence in Japan")).toBeInTheDocument();
    expect(screen.getByText("Keep everything on-brand and compliant")).toBeInTheDocument();
    expect(screen.getByText("Launch this release everywhere").closest("article")).toHaveClass(
      "bg-fog",
      "text-ink",
    );
    expect(screen.getByText("Take this campaign global").closest("article")).toHaveClass(
      "bg-steel",
      "text-ink",
    );
    expect(screen.getByText("Grow our presence in Japan").closest("article")).toHaveClass(
      "bg-navy",
      "text-fog",
    );
    expect(screen.getByRole("button", { name: "Next slide" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Previous slide" })).toBeInTheDocument();
  });
});

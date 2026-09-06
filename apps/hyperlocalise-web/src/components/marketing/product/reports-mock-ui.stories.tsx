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
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect } from "storybook/test";

import { ReportsMockUI } from "./reports-mock-ui";

function ReportsMockShowcase({ variant = "full" }: { variant?: "full" | "embedded" }) {
  return (
    <div className="mx-auto max-w-6xl p-6">
      <ReportsMockUI variant={variant} />
    </div>
  );
}

const meta = {
  title: "Marketing/Product/ReportsMock",
  component: ReportsMockShowcase,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof ReportsMockShowcase>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("heading", {
        name: "Word counts, time, and cost in one workspace view",
      }),
    ).toBeInTheDocument();
    await expect(canvas.getByRole("button", { name: "Word counts" })).toBeInTheDocument();
    await expect(canvas.getByRole("button", { name: "Spend" })).toBeInTheDocument();
    await expect(canvas.getByRole("button", { name: "Time" })).toBeInTheDocument();
    await expect(canvas.getByText("Q3 localisation volume")).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Request a Demo" })).toBeInTheDocument();
  },
};

export const Embedded: Story = {
  render: () => <ReportsMockShowcase variant="embedded" />,
  play: async ({ canvas }) => {
    await expect(canvas.getByText("Q3 localisation volume")).toBeInTheDocument();
    await expect(canvas.getByText("Source words")).toBeInTheDocument();
    await expect(canvas.queryByRole("link", { name: "Request a Demo" })).not.toBeInTheDocument();
  },
};

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
import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect } from "storybook/test";

import { FeatureTeaserPage } from "./feature-teaser-page";

function FeatureTeaserShowcase(props: ComponentProps<typeof FeatureTeaserPage>) {
  return (
    <div className="bg-background px-6 py-8 text-foreground">
      <FeatureTeaserPage {...props} />
    </div>
  );
}

const meta = {
  title: "App/FeatureTeaser",
  component: FeatureTeaserShowcase,
  parameters: {
    layout: "fullscreen",
    nextjs: {
      navigation: {
        pathname: "/org/acme/automations",
      },
    },
  },
} satisfies Meta<typeof FeatureTeaserShowcase>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Automations: Story = {
  args: {
    feature: "automations",
    scope: "workspace",
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("heading", { name: "Automations" })).toBeInTheDocument();
    await expect(canvas.getAllByText("Preview").length).toBeGreaterThan(0);
    await expect(
      canvas.getByText("Ship to more markets without growing the team"),
    ).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Request a demo" })).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Contact us" })).toBeInTheDocument();
    await expect(canvas.getByText("GTM brief approved · Q2 launch")).toBeInTheDocument();
  },
};

export const Guideline: Story = {
  args: {
    feature: "guideline",
    scope: "workspace",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/knowledge",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("heading", { name: "Guideline" })).toBeInTheDocument();
    await expect(
      canvas.getByText("One playbook for global growth in every market"),
    ).toBeInTheDocument();
    await expect(canvas.getByText("Market: Germany")).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Request a demo" })).toHaveAttribute(
      "href",
      "https://calendar.app.google/gEiRwNvAZ1ERXvT26",
    );
  },
};

export const Domains: Story = {
  args: {
    feature: "domains",
    scope: "workspace",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/domains",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("heading", { name: "Domains" })).toBeInTheDocument();
    await expect(
      canvas.getByText("See what is hurting search and AI answers in every locale"),
    ).toBeInTheDocument();
    await expect(canvas.getByText("Website audit · acme.com")).toBeInTheDocument();
    await expect(
      canvas.getByText("Find hreflang errors, missing locales, and content gaps"),
    ).toBeInTheDocument();
  },
};

export const ProjectAutomations: Story = {
  args: {
    feature: "automations",
    scope: "project",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/projects/proj_1/automations",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByText("Project")).toBeInTheDocument();
    await expect(
      canvas.getByText("Keep this project releasing on time without chasing manual handoffs."),
    ).toBeInTheDocument();
  },
};

export const ProjectGuideline: Story = {
  args: {
    feature: "guideline",
    scope: "project",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/projects/proj_1/knowledge",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByText("Project")).toBeInTheDocument();
    await expect(
      canvas.getByText(
        "Give this project the GTM context it needs to launch and grow in new markets.",
      ),
    ).toBeInTheDocument();
  },
};

export const Hyperlab: Story = {
  args: {
    feature: "hyperlab",
    scope: "workspace",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/hyperlab",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("heading", { name: "Hyperlab" })).toBeInTheDocument();
    await expect(
      canvas.getByText("Target, split, and ship without another vendor"),
    ).toBeInTheDocument();
    await expect(canvas.getByText("checkout-cta")).toBeInTheDocument();
    await expect(
      canvas.getByText("Create experiment and config flags unique to your workspace"),
    ).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Request a demo" })).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Contact us" })).toBeInTheDocument();
  },
};

export const Reports: Story = {
  args: {
    feature: "reports",
    scope: "workspace",
  },
  parameters: {
    nextjs: {
      navigation: {
        pathname: "/org/acme/reports",
      },
    },
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("heading", { name: "Reports" })).toBeInTheDocument();
    await expect(
      canvas.getByText("Know what translation is costing before the invoice arrives"),
    ).toBeInTheDocument();
    await expect(canvas.getByText("Q3 localisation volume")).toBeInTheDocument();
    await expect(
      canvas.getByText("Track words, time, and cost by project and locale"),
    ).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Request a demo" })).toBeInTheDocument();
    await expect(canvas.getByRole("link", { name: "Contact us" })).toBeInTheDocument();
  },
};

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
import { homepageMessages as m } from "./homepage.messages";

/** Hosted hero preview video. Empty until a URL is available. */
export const PRODUCT_PREVIEW_VIDEO_URL = "";

export function hasProductPreviewVideoUrl(url: string | null | undefined): boolean {
  return typeof url === "string" && url.trim().length > 0;
}

export const PRODUCTS = [
  {
    id: "studio",
    title: m.studioCardTitle,
    short: m.studioShort,
    body: m.studioBody,
    mark: "Aa",
  },
  {
    id: "automation",
    title: m.automationCardTitle,
    short: m.automationShort,
    body: m.automationBody,
    mark: "↳",
  },
  {
    id: "domains",
    title: m.domainsCardTitle,
    short: m.domainsShort,
    body: m.domainsBody,
    mark: "◎",
  },
  {
    id: "hyperlab",
    title: m.hyperlabCardTitle,
    short: m.hyperlabShort,
    body: m.hyperlabBody,
    mark: "A/B",
  },
  {
    id: "guidelines",
    title: m.guidelinesCardTitle,
    short: m.guidelinesShort,
    body: m.guidelinesBody,
    mark: "≋",
  },
] as const;

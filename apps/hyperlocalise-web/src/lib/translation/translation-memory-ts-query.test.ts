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
import { describe, expect, it } from "vite-plus/test";

import {
  buildReportingMemoryMatchTsQuery,
  buildTranslationMemoryTsQuery,
} from "./translation-memory-ts-query";

describe("buildTranslationMemoryTsQuery", () => {
  it("strips operators and builds prefix AND terms", () => {
    expect(buildTranslationMemoryTsQuery(`save & (now)!`)).toBe("save:* & now:*");
  });

  it("returns an empty string when no tokens remain", () => {
    expect(buildTranslationMemoryTsQuery("&&&")).toBe("");
  });
});

describe("buildReportingMemoryMatchTsQuery", () => {
  it("ORs terms so added words still retrieve shorter memory entries", () => {
    expect(buildReportingMemoryMatchTsQuery("Hello brave world")).toBe(
      "Hello:* | brave:* | world:*",
    );
  });

  it("adds shorter prefixes for unspaced tokens", () => {
    expect(buildReportingMemoryMatchTsQuery("日本語の翻訳")).toContain("日本語の翻:*");
    expect(buildReportingMemoryMatchTsQuery("日本語の翻訳")).toContain("日本語の:*");
  });
});

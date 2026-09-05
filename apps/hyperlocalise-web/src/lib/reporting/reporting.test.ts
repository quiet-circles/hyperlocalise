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
import { countSourceWords, matchBucket, reportingPeriod, csvCell } from "./word-analysis";
import { usdUnits, usdString, hourlyCost, wordCost, budgetWarning } from "./money";
import { reportQuerySchema, rateSchema, timeSchema } from "@/api/routes/reports/reports.schema";

describe("prospective translation accounting", () => {
  it("counts literal ICU alternatives without counting arguments or tags", () => {
    expect(countSourceWords("<b>Hello</b> {name}!", "en")).toBe(1);
    expect(countSourceWords("{count, plural, one {One file} other {Many files}}", "en")).toBe(4);
    expect(countSourceWords("Hi {{name}} %s ${value}", "en")).toBe(1);
    expect(countSourceWords("", "en")).toBe(0);
    expect(countSourceWords("!!!", "en")).toBe(0);
    expect(countSourceWords("日本語の翻訳", "ja")).toBeGreaterThan(0);
    expect(countSourceWords("{unclosed", "en")).toBeNull();
  });
  it.each([
    [100, "100"],
    [99, "95-99"],
    [95, "95-99"],
    [94, "85-94"],
    [85, "85-94"],
    [84, "75-84"],
    [75, "75-84"],
    [74, "50-74"],
    [50, "50-74"],
    [49, "new"],
    [0, "new"],
    [null, "unavailable"],
    [101, "unavailable"],
  ] as const)("classifies %s", (score, expected) => expect(matchBucket(score)).toBe(expected));
  it("gives repetitions precedence and never invents unavailable matches", () => {
    expect(matchBucket(100, true)).toBe("repetition");
    expect(matchBucket(null)).toBe("unavailable");
  });
  it("keeps decimal costs exact", () => {
    expect(usdString(usdUnits("0.10") + usdUnits("0.20"))).toBe("0.30000000");
    expect(wordCost("0.12", 1000, 25)).toBe("30.00000000");
    expect(hourlyCost("60", 15)).toBe("15.00000000");
    expect(wordCost("0.00000001", 1)).toBe("0.00000001");
    expect(() => usdUnits("-1")).toThrow();
  });
  it("warns at 80 and 100 percent without blocking work", () => {
    expect(budgetWarning("100", "79.99")).toBe("none");
    expect(budgetWarning("100", "80")).toBe("approaching");
    expect(budgetWarning("100", "100")).toBe("exceeded");
  });
  it("groups UTC weeks on Mondays across year boundaries", () => {
    expect(reportingPeriod(new Date("2026-01-04T23:59:59Z"), "week")).toBe("2025-12-29");
    expect(reportingPeriod(new Date("2026-01-05T00:00:00Z"), "week")).toBe("2026-01-05");
  });
  it("escapes CSV and neutralizes spreadsheet formulas", () => {
    expect(csvCell('a,"b"')).toBe('"a,""b"""');
    expect(csvCell('=IMPORTXML("url")')).toContain("'=IMPORTXML");
  });
  it("rejects invalid date ranges, time and rate percentages", () => {
    expect(reportQuerySchema.safeParse({ from: "2026-09-06", to: "2026-09-01" }).success).toBe(
      false,
    );
    expect(
      timeSchema.safeParse({ jobId: "job", step: "review", minutes: 0, workDate: "2026-09-01" })
        .success,
    ).toBe(false);
    expect(
      rateSchema.safeParse({
        name: "Default",
        sourceLocale: "en",
        targetLocale: "fr",
        step: "translation",
        basis: "word",
        rate: "0.1",
        percentages: { "100": 101 },
      }).success,
    ).toBe(false);
  });
});

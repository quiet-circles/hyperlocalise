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
import { parse, TYPE, type MessageFormatElement } from "@formatjs/icu-messageformat-parser";

export const WORD_COUNT_VERSION = 1;
export const MATCH_BUCKETS = [
  "repetition",
  "100",
  "95-99",
  "85-94",
  "75-84",
  "50-74",
  "new",
  "unavailable",
] as const;
export type MatchBucket = (typeof MATCH_BUCKETS)[number];

function literals(elements: MessageFormatElement[]): string {
  return elements
    .map((element) => {
      if (element.type === TYPE.literal) return element.value;
      if (element.type === TYPE.tag) return literals(element.children);
      if (element.type === TYPE.select || element.type === TYPE.plural) {
        return Object.values(element.options)
          .map((option) => literals(option.value))
          .join(" ");
      }
      return " ";
    })
    .join(" ");
}

export function countSourceWords(source: string, locale: string): number | null {
  const cleaned = source
    .replace(/<[^>]*>/g, " ")
    .replace(/\{\{[^}]*\}\}|%\d*\$?[sdif]|\$\{[^}]*\}/g, " ");
  let text: string;
  try {
    text = literals(parse(cleaned, { ignoreTag: true }));
  } catch {
    return null;
  }
  try {
    return Array.from(new Intl.Segmenter(locale, { granularity: "word" }).segment(text)).filter(
      (part) => part.isWordLike,
    ).length;
  } catch {
    return null;
  }
}

export function matchBucket(score: number | null, repetition = false): MatchBucket {
  if (repetition) return "repetition";
  if (score === null || !Number.isFinite(score) || score < 0 || score > 100) return "unavailable";
  if (score === 100) return "100";
  if (score >= 95) return "95-99";
  if (score >= 85) return "85-94";
  if (score >= 75) return "75-84";
  if (score >= 50) return "50-74";
  return "new";
}

export function reportingPeriod(date: Date, interval: "day" | "week") {
  const result = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
  if (interval === "week") result.setUTCDate(result.getUTCDate() - ((result.getUTCDay() + 6) % 7));
  return result.toISOString().slice(0, 10);
}

export function csvCell(value: string | number | boolean | null | undefined): string {
  const text = String(value ?? "");
  const safe = /^[=+@\-\t\r]/.test(text) ? "'" + text : text;
  return '"' + safe.replaceAll('"', '""') + '"';
}

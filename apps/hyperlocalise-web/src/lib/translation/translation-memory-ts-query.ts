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

const MAX_SEARCH_TERMS = 50;

function tokenizeTranslationMemoryQuery(input: string): string[] {
  return input
    .replace(/[&|!():*<>'"-]/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, MAX_SEARCH_TERMS);
}

/**
 * Build a `simple` prefix `tsquery` for TM management and concordance search.
 * Operators are stripped so user input cannot change query structure.
 */
export function buildTranslationMemoryTsQuery(input: string): string {
  return tokenizeTranslationMemoryQuery(input)
    .map((word) => `${word}:*`)
    .join(" & ");
}

/**
 * Retrieve fuzzy TM candidates when words are added, removed, or changed.
 * AND-of-all-terms would exclude `Hello world` for `Hello brave world`.
 */
export function buildReportingMemoryMatchTsQuery(input: string): string {
  const terms = tokenizeTranslationMemoryQuery(input);
  if (terms.length === 0) return "";
  const prefixes = new Set(terms.map((word) => `${word}:*`));
  if (terms.length === 1) {
    const chars = Array.from(terms[0]);
    if (chars.length > 1) prefixes.add(`${chars.slice(0, -1).join("")}:*`);
    if (chars.length > 2) prefixes.add(`${chars.slice(0, -2).join("")}:*`);
  }
  return [...prefixes].join(" | ");
}

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
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";

import { err, ok } from "@/lib/primitives/result/results";

const { ensureAiFeaturesAllowedMock } = vi.hoisted(() => ({
  ensureAiFeaturesAllowedMock: vi.fn(),
}));

vi.mock("@/lib/billing/ai-features", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/billing/ai-features")>();
  return {
    ...actual,
    ensureAiFeaturesAllowed: ensureAiFeaturesAllowedMock,
  };
});

import { AI_FEATURES_REQUIRED_CODE, AI_FEATURES_REQUIRED_MESSAGE } from "@/lib/billing/ai-features";
import { ensureAiFeaturesAllowedStep } from "@/workflows/steps/translation-job";

function expectPlainObject(value: unknown) {
  expect(value).toEqual(expect.any(Object));
  expect(Object.getPrototypeOf(value)).toBe(Object.prototype);
}

describe("ensureAiFeaturesAllowedStep", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns a serializable POJO when AI features are allowed", async () => {
    ensureAiFeaturesAllowedMock.mockResolvedValue(ok(undefined));

    const result = await ensureAiFeaturesAllowedStep({ organizationId: "org_1" });

    expect(result).toEqual({ ok: true });
    expectPlainObject(result);
    expect("value" in result).toBe(false);
    expect(ensureAiFeaturesAllowedMock).toHaveBeenCalledWith({ organizationId: "org_1" });
  });

  it("returns a serializable POJO when AI features are denied", async () => {
    ensureAiFeaturesAllowedMock.mockResolvedValue(
      err({
        code: AI_FEATURES_REQUIRED_CODE,
        message: AI_FEATURES_REQUIRED_MESSAGE,
      }),
    );

    const result = await ensureAiFeaturesAllowedStep({ organizationId: "org_1" });

    expect(result).toEqual({
      ok: false,
      error: {
        code: AI_FEATURES_REQUIRED_CODE,
        message: AI_FEATURES_REQUIRED_MESSAGE,
      },
    });
    expectPlainObject(result);
    if (!result.ok) {
      expectPlainObject(result.error);
    }
  });
});

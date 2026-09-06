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
import "dotenv/config";

import { afterEach, beforeAll, describe, expect, it, vi } from "vite-plus/test";

const { resolveApiAuthContextFromSessionMock } = vi.hoisted(() => ({
  resolveApiAuthContextFromSessionMock: vi.fn(
    (options) =>
      globalThis.__resolveTestApiAuthContextFromSession?.(options) ??
      globalThis.__testApiAuthContext ??
      null,
  ),
}));

vi.mock("@/api/auth/workos-session", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/auth/workos-session")>();
  return {
    ...actual,
    resolveApiAuthContextFromSession: resolveApiAuthContextFromSessionMock,
  };
});

import { createApp } from "@/api/app";
import { createAuthTestFixture } from "@/api/test-auth.fixture";
import { db } from "@/lib/database/client";

const app = createApp();
const fixture = createAuthTestFixture();

beforeAll(async () => {
  await db.$client.query("select 1");
});

afterEach(async () => {
  resolveApiAuthContextFromSessionMock.mockClear();
  await fixture.cleanup();
});

function reportsUrl(organizationSlug: string, path = "") {
  return `http://localhost/api/orgs/${organizationSlug}/reports${path}`;
}

const rateBody = {
  name: "Default",
  sourceLocale: "en",
  targetLocale: "fr",
  step: "translation",
  basis: "word",
  rate: "0.12",
};

describe("reports routes", () => {
  it("lets localization managers read costs but not mutate rates", async () => {
    const identity = fixture.createWorkosIdentityWithRole("localization_manager");
    const headers = await fixture.authHeadersFor(identity);
    const slug = identity.organization.slug ?? "missing-slug";

    const costs = await app.request(reportsUrl(slug, "?view=costs"), { headers });
    expect(costs.status).toBe(200);

    const settings = await app.request(reportsUrl(slug, "/settings"), { headers });
    expect(settings.status).toBe(200);
    await expect(settings.json()).resolves.toMatchObject({
      settings: { financial: true, canManage: false },
    });

    const created = await app.request(reportsUrl(slug, "/rates"), {
      method: "POST",
      headers: { ...headers, "Content-Type": "application/json" },
      body: JSON.stringify(rateBody),
    });
    expect(created.status).toBe(403);
    await expect(created.json()).resolves.toMatchObject({ error: "report_costs_forbidden" });
  });

  it("lets admins create rates", async () => {
    const identity = fixture.createWorkosIdentityWithRole("admin");
    const headers = await fixture.authHeadersFor(identity);
    const slug = identity.organization.slug ?? "missing-slug";

    const settings = await app.request(reportsUrl(slug, "/settings"), { headers });
    await expect(settings.json()).resolves.toMatchObject({
      settings: { financial: true, canManage: true },
    });

    const created = await app.request(reportsUrl(slug, "/rates"), {
      method: "POST",
      headers: { ...headers, "Content-Type": "application/json" },
      body: JSON.stringify(rateBody),
    });
    expect(created.status).toBe(201);
    await expect(created.json()).resolves.toMatchObject({
      rate: { name: "Default", step: "translation" },
    });
  });
});

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

import { randomUUID } from "node:crypto";

import { eq } from "drizzle-orm";
import { afterEach, beforeAll, describe, expect, it } from "vite-plus/test";

import { createProjectTestFixture } from "@/api/routes/project/project.fixture";
import { db, schema } from "@/lib/database/client";
import { captureJobStatus } from "@/lib/reporting/capture";
import { completeReviewJob } from "@/lib/translation/review-job-queued-function";

const projectFixture = createProjectTestFixture();

beforeAll(async () => {
  await db.$client.query("select 1");
});

afterEach(async () => {
  await projectFixture.cleanup();
});

describe("review job reporting", () => {
  it("recognizes fixed review overrides after the job is marked succeeded", async () => {
    const { organization, project, user } = await projectFixture.createStoredProjectFixture();
    const workflowRunId = `run_${randomUUID()}`;
    const [job] = await db
      .insert(schema.jobs)
      .values({
        id: `job_${randomUUID()}`,
        organizationId: organization.id,
        projectId: project.id,
        createdByUserId: user.id,
        kind: "review",
        status: "running",
        workflowRunId,
        inputPayload: {},
      })
      .returning();
    await db.insert(schema.reviewJobDetails).values({
      jobId: job.id,
      criteria: "terminology",
      targetLocale: "fr-FR",
    });
    const [taskRate] = await db
      .insert(schema.reportingTaskRates)
      .values({
        organizationId: organization.id,
        jobId: job.id,
        step: "review",
        overrideUsd: "25.00000000",
      })
      .returning();

    await captureJobStatus({
      jobId: job.id,
      status: "succeeded",
      operationKey: `review:${job.id}:succeeded`,
    });
    const beforeComplete = await db
      .select({ id: schema.reportingCosts.id })
      .from(schema.reportingCosts)
      .where(eq(schema.reportingCosts.operationKey, `task-override:${taskRate.id}`));
    expect(beforeComplete).toHaveLength(0);

    await completeReviewJob({
      jobId: job.id,
      projectId: project.id,
      workflowRunId,
      outcome: { reviewedCount: 1 },
      status: "succeeded",
    });

    const [cost] = await db
      .select({
        amountUsd: schema.reportingCosts.amountUsd,
        basis: schema.reportingCosts.basis,
        step: schema.reportingCosts.step,
      })
      .from(schema.reportingCosts)
      .where(eq(schema.reportingCosts.operationKey, `task-override:${taskRate.id}`));
    expect(cost).toMatchObject({
      amountUsd: "25.00000000",
      basis: "manual",
      step: "review",
    });
  });

  it("lets project deletion null reporting FKs instead of failing", async () => {
    const { organization, project } = await projectFixture.createStoredProjectFixture();
    await db.insert(schema.reportingBudgets).values({
      organizationId: organization.id,
      projectId: project.id,
      budget: "100.00000000",
    });
    await db.delete(schema.projects).where(eq(schema.projects.id, project.id));
    const [budget] = await db
      .select({ projectId: schema.reportingBudgets.projectId })
      .from(schema.reportingBudgets)
      .where(eq(schema.reportingBudgets.organizationId, organization.id));
    expect(budget?.projectId).toBeNull();
  });
});

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
import { OrgPageSuspense } from "../_components/org-page-suspense";
import { ReportsWorkspace } from "@/components/reports/reports-workspace";
export default function ReportsPage({ params }: { params: Promise<{ organizationSlug: string }> }) {
  return (
    <OrgPageSuspense>
      <ReportsLoader params={params} />
    </OrgPageSuspense>
  );
}
async function ReportsLoader({ params }: { params: Promise<{ organizationSlug: string }> }) {
  const { organizationSlug } = await params;
  return <ReportsWorkspace organizationSlug={organizationSlug} />;
}

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
"use client";
import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useIntl } from "react-intl";
import { Button } from "@/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "./report-table";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { reportMessages as messages } from "./reports.messages";
import { reportingFetch } from "./reports-workspace";
export function TaskReportingPanel({ base, jobId }: { base: string; jobId: string }) {
  const intl = useIntl(),
    client = useQueryClient();
  const [removing, setRemoving] = useState<string | null>(null),
    [error, setError] = useState(false);
  const query = useQuery({
    queryKey: ["reports", base, jobId],
    queryFn: () =>
      reportingFetch<{
        taskReport: {
          analyses: {
            id: string;
            segmentId: string;
            targetLocale: string;
            words: number | null;
            bucket: string;
          }[];
          timeEntries: {
            id: string;
            workDate: string;
            step: string;
            minutes: number;
            note: string | null;
          }[];
        };
      }>(`${base}/tasks/${encodeURIComponent(jobId)}`),
  });
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-balance text-lg font-medium">{intl.formatMessage(messages.analysis)}</h2>
      <Table>
        <TableHeader>
          <TableRow>
            {(["segmentId", "targetLocale", "words", "bucket"] as const).map((key) => (
              <TableHead key={key}>{intl.formatMessage(messages[key])}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {query.data?.taskReport.analyses.map((row) => (
            <TableRow key={row.id}>
              <TableCell>{row.segmentId}</TableCell>
              <TableCell>{row.targetLocale}</TableCell>
              <TableCell>{row.words ?? "—"}</TableCell>
              <TableCell>{row.bucket}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <h2 className="text-balance text-lg font-medium">
        {intl.formatMessage(messages.timeEntries)}
      </h2>
      {query.data?.taskReport.timeEntries.map((entry) => (
        <div className="flex flex-wrap items-center gap-4 text-sm tabular-nums" key={entry.id}>
          <span>{entry.workDate.slice(0, 10)}</span>
          <span>{entry.step}</span>
          <span>
            {entry.minutes} {intl.formatMessage(messages.minutes)}
          </span>
          <span>{entry.note}</span>
          <Button variant="outline" size="sm" onClick={() => setRemoving(entry.id)}>
            {intl.formatMessage(messages.void)}
          </Button>
        </div>
      ))}
      {error || query.isError ? <p role="alert">{intl.formatMessage(messages.error)}</p> : null}
      <AlertDialog
        open={removing !== null}
        onOpenChange={(open) => {
          if (!open) setRemoving(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{intl.formatMessage(messages.confirmVoid)}</AlertDialogTitle>
            <AlertDialogDescription>
              {intl.formatMessage(messages.confirmVoidDescription)}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{intl.formatMessage(messages.cancel)}</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                const response = await fetch(`${base}/time-entries/${removing}`, {
                  method: "DELETE",
                });
                if (!response.ok) setError(true);
                else await client.invalidateQueries({ queryKey: ["reports"] });
                setRemoving(null);
              }}
            >
              {intl.formatMessage(messages.void)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}

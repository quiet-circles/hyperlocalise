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
import type { ComponentProps } from "react";
import { cn } from "@/lib/primitives/cn";
export function Table(props: ComponentProps<"table">) {
  return (
    <div className="w-full overflow-x-auto rounded-md border">
      <table {...props} className={cn("w-full text-sm", props.className)} />
    </div>
  );
}
export function TableHeader(props: ComponentProps<"thead">) {
  return <thead {...props} className={cn("border-b bg-muted", props.className)} />;
}
export function TableBody(props: ComponentProps<"tbody">) {
  return <tbody {...props} />;
}
export function TableRow(props: ComponentProps<"tr">) {
  return <tr {...props} className={cn("border-b last:border-0", props.className)} />;
}
export function TableHead(props: ComponentProps<"th">) {
  return (
    <th
      scope="col"
      {...props}
      className={cn("whitespace-nowrap px-3 py-3 text-left font-medium", props.className)}
    />
  );
}
export function TableCell(props: ComponentProps<"td">) {
  return <td {...props} className={cn("px-3 py-3", props.className)} />;
}

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
import { Suspense } from "react";
import { notFound } from "next/navigation";

import { isSupportedAppLocale } from "@/lib/app-i18n/locales";

type LocaleLayoutProps = {
  children: React.ReactNode;
  params: Promise<{ lang: string }>;
};

export default function LocaleLayout({ children, params }: LocaleLayoutProps) {
  return (
    <>
      <Suspense>
        <LocaleParamGate params={params} />
      </Suspense>
      {children}
    </>
  );
}

async function LocaleParamGate({ params }: { params: Promise<{ lang: string }> }) {
  const { lang } = await params;

  if (!isSupportedAppLocale(lang)) {
    notFound();
  }

  return null;
}

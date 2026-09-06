"use client";

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
import type { CSSProperties, ReactNode } from "react";
import { useAuth } from "@workos-inc/authkit-nextjs/components";
import Image from "next/image";
import Link from "next/link";
import { FormattedMessage, useIntl, type MessageDescriptor } from "react-intl";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAppLocale } from "@/lib/app-i18n/use-app-locale";
import { rewriteAppLocalePath } from "@/lib/app-i18n/rewrite-app-locale-path";
import { cn } from "@/lib/primitives/cn";
import { REQUEST_DEMO_URL } from "../request-demo";
import { heroSectionMessages } from "../hero-section.messages";
import { ContentOpsMockStage } from "../content-ops/content-ops-mock-stage";
import { homepageMessages as m } from "./homepage.messages";
import { AgentChannelPreview } from "./agent-channel-preview";
import { McpClientLogos } from "./mcp-client-logos";
import { ConnectedCampaign } from "./connected-campaign";
import { hasProductPreviewVideoUrl, PRODUCT_PREVIEW_VIDEO_URL, PRODUCTS } from "./product-preview";
import { SEAFOAM_MESH_GRADIENT_SRC, SectionMeshBackground } from "../hero-frame-mesh-stage";

const SECTION_CLASS = "mx-auto max-w-7xl px-5 py-20 sm:px-8 sm:py-28 lg:px-10";
const HOME_STYLE = {
  "--home-ink": "#123c3b",
  "--home-paper": "#e5eee5",
} as CSSProperties;

const TRUSTED_BY_LOGOS = [
  {
    id: "heidi-health",
    href: "https://www.heidihealth.com",
    src: "/images/customers/heidi-health-logo.png",
    alt: heroSectionMessages.heidiHealthAlt,
    width: 800,
    height: 332,
    className: "h-7 sm:h-8",
  },
  {
    id: "tourfinder",
    href: "https://tourfinder.vn",
    src: "/images/customers/tourfinder-logo.png",
    alt: heroSectionMessages.tourfinderAlt,
    width: 1177,
    height: 294,
    className: "h-6 sm:h-7",
  },
  {
    id: "tourmatic",
    href: "https://tourmatic.io",
    src: "/images/customers/tourmatic-logo.svg",
    alt: heroSectionMessages.tourmaticAlt,
    width: 315,
    height: 58,
    className: "h-6 sm:h-7",
  },
  {
    id: "weex",
    href: "https://www.weex.com",
    src: "/images/customers/weex-logo.svg",
    alt: heroSectionMessages.weexAlt,
    width: 134,
    height: 28,
    className: "h-6 sm:h-7",
  },
] as const;

function SectionHeading({
  eyebrow,
  title,
  body,
}: {
  eyebrow?: MessageDescriptor;
  title: MessageDescriptor;
  body?: MessageDescriptor;
}) {
  return (
    <div className="mx-auto mb-10 flex max-w-3xl flex-col items-center gap-4 text-center sm:mb-14">
      {eyebrow ? (
        <p className="text-xs font-medium text-muted-foreground">
          <FormattedMessage {...eyebrow} />
        </p>
      ) : null}
      <h2 className="font-heading text-[clamp(2.25rem,4vw,3.5rem)] leading-[1.1] text-balance">
        <FormattedMessage {...title} />
      </h2>
      {body ? (
        <p className="max-w-2xl text-base leading-7 text-pretty text-muted-foreground">
          <FormattedMessage {...body} />
        </p>
      ) : null}
    </div>
  );
}

function DemoCta() {
  const { user, loading } = useAuth();
  const locale = useAppLocale();
  if (loading) return <Skeleton className="h-11 w-40" />;
  return (
    <Button
      size="lg"
      nativeButton={false}
      render={
        user ? (
          <Link href={rewriteAppLocalePath("/dashboard", locale)} />
        ) : (
          <a href={REQUEST_DEMO_URL} target="_blank" rel="noopener noreferrer" />
        )
      }
    >
      <FormattedMessage
        {...(user ? heroSectionMessages.goToDashboard : heroSectionMessages.joinWaitlist)}
      />
    </Button>
  );
}

function CustomerLogos({ onDark }: { onDark?: boolean }) {
  const intl = useIntl();
  return (
    <div className="mx-auto flex max-w-5xl flex-col items-center gap-5 px-5">
      <p
        className={cn(
          "text-[0.7rem] font-medium tracking-[0.18em] uppercase",
          onDark ? "text-white/55" : "text-muted-foreground",
        )}
      >
        <FormattedMessage {...heroSectionMessages.trustedBy} />
      </p>
      <ul className="flex flex-wrap items-center justify-center gap-2 sm:gap-3">
        {TRUSTED_BY_LOGOS.map((logo) => (
          <li key={logo.id}>
            <a
              href={logo.href}
              target="_blank"
              rel="noopener noreferrer"
              className={cn(
                "group inline-flex items-center rounded-lg px-4 py-3 focus-visible:outline-2 focus-visible:outline-offset-4",
                onDark && "hover:bg-white focus-visible:bg-white",
              )}
            >
              <Image
                src={logo.src}
                alt={intl.formatMessage(logo.alt)}
                width={logo.width}
                height={logo.height}
                unoptimized={logo.src.endsWith(".svg")}
                className={cn(
                  "w-auto",
                  onDark
                    ? "opacity-70 brightness-0 invert transition-[filter,opacity] duration-200 ease-out motion-reduce:transition-none group-hover:opacity-100 group-hover:brightness-100 group-hover:invert-0 group-focus-visible:opacity-100 group-focus-visible:brightness-100 group-focus-visible:invert-0"
                    : "grayscale dark:brightness-0 dark:invert",
                  logo.className,
                )}
              />
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

function ProductPreviewVideo({ src }: { src: string }) {
  const intl = useIntl();
  return (
    <div className="mx-auto mt-12 max-w-5xl rounded-2xl bg-white/10 p-2 sm:mt-16 sm:p-3">
      <video
        className="aspect-video w-full rounded-xl border border-white/15 bg-black/25"
        src={src}
        controls
        playsInline
        preload="metadata"
        aria-label={intl.formatMessage(m.preview)}
      />
    </div>
  );
}

function AgentsSection() {
  const locale = useAppLocale();
  return (
    <section id="agents" className="scroll-mt-20 bg-muted">
      <div className={SECTION_CLASS}>
        <div className="grid items-start gap-10 lg:grid-cols-[0.8fr_1fr] lg:gap-16">
          <div className="flex flex-col items-start gap-5">
            <p className="text-xs font-medium text-muted-foreground">
              <FormattedMessage {...m.agentsEyebrow} />
            </p>
            <h2 className="font-heading text-[clamp(2.25rem,4vw,3.5rem)] leading-[1.1] text-balance">
              <FormattedMessage {...m.agentsTitle} />
            </h2>
            <p className="text-base leading-7 text-pretty text-muted-foreground">
              <FormattedMessage {...m.agentsBody} />
            </p>
            <Link
              className="mt-2 rounded-sm text-sm font-medium underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4"
              href={rewriteAppLocalePath("/integrations", locale)}
            >
              <FormattedMessage {...m.integrationsLink} /> <span aria-hidden>↗</span>
            </Link>
          </div>
          <div>
            <AgentChannelPreview />
            <p className="mt-4 text-center text-xs text-muted-foreground">
              <FormattedMessage {...m.agentExample} />
            </p>
          </div>
        </div>
        <div className="mt-12 flex flex-col gap-5 rounded-xl border border-border bg-background p-6 sm:flex-row sm:items-center sm:gap-8 sm:p-8">
          <McpClientLogos />
          <div className="min-w-0 flex-1">
            <h3 className="mb-2 text-lg font-medium">
              <FormattedMessage {...m.mcpTitle} />
            </h3>
            <p className="text-sm leading-6 text-pretty text-muted-foreground">
              <FormattedMessage {...m.mcpBody} />
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

export function PlatformHomepage({ children, plans }: { children: ReactNode; plans: ReactNode }) {
  const locale = useAppLocale();
  const hasPreviewVideo = hasProductPreviewVideoUrl(PRODUCT_PREVIEW_VIDEO_URL);
  return (
    <div style={HOME_STYLE} className="bg-background text-foreground">
      <section
        id="home"
        className={cn(
          "relative isolate -mt-16 overflow-hidden px-5 pb-10 pt-28 text-white sm:px-8 sm:pb-14 sm:pt-32 lg:px-10",
          !hasPreviewVideo && "flex min-h-svh flex-col",
        )}
      >
        <SectionMeshBackground src={SEAFOAM_MESH_GRADIENT_SRC} priority />
        <div aria-hidden className="absolute inset-0 -z-10 bg-black/30" />
        <div
          className={cn(
            "mx-auto w-full max-w-6xl",
            !hasPreviewVideo && "flex min-h-0 flex-1 flex-col",
          )}
        >
          <div
            className={cn(
              "mx-auto flex max-w-4xl flex-col items-center gap-6 text-center",
              !hasPreviewVideo && "min-h-0 flex-1 justify-center",
            )}
          >
            <h1 className="font-heading text-[clamp(2.5rem,5vw,4.25rem)] leading-[1.05] text-balance">
              <FormattedMessage {...m.heroHeadline} />
            </h1>
            <p className="max-w-2xl text-base leading-7 text-pretty text-white/80 sm:text-lg">
              <FormattedMessage {...m.heroBody} />
            </p>
            <div className="mt-2 flex flex-wrap items-center justify-center gap-4">
              <div className="text-foreground">
                <DemoCta />
              </div>
              <a
                href="#explore"
                className="rounded-md px-4 py-3 text-sm font-medium underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4"
              >
                <FormattedMessage {...m.explore} /> <span aria-hidden>↓</span>
              </a>
            </div>
          </div>
          <div className={hasPreviewVideo ? "mt-12 sm:mt-16" : "shrink-0 pb-2 pt-10 sm:pt-12"}>
            <CustomerLogos onDark />
          </div>
          {hasPreviewVideo ? <ProductPreviewVideo src={PRODUCT_PREVIEW_VIDEO_URL.trim()} /> : null}
        </div>
      </section>
      <section id="products" className={SECTION_CLASS}>
        <SectionHeading eyebrow={m.pillarsEyebrow} title={m.pillarsTitle} body={m.pillarsBody} />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {PRODUCTS.map((product, index) => (
            <a
              key={product.id}
              href="#explore"
              className="group flex flex-col overflow-hidden rounded-xl border border-border focus-visible:outline-2 focus-visible:outline-offset-4"
            >
              <div className="relative isolate flex h-44 items-center justify-between overflow-hidden px-6 text-white">
                <Image
                  src={
                    [
                      "/images/mesh/mesh-gradient-1784863888954.jpg",
                      "/images/mesh/mesh-gradient-1784864073608.jpg",
                      "/images/vimal-s-GBg3jyGS-Ug-unsplash.jpg",
                      "/images/mesh/mesh-gradient-1784863799475.jpg",
                      SEAFOAM_MESH_GRADIENT_SRC,
                    ][index]!
                  }
                  alt=""
                  fill
                  sizes="(min-width: 1024px) 240px, 50vw"
                  className="-z-20 object-cover"
                />
                <div aria-hidden className="absolute inset-0 -z-10 bg-black/25" />
                <span
                  aria-hidden
                  className="font-heading text-5xl transition-transform duration-150 group-hover:-translate-y-1 motion-reduce:transition-none"
                >
                  {product.mark}
                </span>
              </div>
              <div className="flex flex-1 flex-col gap-3 p-5">
                <h3 className="text-sm font-semibold">
                  <FormattedMessage {...product.title} />
                </h3>
                <p className="text-sm leading-6 text-muted-foreground">
                  <FormattedMessage {...product.short} />
                </p>
                <span aria-hidden className="mt-auto pt-3 text-lg">
                  ↗
                </span>
              </div>
            </a>
          ))}
        </div>
      </section>
      <AgentsSection />
      <section id="explore" className="scroll-mt-16">
        <div className={SECTION_CLASS}>
          <SectionHeading
            eyebrow={m.explorerEyebrow}
            title={m.explorerTitle}
            body={m.explorerBody}
          />
          <ContentOpsMockStage />
        </div>
      </section>
      <ConnectedCampaign />
      {children}
      <section id="plans" className="bg-muted/70">
        <div className="mx-auto max-w-7xl px-5 py-20 sm:px-8 sm:py-28 lg:px-10">
          <SectionHeading title={m.pricingTitle} body={m.pricingBody} />
          <div className="overflow-hidden rounded-xl border border-border bg-background">
            {plans}
          </div>
          <div className="mt-10 flex justify-center">
            <Button
              size="lg"
              variant="outline"
              nativeButton={false}
              render={<Link href={rewriteAppLocalePath("/pricing", locale)} />}
            >
              <FormattedMessage {...m.pricingLink} />
              <span aria-hidden>↗</span>
            </Button>
          </div>
        </div>
      </section>
    </div>
  );
}

export function HomepageFinalCta() {
  return (
    <section
      aria-labelledby="homepage-final-cta-heading"
      className="border-t border-border bg-muted"
    >
      <div className="mx-auto max-w-7xl px-5 py-28 sm:px-8 sm:py-32 lg:px-10">
        <div className="flex flex-col items-center gap-6 text-center">
          <h2
            id="homepage-final-cta-heading"
            className="max-w-3xl font-heading text-[clamp(2.25rem,4vw,3.5rem)] leading-[1.1] text-balance text-foreground"
          >
            <FormattedMessage {...m.finalTitle} />
          </h2>
          <p className="max-w-xl text-base leading-7 text-pretty text-muted-foreground">
            <FormattedMessage {...m.finalBody} />
          </p>
          <DemoCta />
        </div>
      </div>
    </section>
  );
}

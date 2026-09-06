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
import { Button } from "@/components/ui/button";
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
  navigationMenuTriggerStyle,
} from "@/components/ui/navigation-menu";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  cliDocsUrl,
  contactUrl,
  docsUrl,
  githubActionUrl,
  githubRepoUrl,
  trustCenterUrl,
} from "@/components/marketing/marketing-page-content";
import { productFooterLinks } from "@/components/marketing/product/product-page-content";
import { productPageMessages } from "@/components/marketing/product/product-page-content.messages";
import type { ProductMessageKey } from "@/components/marketing/product/product-page-content.messages";
import { useCaseFooterLinks } from "@/components/marketing/use-case/use-case-page-content";
import { useCasePageMessages } from "@/components/marketing/use-case/use-case-page-content.messages";
import type { UseCaseMessageKey } from "@/components/marketing/use-case/use-case-page-content.messages";
import { cn } from "@/lib/primitives/cn";
import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";

import LocaleToggle from "@/components/locale-toggle/locale-toggle";
import ThemeToggle from "@/components/theme-toggle/theme-toggle";
import { themeToggleMessages } from "@/components/theme-toggle/theme-toggle.messages";
import type { AppLocale } from "@/lib/app-i18n/locales";
import { rewriteAppLocalePath } from "@/lib/app-i18n/rewrite-app-locale-path";
import { useAppLocale } from "@/lib/app-i18n/use-app-locale";

import {
  NavbarDesktopAuthActions,
  NavbarMobileAuthCta,
  NavbarMobileAuthFooter,
  type NavbarAuthState,
} from "./navbar-auth-actions";
import { navbarMessages } from "./navbar.messages";

type NavbarMessageKey = keyof typeof navbarMessages;

type NavLink =
  | {
      href: string;
      kind: "navbar";
      labelKey: NavbarMessageKey;
      external?: boolean;
    }
  | {
      href: string;
      kind: "product";
      labelKey: ProductMessageKey;
      external?: boolean;
    }
  | {
      href: string;
      kind: "useCase";
      labelKey: UseCaseMessageKey;
      external?: boolean;
    };

const productLinks: NavLink[] = productFooterLinks.map((link) => ({
  href: link.href,
  kind: "product" as const,
  labelKey: link.productLabelKey,
}));

const useCaseLinks: NavLink[] = useCaseFooterLinks.map((link) => ({
  href: link.href,
  kind: "useCase" as const,
  labelKey: link.useCaseLabelKey,
}));

const pricingLink: NavLink = {
  href: "/pricing",
  kind: "navbar",
  labelKey: "navPricing",
};

const companyPageLink: NavLink = {
  href: "/company",
  kind: "navbar",
  labelKey: "navCompany",
};

const resourceLinks: NavLink[] = [
  { href: docsUrl, kind: "navbar", labelKey: "navDocumentation", external: true },
  { href: cliDocsUrl, kind: "navbar", labelKey: "navCliDocs", external: true },
  { href: "/localisation-audit", kind: "navbar", labelKey: "navLocalisationAudit" },
  { href: "/integrations", kind: "navbar", labelKey: "navIntegrations" },
  { href: "/blog", kind: "navbar", labelKey: "navBlog" },
  { href: "/startups", kind: "navbar", labelKey: "navStartups" },
  { href: githubActionUrl, kind: "navbar", labelKey: "navGitHubAction", external: true },
  { href: githubRepoUrl, kind: "navbar", labelKey: "navGitHub", external: true },
];

const legalLinks: NavLink[] = [
  { href: contactUrl, kind: "navbar", labelKey: "navContact" },
  { href: trustCenterUrl, kind: "navbar", labelKey: "navTrustCenter", external: true },
  { href: "/privacy", kind: "navbar", labelKey: "navPrivacy" },
  { href: "/terms", kind: "navbar", labelKey: "navTerms" },
];

const mobileNavLinkClassName =
  "flex min-h-11 items-center rounded-3xl px-4 py-3 text-base font-medium text-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 data-[active=true]:bg-muted data-[active=true]:text-foreground";

const megaMenuLinkClassName =
  "group/mega-menu-link flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus:bg-muted focus:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50";

const megaMenuHeadingClassName =
  "px-3 pb-2 text-[11px] font-medium tracking-[0.16em] text-muted-foreground uppercase";

function getNavLinkHref(link: NavLink, locale: AppLocale) {
  return link.href.startsWith("/") ? rewriteAppLocalePath(link.href, locale) : link.href;
}

function NavLinkLabel({ link }: { link: NavLink }) {
  if (link.kind === "product") {
    return <FormattedMessage {...productPageMessages[link.labelKey]} />;
  }

  if (link.kind === "useCase") {
    return <FormattedMessage {...useCasePageMessages[link.labelKey]} />;
  }

  return <FormattedMessage {...navbarMessages[link.labelKey]} />;
}

function ExternalLinkIcon() {
  return (
    <svg
      aria-hidden="true"
      className="size-3.5 shrink-0 text-muted-foreground transition-colors group-hover/mega-menu-link:text-foreground"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M4.5 11.5 11.5 4.5M6.5 4.5h5v5"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function MegaMenuLink({ link, locale }: { link: NavLink; locale: AppLocale }) {
  const intl = useIntl();

  if (link.external) {
    return (
      <NavigationMenuLink
        href={link.href}
        target="_blank"
        rel="noopener noreferrer"
        className={megaMenuLinkClassName}
      >
        <span>
          <NavLinkLabel link={link} />
        </span>
        <span className="sr-only">{intl.formatMessage(navbarMessages.externalLinkAriaLabel)}</span>
        <ExternalLinkIcon />
      </NavigationMenuLink>
    );
  }

  return (
    <NavigationMenuLink href={getNavLinkHref(link, locale)} className={megaMenuLinkClassName}>
      <NavLinkLabel link={link} />
    </NavigationMenuLink>
  );
}

function MegaMenuColumn({
  headingKey,
  links,
  locale,
}: {
  headingKey: NavbarMessageKey;
  links: NavLink[];
  locale: AppLocale;
}) {
  return (
    <div className="min-w-[11.5rem]">
      <div className={megaMenuHeadingClassName}>
        <FormattedMessage {...navbarMessages[headingKey]} />
      </div>
      <ul className="grid gap-0.5">
        {links.map((link) => (
          <li key={`${link.kind}-${link.href}`}>
            <MegaMenuLink link={link} locale={locale} />
          </li>
        ))}
      </ul>
    </div>
  );
}

function Logo({ locale, onHero = false }: { locale: AppLocale; onHero?: boolean }) {
  const intl = useIntl();

  return (
    <Link href={rewriteAppLocalePath("/", locale)} className="flex items-center gap-2.5">
      <Image
        src="/images/logo.png"
        className="size-8"
        width={32}
        height={32}
        alt={intl.formatMessage(navbarMessages.logoAlt)}
      />
      <span
        className={cn(
          "hidden font-sans text-base font-semibold tracking-tight md:inline",
          onHero && "text-white",
        )}
      >
        <FormattedMessage {...navbarMessages.brandName} />
      </span>
    </Link>
  );
}

const HERO_NAV_SCROLL_THRESHOLD_PX = 16;

function useHomeHeroNavTone() {
  const pathname = usePathname();
  const pathSegments = pathname.split("/").filter(Boolean);
  const isMarketingHome = pathSegments.length === 1;
  const isStartupsPage = pathSegments.length === 2 && pathSegments[1] === "startups";
  const usesFullBleedHero = isMarketingHome || isStartupsPage;
  const [onHero, setOnHero] = useState(usesFullBleedHero);

  useEffect(() => {
    if (!usesFullBleedHero) {
      setOnHero(false);
      return;
    }

    const update = () => {
      setOnHero(window.scrollY < HERO_NAV_SCROLL_THRESHOLD_PX);
    };

    update();
    window.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update, { passive: true });
    return () => {
      window.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [pathname, usesFullBleedHero]);

  return usesFullBleedHero && onHero;
}

function MobileNavSection({
  headingKey,
  links,
  locale,
}: {
  headingKey: NavbarMessageKey;
  links: NavLink[];
  locale: AppLocale;
}) {
  const intl = useIntl();

  return (
    <div className="space-y-1.5">
      <div className="px-4 pt-3 pb-1 text-[11px] font-medium tracking-[0.16em] text-muted-foreground uppercase">
        <FormattedMessage {...navbarMessages[headingKey]} />
      </div>
      {links.map((link) => {
        const content = (
          <>
            <NavLinkLabel link={link} />
            {link.external ? (
              <>
                <span className="sr-only">
                  {intl.formatMessage(navbarMessages.externalLinkAriaLabel)}
                </span>
                <ExternalLinkIcon />
              </>
            ) : null}
          </>
        );

        if (link.external) {
          return (
            <SheetClose
              key={`${link.kind}-${link.href}`}
              render={<a href={link.href} target="_blank" rel="noopener noreferrer" />}
              className={cn(mobileNavLinkClassName, "justify-between gap-3")}
            >
              {content}
            </SheetClose>
          );
        }

        return (
          <SheetClose
            key={`${link.kind}-${link.href}`}
            render={<a href={getNavLinkHref(link, locale)} />}
            className={mobileNavLinkClassName}
          >
            {content}
          </SheetClose>
        );
      })}
    </div>
  );
}

function MobileNavigation({ auth, locale }: { auth: NavbarAuthState; locale: AppLocale }) {
  const intl = useIntl();

  return (
    <Sheet>
      <SheetTrigger
        render={
          <Button
            variant="outline"
            size="icon-sm"
            className="rounded-full border-border bg-background/80 backdrop-blur-sm"
          />
        }
      >
        <span className="sr-only">
          <FormattedMessage {...navbarMessages.openNavigationMenu} />
        </span>
        <svg
          aria-hidden="true"
          className="pointer-events-none"
          width={18}
          height={18}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path d="M4 7H20" />
          <path d="M4 12H20" />
          <path d="M4 17H20" />
        </svg>
      </SheetTrigger>
      <SheetContent
        side="right"
        className="w-[min(90vw,22rem)] border-s border-border bg-background/98 px-0"
      >
        <SheetHeader className="gap-4 border-b border-border px-5 pb-5 pt-6 text-left">
          <SheetTitle className="sr-only">
            <FormattedMessage {...navbarMessages.navigationMenuTitle} />
          </SheetTitle>
          <div className="text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
            <FormattedMessage {...navbarMessages.navigationHeading} />
          </div>
          <div className="pr-10">
            <Logo locale={locale} />
          </div>
        </SheetHeader>
        <div className="flex flex-1 flex-col overflow-y-auto px-3 py-2">
          <nav
            aria-label={intl.formatMessage(navbarMessages.mobileNavAriaLabel)}
            className="flex flex-col gap-2 pb-4"
          >
            <MobileNavSection
              headingKey="navPlatformHeading"
              links={productLinks}
              locale={locale}
            />
            <MobileNavSection
              headingKey="navUseCasesHeading"
              links={useCaseLinks}
              locale={locale}
            />
            <MobileNavSection
              headingKey="navResourcesHeading"
              links={resourceLinks}
              locale={locale}
            />
            <MobileNavSection headingKey="navLegalHeading" links={legalLinks} locale={locale} />
            <div className="space-y-1.5">
              <SheetClose
                render={<a href={rewriteAppLocalePath(pricingLink.href, locale)} />}
                className={mobileNavLinkClassName}
              >
                <NavLinkLabel link={pricingLink} />
              </SheetClose>
              <SheetClose
                render={<a href={rewriteAppLocalePath(companyPageLink.href, locale)} />}
                className={mobileNavLinkClassName}
              >
                <NavLinkLabel link={companyPageLink} />
              </SheetClose>
            </div>
          </nav>
        </div>
        <SheetFooter className="gap-3 border-t border-border px-5 py-5">
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm text-muted-foreground">
              <FormattedMessage {...themeToggleMessages.changeTheme} />
            </span>
            <ThemeToggle />
          </div>
          <NavbarMobileAuthFooter auth={auth} locale={locale} />
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function DesktopNavigation({ locale }: { locale: AppLocale }) {
  return (
    <NavigationMenu className="mx-auto hidden max-w-none md:flex">
      <NavigationMenuList className="gap-1">
        <NavigationMenuItem>
          <NavigationMenuTrigger className="px-3 py-2 font-medium text-muted-foreground hover:text-foreground data-popup-open:text-foreground data-open:text-foreground">
            <FormattedMessage {...navbarMessages.navProduct} />
          </NavigationMenuTrigger>
          <NavigationMenuContent>
            <div className="grid w-max grid-cols-2 gap-6 p-3 pe-4">
              <MegaMenuColumn
                headingKey="navPlatformHeading"
                links={productLinks}
                locale={locale}
              />
              <MegaMenuColumn
                headingKey="navUseCasesHeading"
                links={useCaseLinks}
                locale={locale}
              />
            </div>
          </NavigationMenuContent>
        </NavigationMenuItem>
        <NavigationMenuItem>
          <NavigationMenuTrigger className="px-3 py-2 font-medium text-muted-foreground hover:text-foreground data-popup-open:text-foreground data-open:text-foreground">
            <FormattedMessage {...navbarMessages.navResources} />
          </NavigationMenuTrigger>
          <NavigationMenuContent>
            <div className="grid w-max grid-cols-2 gap-6 p-3 pe-4">
              <MegaMenuColumn
                headingKey="navResourcesHeading"
                links={resourceLinks}
                locale={locale}
              />
              <MegaMenuColumn headingKey="navLegalHeading" links={legalLinks} locale={locale} />
            </div>
          </NavigationMenuContent>
        </NavigationMenuItem>
        <NavigationMenuItem>
          <NavigationMenuLink
            href={rewriteAppLocalePath(pricingLink.href, locale)}
            className={cn(
              navigationMenuTriggerStyle(),
              "px-3 py-2 font-medium text-muted-foreground hover:text-foreground",
            )}
          >
            <NavLinkLabel link={pricingLink} />
          </NavigationMenuLink>
        </NavigationMenuItem>
        <NavigationMenuItem>
          <NavigationMenuLink
            href={rewriteAppLocalePath(companyPageLink.href, locale)}
            className={cn(
              navigationMenuTriggerStyle(),
              "px-3 py-2 font-medium text-muted-foreground hover:text-foreground",
            )}
          >
            <NavLinkLabel link={companyPageLink} />
          </NavigationMenuLink>
        </NavigationMenuItem>
      </NavigationMenuList>
    </NavigationMenu>
  );
}

export function NavbarView({ auth }: { auth: NavbarAuthState }) {
  const onHero = useHomeHeroNavTone();
  const locale = useAppLocale();

  return (
    <header
      className={cn(
        "sticky top-0 z-40 border-b transition-[background-color,border-color,color,backdrop-filter] duration-300",
        onHero
          ? "border-transparent bg-transparent text-white"
          : "border-border bg-background/95 text-foreground backdrop-blur supports-[backdrop-filter]:bg-background/80",
      )}
    >
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-3 px-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-3 lg:gap-8">
          <Logo locale={locale} onHero={onHero} />
          <div
            className={cn(
              onHero &&
                [
                  "[&_[data-slot=navigation-menu-trigger]]:bg-transparent [&_[data-slot=navigation-menu-trigger]]:text-white [&_[data-slot=navigation-menu-trigger]]:hover:bg-white/10 [&_[data-slot=navigation-menu-trigger]]:hover:text-white",
                  "[&_[data-slot=navigation-menu-link]]:bg-transparent [&_[data-slot=navigation-menu-link]]:text-white [&_[data-slot=navigation-menu-link]]:hover:bg-white/10 [&_[data-slot=navigation-menu-link]]:hover:text-white",
                ].join(" "),
            )}
          >
            <DesktopNavigation locale={locale} />
          </div>
        </div>

        <div
          className={cn(
            "hidden items-center gap-2 md:flex",
            onHero &&
              "[&_button]:text-white [&_button[data-variant=ghost]]:hover:bg-white/10 [&_button[data-variant=ghost]]:hover:text-white [&_a]:text-white",
          )}
        >
          <NavbarDesktopAuthActions auth={auth} locale={locale} />
          <LocaleToggle />
          <ThemeToggle />
        </div>

        <div
          className={cn(
            "flex items-center gap-2 md:hidden",
            onHero && "[&_button]:border-white/30 [&_button]:text-white",
          )}
        >
          <NavbarMobileAuthCta auth={auth} locale={locale} />
          <MobileNavigation auth={auth} locale={locale} />
          <LocaleToggle />
        </div>
      </div>
    </header>
  );
}

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
import { useEffect, useState, type ReactNode } from "react";
import { FormattedMessage, useIntl, type MessageDescriptor } from "react-intl";

import {
  DUSK_MESH_GRADIENT_SRC,
  SectionMeshBackground,
} from "@/components/marketing/hero-frame-mesh-stage";
import {
  Carousel,
  type CarouselApi,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from "@/components/ui/carousel";
import { cn } from "@/lib/primitives/cn";

import { homepageMessages as m } from "./homepage.messages";

type JourneyCardData = {
  id: string;
  index: string;
  title: MessageDescriptor;
  body: MessageDescriptor;
  kicker: MessageDescriptor;
  display: ReactNode;
  surface: string;
};

const CARD_BASIS = "basis-[min(20.5rem,calc(100%-1.5rem))] sm:basis-[22.5rem] lg:basis-[24.5rem]";

function DisplayMark({ children }: { children: ReactNode }) {
  return (
    <p className="font-heading whitespace-nowrap text-[2.5rem] leading-none sm:text-[2.75rem]">
      {children}
    </p>
  );
}

const CARDS: JourneyCardData[] = [
  {
    id: "launch",
    index: "01",
    title: m.capabilityLaunchTitle,
    body: m.capabilityLaunchBody,
    kicker: m.capabilityLaunchKicker,
    display: <DisplayMark>1 → 12</DisplayMark>,
    surface: "bg-fog text-ink",
  },
  {
    id: "campaign",
    index: "02",
    title: m.capabilityCampaignTitle,
    body: m.capabilityCampaignBody,
    kicker: m.capabilityCampaignKicker,
    display: <DisplayMark>US · JP · DE</DisplayMark>,
    surface: "bg-steel text-ink",
  },
  {
    id: "presence",
    index: "03",
    title: m.capabilityPresenceTitle,
    body: m.capabilityPresenceBody,
    kicker: m.capabilityPresenceKicker,
    display: <DisplayMark>日本</DisplayMark>,
    surface: "bg-navy text-fog",
  },
  {
    id: "adapt",
    index: "04",
    title: m.capabilityAdaptTitle,
    body: m.capabilityAdaptBody,
    kicker: m.capabilityAdaptKicker,
    display: <DisplayMark>Aa / あ / À</DisplayMark>,
    surface: "bg-slate text-fog",
  },
  {
    id: "brand",
    index: "05",
    title: m.capabilityBrandTitle,
    body: m.capabilityBrandBody,
    kicker: m.capabilityBrandKicker,
    display: <DisplayMark>Brand · 法</DisplayMark>,
    surface: "bg-ink text-fog",
  },
];

function JourneyCard({ card }: { card: JourneyCardData }) {
  return (
    <article
      className={cn(
        "relative flex h-[30rem] select-none flex-col overflow-hidden rounded-[1.75rem] sm:h-[32rem] sm:rounded-[2rem]",
        card.surface,
      )}
    >
      <div className="flex min-h-0 flex-1 flex-col px-6 pt-6 sm:px-7 sm:pt-7">
        <p className="mb-3 text-xs tracking-[0.14em] text-current/55">{card.index}</p>
        <h3 className="font-heading text-2xl leading-[1.15] text-balance sm:text-[1.7rem]">
          <FormattedMessage {...card.title} />
        </h3>
        <p className="mt-3 text-pretty text-sm leading-6 text-current/75">
          <FormattedMessage {...card.body} />
        </p>
        <div className="mt-auto flex min-h-0 flex-1 flex-col justify-end pb-6 sm:pb-7">
          <div aria-hidden>{card.display}</div>
          <p className="mt-3 text-sm leading-6 text-current/70">
            <FormattedMessage {...card.kicker} />
          </p>
        </div>
      </div>
    </article>
  );
}

function JourneyCarousel() {
  const intl = useIntl();
  const [api, setApi] = useState<CarouselApi>();
  const [current, setCurrent] = useState(1);
  const [count, setCount] = useState(CARDS.length);

  useEffect(() => {
    if (!api) {
      return;
    }

    const sync = () => {
      const snaps = api.scrollSnapList().length;
      if (snaps === 0) {
        return;
      }

      setCurrent(api.selectedScrollSnap() + 1);
      setCount(snaps);
    };

    sync();
    api.on("select", sync);
    api.on("reInit", sync);

    return () => {
      api.off("select", sync);
      api.off("reInit", sync);
    };
  }, [api]);

  const controlClassName =
    "static top-auto start-auto size-10 translate-none border-white/25 bg-white/10 text-white hover:bg-white/20 hover:text-white disabled:opacity-35";

  return (
    <Carousel
      opts={{ align: "start", containScroll: "trimSnaps" }}
      setApi={setApi}
      scrollOnWheel
      className="w-full min-w-0"
      aria-label={intl.formatMessage(m.journeyTitle)}
    >
      <CarouselContent className="cursor-grab active:cursor-grabbing">
        {CARDS.map((card) => (
          <CarouselItem key={card.id} className={CARD_BASIS}>
            <JourneyCard card={card} />
          </CarouselItem>
        ))}
      </CarouselContent>
      <div className="mt-8 flex items-center justify-end gap-3">
        <p className="tabular-nums text-sm text-white/70">
          <FormattedMessage {...m.journeyCarouselPage} values={{ current, count }} />
        </p>
        <CarouselPrevious className={controlClassName} />
        <CarouselNext className={controlClassName} />
      </div>
    </Carousel>
  );
}

export function ConnectedCampaign() {
  return (
    <section className="relative isolate overflow-hidden text-white">
      <SectionMeshBackground src={DUSK_MESH_GRADIENT_SRC} />
      <div aria-hidden className="absolute inset-0 -z-10 bg-black/30" />
      <div className="mx-auto max-w-7xl px-5 py-20 sm:px-8 sm:py-28 lg:px-10">
        <div className="mx-auto mb-14 max-w-3xl text-center">
          <p className="mb-5 text-xs text-white/70">
            <FormattedMessage {...m.workflowEyebrow} />
          </p>
          <h2 className="font-heading text-[clamp(2.25rem,4vw,3.5rem)] leading-[1.1] text-balance">
            <FormattedMessage {...m.journeyTitle} />
          </h2>
          <p className="mx-auto mt-6 max-w-2xl text-base leading-7 text-pretty text-white/80">
            <FormattedMessage {...m.journeyBody} />
          </p>
        </div>
        <JourneyCarousel />
      </div>
    </section>
  );
}

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
import type { ReactNode } from "react";

import { motion, useReducedMotion } from "motion/react";
import dynamic from "next/dynamic";
import Image from "next/image";

import { cn } from "@/lib/primitives/cn";

export function HeroFrameLoadingShell() {
  return (
    <div
      aria-hidden
      className="overflow-hidden rounded-2xl border border-border bg-background shadow-2xl"
    >
      <div className="flex h-[min(42rem,78svh)] min-h-136 flex-col bg-muted/20" />
    </div>
  );
}

const ClientOnlyHeroFrame = dynamic(
  () => import("./hero-frame").then((module) => module.HeroFrame),
  {
    loading: HeroFrameLoadingShell,
    ssr: false,
  },
);

export const SEAFOAM_MESH_GRADIENT_SRC = "/images/mesh/mesh-gradient-1784864145512.jpg";
export const LAVENDER_MESH_GRADIENT_SRC = "/images/mesh/mesh-gradient-1784864042890.jpg";
export const SAGE_MESH_GRADIENT_SRC = "/images/mesh/mesh-gradient-1784864073608.jpg";
export const DUSK_MESH_GRADIENT_SRC = "/images/mesh/mesh-gradient-1784863799475.jpg";

/** Full-bleed section mesh. Uses the source JPG (no optimizer) so soft gradients stay smooth. */
export function SectionMeshBackground({
  src,
  priority = false,
  className,
}: {
  src: string;
  priority?: boolean;
  className?: string;
}) {
  return (
    <Image
      src={src}
      alt=""
      aria-hidden
      fill
      priority={priority}
      unoptimized
      sizes="100vw"
      className={cn("-z-20 object-cover object-center", className)}
    />
  );
}

type MeshStageProps = {
  children: ReactNode;
  className?: string;
  contentClassName?: string;
  priority?: boolean;
  /** Stretch near the viewport edges; otherwise fill the parent width. */
  layout?: "breakout" | "contained";
  /** Mesh image source. Defaults to the seafoam gradient used by the CAT stage. */
  meshSrc?: string;
  /** Scroll-into-view entrance. Use `none` when children animate themselves (e.g. tab crossfades). */
  entranceAnimation?: "default" | "fade" | "none";
};

export function MeshStage({
  children,
  className,
  contentClassName,
  priority = false,
  layout = "contained",
  meshSrc = SEAFOAM_MESH_GRADIENT_SRC,
  entranceAnimation = "default",
}: MeshStageProps) {
  const shouldReduceMotion = useReducedMotion();

  const content =
    entranceAnimation === "none" || shouldReduceMotion ? (
      children
    ) : (
      <motion.div
        initial={entranceAnimation === "fade" ? { opacity: 0 } : { opacity: 0, y: 24, scale: 0.98 }}
        whileInView={{ opacity: 1, ...(entranceAnimation === "fade" ? {} : { y: 0, scale: 1 }) }}
        viewport={{ once: true, amount: 0.25 }}
        transition={{
          duration: entranceAnimation === "fade" ? 0.35 : 0.72,
          ease: [0.19, 1, 0.22, 1],
        }}
      >
        {children}
      </motion.div>
    );

  return (
    <div
      className={cn(
        layout === "breakout"
          ? "relative left-1/2 w-screen max-w-[calc(100vw-2.5rem)] -translate-x-1/2 lg:max-w-[min(92rem,calc(100vw-5rem))]"
          : "relative w-full",
        className,
      )}
    >
      <div className="relative overflow-hidden rounded-[1.5rem] shadow-[0_20px_48px_rgba(0,0,0,0.18)] sm:rounded-[2rem] sm:shadow-[0_32px_80px_rgba(0,0,0,0.22)]">
        <Image
          src={meshSrc}
          alt=""
          aria-hidden
          fill
          priority={priority}
          sizes="(min-width: 1280px) 92rem, 100vw"
          className="pointer-events-none object-cover object-center"
        />
        <div className={cn("relative p-3 sm:p-5 lg:p-8 xl:p-10", contentClassName)}>{content}</div>
      </div>
    </div>
  );
}

type HeroFrameMeshStageProps = {
  className?: string;
  priority?: boolean;
};

export function HeroFrameMeshStage({ className, priority = false }: HeroFrameMeshStageProps) {
  return (
    <MeshStage layout="breakout" className={className} priority={priority}>
      <ClientOnlyHeroFrame layout="contained" className="shadow-[0_24px_64px_rgba(0,0,0,0.28)]" />
    </MeshStage>
  );
}

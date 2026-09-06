# Platform homepage redesign

Approved direction: position Hyperlocalise as AI-native infrastructure for multilingual content operations.

## Structure

1. Platform hero with a substantial illustrative workspace preview.
2. Five product pillars: Content Studio, Automation Workflow, Domains, Hyperlab, Guidelines.
3. Agents in Slack, Teams, GitHub and other integrations, plus MCP as an access layer.
4. Accessible, user-controlled product explorer tabs.
5. Connected creation, orchestration, publishing and optimisation workflow with Guidelines as shared context.
6. Existing customer proof, pricing entry point, FAQ, final demo CTA and footer.

Domains covers CMS publishing and automated AEO/SEO operations. Hyperlab covers A/B testing for CMS/distribution. Translation is a capability within the platform, not its category.

Keep brand typography and theme support. Use a deep teal hero, spacious neutral sections, restrained product accents, and responsive previews. Product illustrations must be labelled as examples, without fabricated performance claims. Preserve authentication-aware demo/dashboard actions, locale-aware links, existing customer proof and FAQ structured data. Validate with vp test, vp check --fix, and browser checks of desktop/mobile and explorer interaction.

## Refinement after the first preview

Use existing mesh imagery for the hero, product cards and campaign section. Shorten the display headline to “Every market. One content operation.” while keeping the exact platform category above it. The explorer uses the existing interactive content editor, workflow, discovery, Hyperlab and Guidelines mocks; Domains opens with an interactive multilingual publishing preview. Expand the connected workflow into a three-market campaign with specific actions and outputs for every pillar.

## Validation

`vp check --fix` passes (eight existing warnings). Marketing tests: 14 files and 30 tests pass. The full suite with local database access passed 882 files and 6,486 tests, with eight failures in four reporting/runtime test files outside the homepage changes. Failures include local reporting schema constraints and sandbox script assumptions. Full-suite validation remains unresolved. Browser checks cover the desktop and 390px mobile hero, tab switching, the Hyperlab experiment selector, and French CMS preview selection.

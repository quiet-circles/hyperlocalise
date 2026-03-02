---
name: hyperlocalise
description: Guide users to use the Hyperlocalise CLI for AI-powered localization workflows. Use when users ask how to set up, configure, run, evaluate, or sync translations with commands like `init`, `run`, `status`, `sync pull`, `sync push`, and `eval`, or when they need CLI troubleshooting and best-practice command sequences.
---

# Hyperlocalise Skill

Use this skill to help users operate Hyperlocalise CLI projects safely and efficiently.

## Install with skills.sh

```bash
npx skills add https://github.com/quiet-circles/hyperlocalise --skill hyperlocalise
```

## Work in User-Operations Mode

- Prioritize actionable CLI guidance over implementation details.
- Recommend concrete commands with short explanations of intent.
- Keep users on stable, repeatable workflows (`init` -> configure -> `run` -> `status` -> `sync`/`eval`).

## Default CLI Workflow

1. Initialize project scaffolding:

```bash
hyperlocalise init
```

2. Configure `i18n.jsonc` with locales, providers, and paths.
3. Generate or update translations:

```bash
hyperlocalise run
```

4. Inspect status and pending changes:

```bash
hyperlocalise status
```

5. Sync with translation storage when enabled:

```bash
hyperlocalise sync pull
hyperlocalise sync push
```

6. Evaluate translation quality when needed:

```bash
hyperlocalise eval run
```

## Command Guidance Rules

- Suggest `status` before and after `run`/`sync` to make changes visible.
- For destructive or broad operations, recommend backing up or using VCS before proceeding.
- Prefer incremental runs and clear diff review over large one-shot changes.
- If provider credentials are missing, guide user to set env vars first, then retry command.
- When users ask “what next?”, provide the next 1-2 commands only.

## Troubleshooting Playbook

- Config errors: check `i18n.jsonc` keys and locale/path mapping first.
- Auth/provider errors: verify required environment variables and provider selection.
- File parsing errors: identify the exact file and format (`json`, `strings`, `po`, `xliff`, `xlf`, `md`, `mdx`, `csv`) and isolate a minimal repro.
- Sync mismatches: run `sync pull` first, inspect status, then run `sync push` intentionally.
- Quality concerns: run `eval` and recommend prompt/provider/config adjustments based on findings.

## Response Style for This Skill

- Use concise, command-first instructions.
- Include copy-pastable command blocks.
- State assumptions explicitly (platform, provider, project root) when not given.
- If blocked by missing user config/credentials, ask only for the minimum required input.

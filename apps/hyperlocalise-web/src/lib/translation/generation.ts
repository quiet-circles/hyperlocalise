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
import { randomUUID } from "node:crypto";
import { captureAiUsage } from "@/lib/reporting/ai-cost";
import { generateText, Output, type LanguageModel } from "ai";
import { eq } from "drizzle-orm";
import { z } from "zod";

import { DEFAULT_APP_LOCALE, normalizeAppLocale } from "@/lib/app-i18n/locales";
import { db, schema } from "@/lib/database/client";
import type { LlmProvider } from "@/lib/database/types";
import {
  getManagedLanguageModel,
  resolveProviderLanguageModel,
} from "@/lib/providers/language-model";
import { loadLatestOrganizationProviderCredential } from "@/lib/providers/organization-language-model";
import type {
  ContentEditorAiRecommendationInput,
  ContentEditorAiRecommendationResult,
  SandboxTranslationContext,
  StringTranslationGenerator,
  StringTranslationGeneratorInput,
  StringTranslationJobResult,
} from "@/lib/translation/domain";

function resolveCatRecommendationDisplayLocale(displayLocale?: string): string {
  const trimmed = displayLocale?.trim();
  if (!trimmed) {
    return DEFAULT_APP_LOCALE;
  }

  return normalizeAppLocale(trimmed) ?? trimmed;
}

const stringTranslationOutputSchema = z.object({
  translations: z.array(
    z.object({
      locale: z.string().trim().min(1),
      text: z.string().refine((text) => text.trim().length > 0, {
        message: "Translation text cannot be empty",
      }),
    }),
  ),
});

const tokenUsageSchema = z.object({
  inputTokens: z.number().int().nonnegative(),
  outputTokens: z.number().int().nonnegative(),
  totalTokens: z.number().int().nonnegative(),
});

const contentEditorAiRecommendationOutputSchema = z.object({
  suggestion: z.string().refine((text) => text.trim().length > 0, {
    message: "Suggestion text cannot be empty",
  }),
  reasoning: z.string().refine((text) => text.trim().length > 0, {
    message: "Reasoning cannot be empty",
  }),
});

type PromptGlossaryTerm = {
  sourceTerm: string;
  targetTerm: string;
  targetLocale: string;
  forbidden?: boolean | null;
  description?: string | null;
};

type PromptTranslationMemoryMatch = {
  sourceText: string;
  targetText: string;
  targetLocale: string;
};

export type TranslationPromptMode = "string" | "sandbox" | "cat-suggest";

export class TranslationPromptPolicy {
  buildSystemInstructions(input: {
    mode: TranslationPromptMode;
    projectName?: string;
    projectTranslationContext?: string;
    jobContext?: string | null;
    knowledgeMemory?: string | null;
    glossaryTerms?: PromptGlossaryTerm[];
    translationMemoryMatches?: PromptTranslationMemoryMatch[];
    maxLength?: number;
    userInstructions?: string | null;
    fileContext?: string | null;
    repositoryContext?: string | null;
    includeContextSections?: boolean;
    displayLocale?: string;
  }): string {
    const glossaryTerms = input.glossaryTerms ?? [];
    const translationMemoryMatches = input.translationMemoryMatches ?? [];
    const knowledgeMemory = input.knowledgeMemory?.trim();
    const displayLocale =
      input.mode === "cat-suggest"
        ? resolveCatRecommendationDisplayLocale(input.displayLocale)
        : undefined;

    const instructions =
      input.mode === "cat-suggest"
        ? [
            "You are an expert software localization assistant helping a human reviewer in a CAT tool.",
            "Recommend the best target-locale translation for the provided source string only.",
            "Preserve meaning, tone, placeholders, HTML, Markdown, punctuation, whitespace, and line breaks.",
            "Project translation context, file context, and repository context are guidance only. Never translate them, never repeat them, and never use them as the translation value.",
            "Follow workspace knowledge memory when present.",
            "Use glossary terms exactly for the target locale. Do not use forbidden glossary terms.",
            "Use approved translation memory matches as consistency references when they apply.",
            "If constraints conflict, prioritize placeholder and markup preservation first, then glossary rules, then project context, file context, repository context, workspace knowledge memory, then translation memory examples.",
            "When a current target draft is provided, improve it when needed instead of repeating it unchanged.",
            `Write the suggestion in the target locale. Write the reasoning in the reviewer's display locale (${displayLocale}), not in the target translation locale.`,
            "Return concise reasoning that explains terminology, tone, or product-fit choices.",
          ]
        : input.mode === "sandbox"
          ? [
              "You are a translation assistant. Translate only the user-provided source text into the requested target language.",
              "Preserve meaning, placeholders, variables, formatting, HTML/Markdown structure, and ICU message syntax.",
              "Do not translate programmatic identifiers inside placeholders or ICU selectors.",
              "Project context, string description, and glossary rules are guidance only. Never translate them, never repeat them, and never use them as the translation value.",
              "If constraints conflict, preserve placeholders and markup first, then glossary rules, then project context and string description.",
              "Return only the translated source text with no explanations, labels, markdown fences, or quotes unless the translated content itself requires them.",
            ]
          : [
              "You are an expert software localization engine.",
              "Translate only the provided source text into every requested target locale.",
              "Preserve meaning, tone, placeholders, HTML, Markdown, punctuation, whitespace, and line breaks.",
              "Project translation context and string description are developer guidance for meaning and tone. Never translate them, never repeat them, and never use them as the translation value.",
              "Follow workspace knowledge memory when present.",
              "Use glossary terms exactly for their target locale. Do not use forbidden glossary terms.",
              "Use approved translation memory matches as consistency references when they apply.",
              "If constraints conflict, prioritize placeholder and markup preservation first, then glossary rules, then project context, string description, workspace knowledge memory, then translation memory examples.",
              "Do not explain your work.",
              "Return one translation for each requested locale.",
            ];

    if (input.maxLength) {
      instructions.push(
        input.mode === "string"
          ? `Each translated string must be at most ${input.maxLength} characters long.`
          : `Maximum length: ${input.maxLength} characters`,
      );
    }

    const sections = [...instructions, ""];

    if (input.includeContextSections === false) {
      return sections.filter(Boolean).join("\n");
    }

    if (input.mode === "cat-suggest") {
      sections.push(
        `Project name: ${input.projectName ?? "(none)"}`,
        `Reviewer display locale: ${displayLocale}`,
        `Project translation context: ${input.projectTranslationContext?.trim() || "(none)"}`,
        `Workspace knowledge memory: ${knowledgeMemory || "(none)"}`,
      );
    } else {
      sections.push(
        input.projectName ? `Project: ${input.projectName}` : "",
        `Project translation context: ${input.projectTranslationContext?.trim() || "(none)"}`,
        `String description (guidance only; do not translate or use as the translation): ${input.jobContext?.trim() || "(none)"}`,
        `Workspace knowledge memory: ${knowledgeMemory || "(none)"}`,
        input.userInstructions?.trim()
          ? `User style instructions: ${input.userInstructions.trim()}`
          : "",
      );
    }

    sections.push(
      glossaryTerms.length > 0
        ? [
            "Glossary terms:",
            ...glossaryTerms.map((term) =>
              [
                `- ${term.sourceTerm} -> ${term.targetTerm} (${term.targetLocale})`,
                term.forbidden ? "forbidden" : null,
                term.description ? `note: ${term.description}` : null,
              ]
                .filter(Boolean)
                .join("; "),
            ),
          ].join("\n")
        : "Glossary terms: (none)",
      translationMemoryMatches.length > 0
        ? [
            "Translation memory matches:",
            ...translationMemoryMatches.map(
              (match) => `- ${match.sourceText} -> ${match.targetText} (${match.targetLocale})`,
            ),
          ].join("\n")
        : "Translation memory matches: (none)",
    );

    if (input.mode === "cat-suggest") {
      if (input.fileContext?.trim()) {
        sections.push(
          `File context (guidance only; do not translate or use as the translation): ${input.fileContext.trim()}`,
        );
      }
      if (input.repositoryContext?.trim()) {
        sections.push(
          `Repository context (guidance only; do not translate or use as the translation): ${input.repositoryContext.trim()}`,
        );
      }
    }

    return sections.filter(Boolean).join("\n");
  }

  buildSandboxConfigPrompt(
    context: SandboxTranslationContext | null,
    instructions: string | null,
  ): string {
    const hasContext =
      Boolean(context?.projectName?.trim()) ||
      Boolean(context?.projectTranslationContext?.trim()) ||
      Boolean(context?.jobContext?.trim()) ||
      (context?.glossaryTerms?.length ?? 0) > 0;

    if (!hasContext && !instructions?.trim()) {
      return this.buildSystemInstructions({
        mode: "sandbox",
        includeContextSections: false,
      });
    }

    return this.buildSystemInstructions({
      mode: "sandbox",
      projectName: context?.projectName ?? undefined,
      projectTranslationContext: context?.projectTranslationContext ?? undefined,
      jobContext: context?.jobContext,
      glossaryTerms: context?.glossaryTerms,
      userInstructions: instructions,
    });
  }
}

function estimateMaxOutputTokens(sourceText: string, targetLocaleCount: number) {
  const sourceBudget = Math.ceil(sourceText.length / 2);
  const localeBudget = targetLocaleCount * 256;
  return Math.min(16_000, Math.max(1_000, sourceBudget + localeBudget));
}

function estimateCatRecommendationMaxOutputTokens(input: ContentEditorAiRecommendationInput) {
  const sourceBudget = Math.ceil(input.sourceText.length / 2);
  const contextBudget = Math.ceil(
    ((input.context?.length ?? 0) + (input.agentContext?.length ?? 0)) / 4,
  );
  return Math.min(4_000, Math.max(512, sourceBudget + contextBudget + 256));
}

function normalizeTranslations(
  jobInput: StringTranslationGeneratorInput["jobInput"],
  result: z.infer<typeof stringTranslationOutputSchema>,
  tokenUsage?: StringTranslationJobResult["tokenUsage"],
): StringTranslationJobResult {
  const translationsByLocale = new Map<string, string>();

  for (const translation of result.translations) {
    if (translationsByLocale.has(translation.locale)) {
      throw new Error(`duplicate translation returned for locale ${translation.locale}`);
    }

    translationsByLocale.set(translation.locale, translation.text);
  }

  for (const locale of new Set(jobInput.targetLocales)) {
    if (!translationsByLocale.has(locale)) {
      throw new Error(`missing translation for locale ${locale}`);
    }
  }

  for (const locale of translationsByLocale.keys()) {
    if (!jobInput.targetLocales.includes(locale)) {
      throw new Error(`unexpected translation locale ${locale}`);
    }
  }

  const translations = [...new Set(jobInput.targetLocales)].map((locale) => {
    const text = translationsByLocale.get(locale);

    if (!text) {
      throw new Error(`missing translation for locale ${locale}`);
    }

    if (jobInput.maxLength && text.length > jobInput.maxLength) {
      throw new Error(`translation for locale ${locale} exceeds maxLength`);
    }

    return { locale, text };
  });

  return { translations, ...(tokenUsage ? { tokenUsage } : {}) };
}

function normalizeAiSdkTokenUsage(
  usage: unknown,
): StringTranslationJobResult["tokenUsage"] | undefined {
  const rawUsage = usage as
    | {
        inputTokens?: number;
        outputTokens?: number;
        totalTokens?: number;
      }
    | undefined;

  const inputTokens = rawUsage?.inputTokens ?? 0;
  const outputTokens = rawUsage?.outputTokens ?? 0;
  const totalTokens = rawUsage?.totalTokens ?? inputTokens + outputTokens;
  const parsedUsage = tokenUsageSchema.safeParse({ inputTokens, outputTokens, totalTokens });

  return parsedUsage.success ? parsedUsage.data : undefined;
}

export { resolveProviderLanguageModel } from "@/lib/providers/language-model";

export class OrganizationModelResolver {
  async resolve(projectId: string) {
    const [project] = await db
      .select({
        name: schema.projects.name,
        translationContext: schema.projects.translationContext,
        organizationId: schema.projects.organizationId,
      })
      .from(schema.projects)
      .where(eq(schema.projects.id, projectId))
      .limit(1);

    if (!project) {
      return {
        ok: false as const,
        code: "translation_project_not_found" as const,
        message: `translation project ${projectId} was not found`,
      };
    }

    const loadedCredential = await loadLatestOrganizationProviderCredential(project.organizationId);
    const projectContext = {
      name: project.name,
      translationContext: project.translationContext,
    };

    if (!loadedCredential.ok) {
      return {
        ok: false as const,
        code: loadedCredential.code,
        message: loadedCredential.message,
      };
    }

    if (loadedCredential.credential) {
      const model = resolveProviderLanguageModel({
        provider: loadedCredential.credential.provider,
        apiKey: loadedCredential.credential.apiKey,
        model: loadedCredential.credential.model,
      });

      return {
        ok: true as const,
        project: projectContext,
        model,
        translateStringJob: createStringTranslationGenerator({ model }),
      };
    }

    const model = getManagedTranslationLanguageModel();
    return {
      ok: true as const,
      project: projectContext,
      model,
      translateStringJob: createManagedStringTranslationGenerator(),
    };
  }

  async resolveModel(projectId: string) {
    const setup = await this.resolve(projectId);
    if (!setup.ok) {
      return setup;
    }

    return {
      ok: true as const,
      project: setup.project,
      model: setup.model,
    };
  }
}

export class StringTranslationEngine {
  constructor(
    private readonly model: LanguageModel,
    private readonly promptPolicy = new TranslationPromptPolicy(),
  ) {}

  async translate(input: StringTranslationGeneratorInput): Promise<StringTranslationJobResult> {
    const invocationId = randomUUID();
    const { output, usage } = await generateText({
      model: this.model,
      output: Output.object({ schema: stringTranslationOutputSchema }),
      instructions: this.promptPolicy.buildSystemInstructions({
        mode: "string",
        projectTranslationContext: input.projectTranslationContext,
        jobContext: input.jobInput.context,
        knowledgeMemory: input.contextSnapshot?.knowledgeMemory,
        glossaryTerms: input.contextSnapshot?.glossaryTerms,
        translationMemoryMatches: input.contextSnapshot?.translationMemoryMatches,
        maxLength: input.jobInput.maxLength,
      }),
      prompt: [
        `Project: ${input.projectName}`,
        `Source locale: ${input.jobInput.sourceLocale}`,
        `Target locales: ${input.jobInput.targetLocales.join(", ")}`,
        `Metadata: ${JSON.stringify(input.jobInput.metadata ?? {})}`,
        "Source text:",
        input.jobInput.sourceText,
      ].join("\n\n"),
      temperature: 0,
      maxOutputTokens: estimateMaxOutputTokens(
        input.jobInput.sourceText,
        input.jobInput.targetLocales.length,
      ),
    });

    if (input.reporting) {
      const model = typeof this.model === "string" ? this.model : this.model.modelId;
      const provider = typeof this.model === "string" ? "gateway" : this.model.provider;
      try {
        await captureAiUsage({
          ...input.reporting,
          invocationId,
          targetLocale:
            input.jobInput.targetLocales.length === 1 ? input.jobInput.targetLocales[0] : undefined,
          model,
          provider,
          inputTokens: usage.inputTokens ?? 0,
          outputTokens: usage.outputTokens ?? 0,
        });
      } catch (error) {
        console.warn("[string-translation] usage capture failed", {
          jobId: input.reporting.jobId,
          error,
        });
      }
    }
    return normalizeTranslations(input.jobInput, output, normalizeAiSdkTokenUsage(usage));
  }
}

export class ContentEditorRecommendationEngine {
  constructor(
    private readonly model: LanguageModel,
    private readonly promptPolicy = new TranslationPromptPolicy(),
  ) {}

  async recommend(
    input: ContentEditorAiRecommendationInput,
    context: {
      projectName: string;
      projectTranslationContext: string;
      knowledgeMemory?: string;
      glossaryTerms: PromptGlossaryTerm[];
      translationMemoryMatches: PromptTranslationMemoryMatch[];
    },
    options?: { signal?: AbortSignal },
  ): Promise<ContentEditorAiRecommendationResult> {
    const { output } = await generateText({
      model: this.model,
      abortSignal: options?.signal,
      output: Output.object({ schema: contentEditorAiRecommendationOutputSchema }),
      instructions: this.promptPolicy.buildSystemInstructions({
        mode: "cat-suggest",
        projectName: context.projectName,
        projectTranslationContext: context.projectTranslationContext,
        knowledgeMemory: context.knowledgeMemory,
        glossaryTerms: context.glossaryTerms,
        translationMemoryMatches: context.translationMemoryMatches,
        maxLength: input.maxLength,
        fileContext: input.context,
        repositoryContext: input.agentContext,
        displayLocale: input.displayLocale,
      }),
      prompt: [
        `Project file: ${input.filename}`,
        `Source path: ${input.sourcePath}`,
        `String key: ${input.key}`,
        `Source locale: ${input.sourceLocale}`,
        `Target locale: ${input.targetLocale}`,
        `Reviewer display locale: ${resolveCatRecommendationDisplayLocale(input.displayLocale)}`,
        input.maxLength ? `Maximum length: ${input.maxLength} characters` : null,
        input.targetText?.trim() ? ["Current target draft:", input.targetText].join("\n") : null,
        "Source text:",
        input.sourceText,
      ]
        .filter(Boolean)
        .join("\n\n"),
      temperature: 0,
      maxOutputTokens: estimateCatRecommendationMaxOutputTokens(input),
    });

    if (input.maxLength && output.suggestion.length > input.maxLength) {
      throw new Error(`recommendation exceeds maxLength of ${input.maxLength}`);
    }

    return {
      aiSuggestion: output.suggestion,
      aiReasoning: output.reasoning,
    };
  }
}

export function createStringTranslationGenerator(input: {
  model: LanguageModel;
}): StringTranslationGenerator {
  const engine = new StringTranslationEngine(input.model);
  return (generatorInput) => engine.translate(generatorInput);
}

export function createProviderStringTranslationGenerator(input: {
  provider: LlmProvider;
  apiKey: string;
  model: string;
}): StringTranslationGenerator {
  return createStringTranslationGenerator({
    model: resolveProviderLanguageModel(input),
  });
}

export function createOpenAIStringTranslationGenerator(input: {
  apiKey: string;
  model: string;
}): StringTranslationGenerator {
  return createProviderStringTranslationGenerator({
    provider: "openai",
    apiKey: input.apiKey,
    model: input.model,
  });
}

export function getManagedTranslationLanguageModel(): LanguageModel {
  return getManagedLanguageModel();
}

export function createManagedStringTranslationGenerator(): StringTranslationGenerator {
  return createStringTranslationGenerator({ model: getManagedTranslationLanguageModel() });
}

export const translateStringJobWithOpenAI: StringTranslationGenerator = async (input) => {
  return createStringTranslationGenerator({ model: getManagedLanguageModel() })(input);
};

const defaultModelResolver = new OrganizationModelResolver();

export async function loadOrganizationTranslationGenerator(projectId: string) {
  const setup = await defaultModelResolver.resolve(projectId);
  if (!setup.ok) {
    return setup;
  }

  return {
    ok: true as const,
    project: setup.project,
    translateStringJob: setup.translateStringJob,
  };
}

export async function loadOrganizationTranslationModel(projectId: string) {
  return defaultModelResolver.resolveModel(projectId);
}

/** @deprecated Use `loadOrganizationTranslationGenerator` instead. */
export const loadOrganizationOpenAITranslationGenerator = loadOrganizationTranslationGenerator;

export const translationPromptPolicy = new TranslationPromptPolicy();

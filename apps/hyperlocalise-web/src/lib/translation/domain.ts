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
import type { StringTranslationJobInput } from "@/api/routes/project/job.schema";
import type * as schema from "@/lib/database/schema";
import type { ContextGlossaryMatch } from "@/lib/providers/contracts/glossary-match";
import type { ContextTranslationMemoryMatch } from "@/lib/providers/contracts/translation-memory-match";

export { normalizeTranslationMemorySourceText } from "@/lib/translation/normalizeTranslationMemorySourceText";

export type {
  AgentRunGlossaryMatchUsage,
  ContextGlossaryMatch,
  GlossaryMatchSource,
  NormalizedGlossaryMatch,
} from "@/lib/providers/contracts/glossary-match";

export type {
  AgentRunTranslationMemoryMatchUsage,
  ContextTranslationMemoryMatch,
  NormalizedTranslationMemoryMatch,
  TranslationMemoryMatchSource,
} from "@/lib/providers/contracts/translation-memory-match";

export {
  mergeGlossaryMatches,
  toAgentRunGlossaryMatchUsage,
  toContextGlossaryMatch,
} from "@/lib/providers/contracts/glossary-match";

export {
  mergeTranslationMemoryMatches,
  toAgentRunTranslationMemoryMatchUsage,
  toContextTranslationMemoryMatch,
} from "@/lib/providers/contracts/translation-memory-match";

export type TranslationProjectContext = {
  id: string;
  name: string;
  translationContext: string;
};

export type TranslationContextProjectRecord = TranslationProjectContext &
  Pick<
    typeof schema.projects.$inferSelect,
    | "organizationId"
    | "source"
    | "externalProviderKind"
    | "externalProjectId"
    | "externalProviderCredentialId"
    | "providerMetadata"
  >;

export type SourceSegmentContext = StringTranslationJobInput;

export type StringTranslationContextSnapshot = {
  assembledAt: string;
  project: TranslationProjectContext;
  knowledgeMemory?: string;
  job: StringTranslationJobInput;
  glossaryTerms: ContextGlossaryMatch[];
  translationMemoryMatches: ContextTranslationMemoryMatch[];
};

export type SandboxTranslationContext = {
  projectName?: string | null;
  projectTranslationContext?: string | null;
  jobContext?: string | null;
  glossaryTerms?: Array<{
    sourceTerm: string;
    targetTerm: string;
    targetLocale: string;
    forbidden?: boolean | null;
    caseSensitive?: boolean | null;
    description?: string | null;
  }>;
};

export type StringTranslationGeneratorInput = {
  reporting?: { organizationId: string; projectId: string; jobId: string };
  projectName: string;
  projectTranslationContext: string;
  jobInput: StringTranslationJobInput;
  contextSnapshot?: {
    knowledgeMemory?: string;
    glossaryTerms?: Array<{
      sourceTerm: string;
      targetTerm: string;
      targetLocale: string;
      forbidden?: boolean | null;
      description?: string | null;
    }>;
    translationMemoryMatches?: Array<{
      sourceText: string;
      targetText: string;
      targetLocale: string;
      provenance?: string | null;
      rank?: number;
    }>;
  };
};

export type StringTranslationJobResult = {
  translations: Array<{ locale: string; text: string }>;
  tokenUsage?: {
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
  };
};

export type StringTranslationGenerator = (
  input: StringTranslationGeneratorInput,
) => Promise<StringTranslationJobResult>;

export type ContentEditorAiRecommendationInput = {
  projectId: string;
  organizationId: string;
  sourcePath: string;
  filename: string;
  sourceLocale: string;
  targetLocale: string;
  /** Locale for reviewer-facing reasoning text (UI/display locale), not the translation target. */
  displayLocale?: string;
  key: string;
  sourceText: string;
  targetText?: string;
  context?: string | null;
  agentContext?: string | null;
  maxLength?: number;
  glossaryTerms?: Array<{
    sourceTerm: string;
    targetTerm: string;
    targetLocale: string;
    forbidden?: boolean | null;
    description?: string | null;
  }>;
  translationMemoryMatches?: Array<{
    sourceText: string;
    targetText: string;
    targetLocale: string;
  }>;
};

export type ContentEditorAiRecommendationResult = {
  aiSuggestion: string;
  aiReasoning: string;
};

export type ContentEditorAiRecommendationError = {
  code:
    | "translation_project_not_found"
    | "provider_credential_invalid"
    | "provider_credential_missing"
    | "translation_context_assembly_failed"
    | "ai_recommendation_failed"
    | "ai_features_required"
    | "ai_features_check_failed";
  message: string;
};

export class TranslationContext {
  constructor(
    readonly assembledAt: string,
    readonly project: TranslationProjectContext,
    readonly source: SourceSegmentContext,
    readonly knowledgeMemory: string | null,
    readonly glossaryTerms: readonly ContextGlossaryMatch[],
    readonly translationMemoryMatches: readonly ContextTranslationMemoryMatch[],
  ) {}

  toSnapshot(): StringTranslationContextSnapshot {
    return {
      assembledAt: this.assembledAt,
      project: this.project,
      job: this.source,
      glossaryTerms: [...this.glossaryTerms],
      translationMemoryMatches: [...this.translationMemoryMatches],
      ...(this.knowledgeMemory ? { knowledgeMemory: this.knowledgeMemory } : {}),
    };
  }

  toStringTranslationInput(
    projectName: string,
    projectTranslationContext: string,
  ): StringTranslationGeneratorInput {
    return {
      projectName,
      projectTranslationContext,
      jobInput: this.source,
      contextSnapshot: {
        ...(this.knowledgeMemory ? { knowledgeMemory: this.knowledgeMemory } : {}),
        glossaryTerms: [...this.glossaryTerms],
        translationMemoryMatches: [...this.translationMemoryMatches],
      },
    };
  }

  toSandboxContext(): SandboxTranslationContext {
    return {
      projectName: this.project.name,
      projectTranslationContext: this.project.translationContext,
      jobContext: this.source.context ?? null,
      glossaryTerms: this.glossaryTerms.map((term) => ({
        sourceTerm: term.sourceTerm,
        targetTerm: term.targetTerm,
        targetLocale: term.targetLocale,
        forbidden: term.forbidden,
        description: term.description,
      })),
    };
  }

  toCatRecommendationContext(): Pick<
    ContentEditorAiRecommendationInput,
    "glossaryTerms" | "translationMemoryMatches"
  > {
    return {
      glossaryTerms: this.glossaryTerms.map((term) => ({
        sourceTerm: term.sourceTerm,
        targetTerm: term.targetTerm,
        targetLocale: term.targetLocale,
        forbidden: term.forbidden,
        description: term.description,
      })),
      translationMemoryMatches: this.translationMemoryMatches.map((match) => ({
        sourceText: match.sourceText,
        targetText: match.targetText,
        targetLocale: match.targetLocale,
      })),
    };
  }
}

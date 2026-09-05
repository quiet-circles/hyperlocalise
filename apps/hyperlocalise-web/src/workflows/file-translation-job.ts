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
import { getWorkflowMetadata } from "workflow";

import { validateGlossaryTermsInTranslation } from "@/lib/glossary/validate-glossary-terms-in-translation";
import { mergeTranslationPrefills } from "@/lib/projects/translations/should-retry-same-as-source-prefill";
import { hlEntriesPayloadToStringMap } from "@/lib/projects/files/hl-entries";
import {
  inferSupportedFileTranslationFileFormat,
  isImageTranslationFileFormat,
  isOfficeTranslationFileFormat,
  isDocumentTranslationFileFormat,
  isVideoTranslationFileFormat,
  isSupportedFileTranslationFileFormat,
  type SupportedTranslationFileFormat,
} from "@/lib/translation/file-formats";
import type { SandboxTranslationContext } from "@/lib/translation/domain";
import type { TranslationJobEventData } from "@/lib/workflow/types";
import {
  captureFileAnalysisStep,
  captureFileCompletionsStep,
  claimTranslationJobStep,
  completeFileTranslationJobStep,
  ensureAiFeaturesAllowedStep,
  failTranslationJobStep,
  getProjectOrganizationStep,
  getStoredFileContentStep,
  getStoredFileStep,
  getRepositorySourcePathForStoredFileStep,
  loadProjectTranslationsAsPrefilledEntriesStep,
  localizeImageVariantForJobStep,
  localizeVideoVariantForJobStep,
  persistFileProjectTranslationsStep,
  persistDocumentVariantBytesStep,
  persistFileTranslationMemoryEntriesStep,
  reuseFileTranslationMemoryEntriesStep,
  storeOutputFileStep,
} from "./steps/translation-job";
import {
  FILE_TRANSLATION_MAX_TRANSLATIONS_PER_SESSION,
  calculateFileTranslationMaxPages,
  calculateFileTranslationSandboxTimeoutMs,
  countPendingFileTranslations,
  parseDeferredByLimit,
} from "./file-translation-pagination";

function shellSingleQuote(value: string) {
  return value.replaceAll("'", "'\\''");
}

function sanitizeSandboxFilename(filename: string): string {
  return filename.replace(/[^a-zA-Z0-9._-]/g, "_");
}

function getSandboxInputFilename(attachmentFilename: string): string {
  return sanitizeSandboxFilename(attachmentFilename);
}

function getSandboxOutputFilename(attachmentFilename: string, targetLocale: string): string {
  const inputFilename = sanitizeSandboxFilename(attachmentFilename);
  const lastDot = inputFilename.lastIndexOf(".");
  if (lastDot === -1) {
    return `${inputFilename}-${targetLocale}`;
  }

  const name = inputFilename.slice(0, lastDot);
  const ext = inputFilename.slice(lastDot);
  return `${name}-${targetLocale}${ext}`;
}

function getSandboxOutputFilenamePattern(attachmentFilename: string): string {
  const inputFilename = sanitizeSandboxFilename(attachmentFilename);
  const lastDot = inputFilename.lastIndexOf(".");
  if (lastDot === -1) {
    return `${inputFilename}-{{target}}`;
  }

  const name = inputFilename.slice(0, lastDot);
  const ext = inputFilename.slice(lastDot);
  return `${name}-{{target}}${ext}`;
}

function fileExtension(filename: string): string | null {
  const lastDot = filename.lastIndexOf(".");
  if (lastDot === -1 || lastDot === filename.length - 1) {
    return null;
  }
  return filename.slice(lastDot).toLowerCase();
}

/** Classify CLI failures using only known safe substrings — never log raw CLI output. */
function classifyCliFailureKind(output: string): string {
  if (output.includes("markdown AST parity mismatch")) {
    return "markdown_ast_parity_mismatch";
  }
  if (output.includes("markdown parity retry exhausted")) {
    return "markdown_parity_retry_exhausted";
  }
  if (output.includes("placeholder parity")) {
    return "placeholder_parity_mismatch";
  }
  if (output.includes("escapes root")) {
    return "path_escapes_root";
  }
  if (
    output.includes("OPENAI_API_KEY") ||
    output.includes("AI_GATEWAY_API_KEY") ||
    output.includes("ANTHROPIC_API_KEY") ||
    output.includes("GEMINI_API_KEY") ||
    output.includes("GROQ_API_KEY") ||
    output.includes("MISTRAL_API_KEY")
  ) {
    return "missing_openai_api_key";
  }
  if (output.includes("planning tasks")) {
    return "planning_failed";
  }
  if (output.includes("no extension")) {
    return "missing_file_extension";
  }
  if (output.includes("translation file parser")) {
    return "parser_failed";
  }
  return "unknown";
}

function formatDetectionLabel(input: {
  fileFormat?: string | null;
  sourceExtension?: string | null;
  sandboxInputExtension?: string | null;
}): string | null {
  const fileFormat = input.fileFormat?.trim();
  if (fileFormat) {
    return fileFormat;
  }
  const extension = input.sandboxInputExtension?.trim() || input.sourceExtension?.trim();
  if (extension) {
    return extension.startsWith(".") ? extension.slice(1) : extension;
  }
  return null;
}

function cliFailureKindFromMessage(message: string): string | null {
  const match = /kind=([a-z0-9_]+)/i.exec(message);
  return match?.[1] ?? null;
}

function resolveSupportedFormat(detection?: {
  fileFormat?: string | null;
  sourceExtension?: string | null;
  sandboxInputExtension?: string | null;
}): SupportedTranslationFileFormat | null {
  const fileFormat = detection?.fileFormat?.trim();
  if (
    fileFormat &&
    isSupportedFileTranslationFileFormat(fileFormat as SupportedTranslationFileFormat)
  ) {
    return fileFormat as SupportedTranslationFileFormat;
  }

  const extension = detection?.sandboxInputExtension?.trim() || detection?.sourceExtension?.trim();
  if (!extension) {
    return null;
  }
  return inferSupportedFileTranslationFileFormat(
    `file${extension.startsWith(".") ? extension : `.${extension}`}`,
  );
}

function userFacingTranslationFailureReason(
  message: string,
  detection?: {
    fileFormat?: string | null;
    sourceExtension?: string | null;
    sandboxInputExtension?: string | null;
  },
): string {
  const kind = cliFailureKindFromMessage(message);
  const supportedFormat = resolveSupportedFormat(detection);
  const detected = detection ? formatDetectionLabel(detection) : null;
  const label = detected || supportedFormat;

  if (kind === "markdown_ast_parity_mismatch" || kind === "markdown_parity_retry_exhausted") {
    return "markdown translation finished but the output structure no longer matched the source. Try again, or simplify complex markdown in the source file.";
  }
  if (kind === "placeholder_parity_mismatch") {
    return "the translation changed placeholders or markup that must stay identical to the source.";
  }
  if (kind === "missing_openai_api_key") {
    return "something went wrong while setting up the translation environment on our end.";
  }
  if (kind === "parser_failed" || kind === "missing_file_extension") {
    if (label && !supportedFormat) {
      return `the detected file format (${label}) is not supported for file translation.`;
    }
    return label
      ? `the ${label} file couldn't be parsed for translation.`
      : "the file couldn't be parsed for translation.";
  }

  if (supportedFormat && label) {
    return `translating the ${label} file failed. This is usually temporary — try again.`;
  }
  if (label) {
    return `the detected file format (${label}) may not be supported, or the content didn't match what the translator expected.`;
  }
  return "the file format may not be supported, or the content didn't match what the translator expected.";
}

function isSandboxDisconnectMessage(message: string): boolean {
  return (
    message.includes("Sandbox stream was closed and is not accepting commands") ||
    message.includes("sandbox_stream_closed") ||
    message.includes("stream_ended_early") ||
    message.includes("sandbox_disconnect") ||
    message === "terminated" ||
    message.includes("fetch failed") ||
    message.includes("ECONNRESET") ||
    message.includes("UND_ERR_")
  );
}

function userFacingFailureReason(
  error: unknown,
  detection?: {
    fileFormat?: string | null;
    sourceExtension?: string | null;
    sandboxInputExtension?: string | null;
  },
): string {
  const message = error instanceof Error ? error.message : "Unknown translation failure";

  if (message.startsWith("glossary validation failed")) {
    return message;
  }

  if (isSandboxDisconnectMessage(message)) {
    return "the translation environment disconnected mid-run. This is usually temporary — try again.";
  }

  if (
    message.includes("hyperlocalise CLI installation failed") ||
    message.includes("sandbox tool installation failed")
  ) {
    return "something went wrong while setting up the translation environment on our end.";
  }

  if (message.includes("failed to download attachment")) {
    return "the attachment couldn't be retrieved. It may have been too large or the link expired.";
  }

  if (message.includes("translation failed") || message.includes("failed to extract entries")) {
    return userFacingTranslationFailureReason(message, detection);
  }

  if (message.includes("failed to read translated file")) {
    return "the translation finished, but the output file couldn't be read back. This is usually temporary.";
  }

  if (message.includes("sandbox_timeout")) {
    return "the translation took too long to finish. Try the job again.";
  }

  return "the translation failed before it could finish. This is usually temporary.";
}

async function createSandboxStep() {
  "use step";
  const { createTranslationSandbox } = await import("@/lib/translation/sandbox");
  return createTranslationSandbox();
}

async function updateSandboxTimeoutStep(sandboxId: string, timeoutMs: number) {
  "use step";
  const { updateTranslationSandboxTimeout } = await import("@/lib/translation/sandbox");
  return updateTranslationSandboxTimeout(sandboxId, timeoutMs);
}

async function prepareSandboxStep(sandboxId: string) {
  "use step";
  const { prepareSandbox } = await import("@/lib/translation/sandbox");
  return prepareSandbox(sandboxId);
}

async function writeSourceFileStep(sandboxId: string, filename: string, content: Buffer) {
  "use step";
  const { writeFileToSandbox } = await import("@/lib/translation/sandbox");
  return writeFileToSandbox(sandboxId, filename, content);
}

async function recreateSandboxWithSourceStep(input: {
  previousSandboxId: string | null;
  filename: string;
  content: Buffer;
  timeoutMs: number;
}) {
  "use step";
  const { createTranslationSandbox, prepareSandbox, stopTranslationSandbox, writeFileToSandbox } =
    await import("@/lib/translation/sandbox");

  if (input.previousSandboxId) {
    try {
      await stopTranslationSandbox(input.previousSandboxId);
    } catch {
      // Best-effort cleanup of the dead sandbox.
    }
  }

  const { sandboxId } = await createTranslationSandbox(input.timeoutMs);
  await prepareSandbox(sandboxId);
  await writeFileToSandbox(sandboxId, input.filename, input.content);
  return { sandboxId };
}

async function runTranslationStep(
  sandboxId: string,
  inputFile: string,
  outputPattern: string,
  sourceLocale: string | null,
  targetLocales: string[],
  instructions: string | null,
  context: SandboxTranslationContext,
  prefilledByLocale: Record<string, Record<string, string>>,
  options?: {
    force?: boolean;
    maxTranslations?: number;
    organizationId?: string;
    projectId?: string;
    jobId?: string;
  },
) {
  "use step";

  const {
    buildMultiLocaleTempConfig,
    getSandboxTranslationEnv,
    isSandboxDisconnectError,
    recoverTranslationSandboxSession,
    runSandboxCommand,
    sandboxI18nConfigPath,
    sandboxTranslationCommandTimeoutMs,
    writeFileToSandbox,
    writeTempConfig,
  } = await import("@/lib/translation/sandbox");
  const { loadSandboxByokCredential } = await import("@/lib/translation/sandbox-byok");
  const byok = options?.organizationId
    ? await loadSandboxByokCredential(options.organizationId)
    : null;

  const { randomUUID } = await import("node:crypto");
  const invocationId = randomUUID();
  const reportPath = `/tmp/report-${invocationId}.json`;
  const config = buildMultiLocaleTempConfig(
    inputFile,
    outputPattern,
    sourceLocale,
    targetLocales,
    instructions,
    context,
    byok,
  );
  await writeTempConfig(sandboxId, config, sandboxI18nConfigPath);

  const localeFlags =
    targetLocales.length > 0
      ? targetLocales.map((locale) => `--locale '${shellSingleQuote(locale)}'`).join(" ")
      : "";

  let prefilledFlags = "";
  const localesWithPrefill = Object.entries(prefilledByLocale).filter(
    ([, entries]) => Object.keys(entries).length > 0,
  );
  if (localesWithPrefill.length > 0) {
    const nested: Record<string, Record<string, string>> = {};
    for (const [locale, entries] of localesWithPrefill) {
      nested[locale] = entries;
    }
    const prefilledPath = "/tmp/prefilled-by-locale.json";
    await writeFileToSandbox(sandboxId, prefilledPath, Buffer.from(JSON.stringify(nested), "utf8"));
    prefilledFlags = ` --prefilled-entries '${shellSingleQuote(prefilledPath)}'`;
  }

  const localeArg = localeFlags ? ` ${localeFlags}` : "";
  // First runs use --force for a clean slate. Same-sandbox retries omit it so
  // `.hyperlocalise.lock.json` can skip completed tasks / resume checkpoints.
  const forceFlag = options?.force === false ? "" : " --force";
  const maxTranslations = options?.maxTranslations;
  const maxTranslationsFlag =
    typeof maxTranslations === "number" && maxTranslations > 0
      ? ` --max-translations ${maxTranslations}`
      : "";
  try {
    const result = await runSandboxCommand(
      sandboxId,
      "bash",
      [
        "-lc",
        `hl run --config '${shellSingleQuote(sandboxI18nConfigPath)}'${localeArg}${forceFlag}${maxTranslationsFlag} --progress off --output '${reportPath}'${prefilledFlags}`,
      ],
      {
        env: getSandboxTranslationEnv(byok),
        timeoutMs: sandboxTranslationCommandTimeoutMs,
      },
    );
    if (options?.organizationId && options.projectId && options.jobId) {
      const { readTranslatedFile } = await import("@/lib/translation/sandbox");
      const { captureSandboxUsage } = await import("@/lib/reporting/sandbox-usage");
      const { resolveSandboxLlmProfile } = await import("@/lib/translation/sandbox-llm");
      const { env } = await import("@/lib/env");
      let report: string | null = null;
      try {
        report = (await readTranslatedFile(sandboxId, reportPath)).toString("utf8");
      } catch {
        /* Missing usage is recorded as unpriced. */
      }
      await captureSandboxUsage({
        organizationId: options.organizationId,
        projectId: options.projectId,
        jobId: options.jobId,
        invocationId,
        ...resolveSandboxLlmProfile(env, byok),
        report,
      });
    }
    return result;
  } catch (error) {
    // Surface a stable marker so the workflow can recreate the sandbox when
    // session recovery inside runSandboxCommand is not enough.
    if (isSandboxDisconnectError(error)) {
      try {
        await recoverTranslationSandboxSession(sandboxId);
      } catch {
        // Ignore — workflow will recreate.
      }
      throw new Error(
        `sandbox_disconnect: Sandbox stream was closed and is not accepting commands.`,
      );
    }
    throw error;
  }
}
runTranslationStep.maxRetries = 0;

async function extractEntriesStep(
  sandboxId: string,
  path: string,
  options?: { sourcePath?: string },
) {
  "use step";
  // Use extractSandboxEntries so UTF-8 entries are read via binary file IO,
  // not sandbox stdout string capture (which can turn multi-byte chars into �).
  const { extractSandboxEntries } = await import("@/lib/translation/sandbox");
  const result = await extractSandboxEntries(sandboxId, path, {
    sourcePath: options?.sourcePath,
  });
  if (!result.ok) {
    throw new Error(
      `failed to extract entries: exitCode=${result.exitCode} kind=${classifyCliFailureKind(result.output)}`,
    );
  }
  return hlEntriesPayloadToStringMap(result.entries);
}
async function readOutputStep(sandboxId: string, outputFile: string, _attempt: 1 | 2) {
  "use step";
  const { readTranslatedFile } = await import("@/lib/translation/sandbox");
  return readTranslatedFile(sandboxId, outputFile);
}

async function stopSandboxStep(sandboxId: string) {
  "use step";
  const { stopTranslationSandbox } = await import("@/lib/translation/sandbox");
  return stopTranslationSandbox(sandboxId);
}

async function logDiagnosticsStep(
  jobId: string,
  sourceFilename: string,
  targetLocale: string,
  content: Buffer,
  outputFilename: string,
) {
  "use step";

  const { logTranslatedFileDiagnostics } = await import("@/lib/translation/diagnostics");
  return logTranslatedFileDiagnostics(
    jobId,
    "file-translation",
    sourceFilename,
    targetLocale,
    content,
    outputFilename,
  );
}

async function assembleFileTranslationContextStep(input: {
  jobId: string;
  projectId: string;
  sourceLocale: string;
  targetLocales: string[];
  sourceContent: Buffer;
  metadata?: Record<string, string>;
}) {
  "use step";

  const { and, eq } = await import("drizzle-orm");
  const { db, schema } = await import("@/lib/database/client");
  const { listGlossaryTermsForProject, FILE_TRANSLATION_GLOSSARY_PAIR_LIMIT } =
    await import("@/lib/glossary/query-glossary-terms");
  const { sourceContainsTerm } =
    await import("@/lib/glossary/validate-glossary-terms-in-translation");

  const [project] = await db
    .select({
      id: schema.projects.id,
      name: schema.projects.name,
      organizationId: schema.projects.organizationId,
      translationContext: schema.projects.translationContext,
    })
    .from(schema.projects)
    .where(eq(schema.projects.id, input.projectId))
    .limit(1);

  if (!project) {
    throw new Error(`project ${input.projectId} not found`);
  }

  const sourceText = input.sourceContent.toString("utf8").slice(0, 500_000);
  const attachedTerms = await listGlossaryTermsForProject({
    organizationId: project.organizationId,
    projectId: input.projectId,
    sourceLocale: input.sourceLocale,
    targetLocales: input.targetLocales,
    maxPairs: FILE_TRANSLATION_GLOSSARY_PAIR_LIMIT,
  });

  const glossaryTerms = attachedTerms
    .filter((term) => sourceContainsTerm(sourceText, term))
    .slice(0, 50)
    .map(({ sourceTerm, targetTerm, targetLocale, description, forbidden, caseSensitive }) => ({
      sourceTerm,
      targetTerm,
      targetLocale,
      description,
      forbidden,
      caseSensitive,
    }));

  const context = {
    projectName: project.name,
    projectTranslationContext: project.translationContext,
    jobContext: input.metadata?.context ?? null,
    glossaryTerms,
  } satisfies SandboxTranslationContext;

  await db
    .update(schema.jobs)
    .set({
      contextSnapshot: {
        assembledAt: new Date().toISOString(),
        project,
        job: {
          sourceLocale: input.sourceLocale,
          targetLocales: input.targetLocales,
          metadata: input.metadata,
        },
        glossaryTerms,
      },
    })
    .where(and(eq(schema.jobs.id, input.jobId), eq(schema.jobs.projectId, input.projectId)));

  return context;
}

export async function fileTranslationJobWorkflow(event: TranslationJobEventData) {
  "use workflow";

  const { workflowRunId } = getWorkflowMetadata();
  const claim = await claimTranslationJobStep({ event, runId: workflowRunId });

  if (claim.kind === "skipped") {
    return claim.job;
  }

  if (claim.job.type !== "file") {
    throw new Error(`fileTranslationJobWorkflow received non-file job: ${claim.job.type}`);
  }

  const parsedInput =
    event.type === "file"
      ? (claim.job.inputPayload as {
          sourceFileId: string;
          fileFormat: string;
          sourceLocale: string;
          targetLocales: string[];
          metadata?: Record<string, string>;
        })
      : null;

  if (!parsedInput) {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "invalid_file_job_input",
      message: "missing or invalid file translation job input",
    });
    throw new Error("invalid file job input");
  }

  let organizationId: string;
  try {
    organizationId = await getProjectOrganizationStep(claim.job.projectId);
  } catch {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "translation_project_not_found",
      message: `project ${claim.job.projectId} not found`,
    });
    throw new Error("project not found");
  }

  const aiFeatures = await ensureAiFeaturesAllowedStep({ organizationId });
  if (!aiFeatures.ok) {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: aiFeatures.error.code,
      message: aiFeatures.error.message,
    });
    return { status: "failed" as const, reason: aiFeatures.error.code };
  }

  if (isOfficeTranslationFileFormat(parsedInput.fileFormat as SupportedTranslationFileFormat)) {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "office_file_manual_cat",
      message:
        "Office files are localized in CAT File view. Upload or edit the translated file there.",
    });
    return { status: "failed" as const, reason: "office_file_manual_cat" };
  }

  if (isImageTranslationFileFormat(parsedInput.fileFormat as SupportedTranslationFileFormat)) {
    let sourceFile: Awaited<ReturnType<typeof getStoredFileStep>>;
    try {
      sourceFile = await getStoredFileStep(parsedInput.sourceFileId, organizationId);
    } catch {
      await failTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        code: "source_file_not_found",
        message: `source file ${parsedInput.sourceFileId} not found`,
      });
      throw new Error("source file not found");
    }

    const repositorySourcePath =
      (await getRepositorySourcePathForStoredFileStep(parsedInput.sourceFileId, organizationId)) ??
      sourceFile.filename;

    const outputFiles: Array<{ fileId: string; locale: string; filename: string }> = [];
    try {
      for (const targetLocale of parsedInput.targetLocales) {
        const output = await localizeImageVariantForJobStep({
          organizationId,
          projectId: claim.job.projectId,
          sourcePath: repositorySourcePath,
          targetLocale,
          sourceLocale: parsedInput.sourceLocale,
          sourceStoredFileId: parsedInput.sourceFileId,
          sourceJobId: claim.job.id,
        });
        outputFiles.push(output);
      }

      await completeFileTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        outputFiles,
      });

      return outputFiles;
    } catch (error) {
      const message = error instanceof Error ? error.message : "image translation failed";
      await failTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        code: "image_translation_failed",
        message,
      });
      throw error;
    }
  }

  if (isVideoTranslationFileFormat(parsedInput.fileFormat as SupportedTranslationFileFormat)) {
    let sourceFile: Awaited<ReturnType<typeof getStoredFileStep>>;
    try {
      sourceFile = await getStoredFileStep(parsedInput.sourceFileId, organizationId);
    } catch {
      await failTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        code: "source_file_not_found",
        message: `source file ${parsedInput.sourceFileId} not found`,
      });
      throw new Error("source file not found");
    }

    const repositorySourcePath =
      (await getRepositorySourcePathForStoredFileStep(parsedInput.sourceFileId, organizationId)) ??
      sourceFile.filename;

    const outputFiles: Array<{ fileId: string; locale: string; filename: string }> = [];
    try {
      for (const targetLocale of parsedInput.targetLocales) {
        const output = await localizeVideoVariantForJobStep({
          organizationId,
          projectId: claim.job.projectId,
          sourcePath: repositorySourcePath,
          targetLocale,
          sourceLocale: parsedInput.sourceLocale,
          sourceStoredFileId: parsedInput.sourceFileId,
          sourceJobId: claim.job.id,
        });
        outputFiles.push(output);
      }

      await completeFileTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        outputFiles,
      });

      return outputFiles;
    } catch (error) {
      const message = error instanceof Error ? error.message : "video translation failed";
      const code = message.includes("video_duration")
        ? message.includes("unreadable")
          ? "video_duration_unreadable"
          : "video_duration_unsupported"
        : message.includes("video_edit_region_blocked")
          ? "video_edit_region_blocked"
          : message.includes("video_model_unavailable")
            ? "video_model_unavailable"
            : "video_localization_failed";
      await failTranslationJobStep({
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        workflowRunId: claim.job.workflowRunId,
        code,
        message,
      });
      throw error;
    }
  }

  let sourceFile: Awaited<ReturnType<typeof getStoredFileStep>>;
  try {
    sourceFile = await getStoredFileStep(parsedInput.sourceFileId, organizationId);
  } catch {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "source_file_not_found",
      message: `source file ${parsedInput.sourceFileId} not found`,
    });
    throw new Error("source file not found");
  }

  const repositorySourcePath = await getRepositorySourcePathForStoredFileStep(
    parsedInput.sourceFileId,
    organizationId,
  );

  let sourceContent: Buffer;
  try {
    sourceContent = await getStoredFileContentStep(parsedInput.sourceFileId, organizationId);
  } catch (error) {
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "source_file_unavailable",
      message: `source file ${parsedInput.sourceFileId} could not be retrieved from storage`,
    });
    throw error;
  }

  let { sandboxId } = await createSandboxStep();
  const inputFilename = getSandboxInputFilename(sourceFile.filename);
  const instructions = parsedInput.metadata?.instructions ?? null;
  const context = await assembleFileTranslationContextStep({
    jobId: claim.job.id,
    projectId: claim.job.projectId,
    sourceLocale: parsedInput.sourceLocale,
    targetLocales: parsedInput.targetLocales,
    sourceContent,
    metadata: parsedInput.metadata,
  });

  try {
    await prepareSandboxStep(sandboxId);
    await writeSourceFileStep(sandboxId, inputFilename, sourceContent);

    console.info("[file-translation-workflow] source file written to sandbox", {
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      storedFileId: parsedInput.sourceFileId,
      fileFormat: parsedInput.fileFormat,
      sourceExtension: fileExtension(sourceFile.filename),
      sandboxInputExtension: fileExtension(inputFilename),
      byteLength: sourceContent.byteLength,
      hasRepositorySourcePath: Boolean(repositorySourcePath),
      targetLocaleCount: parsedInput.targetLocales.length,
      sandboxId,
    });

    const outputFiles: Array<{ fileId: string; locale: string; filename: string }> = [];
    let sourceEntries: Record<string, string> | null = null;
    let translationSandboxTimeoutMs = calculateFileTranslationSandboxTimeoutMs(0);
    let translationMaxPages = calculateFileTranslationMaxPages(0);

    try {
      sourceEntries = await extractEntriesStep(sandboxId, inputFilename);
      console.info("[file-translation-workflow] source entries extracted", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        storedFileId: parsedInput.sourceFileId,
        fileFormat: parsedInput.fileFormat,
        sourceEntryCount: Object.keys(sourceEntries).length,
      });
    } catch (error) {
      console.warn("[file-translation-workflow] source TM extraction failed", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        storedFileId: parsedInput.sourceFileId,
        fileFormat: parsedInput.fileFormat,
        hasRepositorySourcePath: Boolean(repositorySourcePath),
        userFacingError: userFacingFailureReason(error, {
          fileFormat: parsedInput.fileFormat,
          sourceExtension: fileExtension(sourceFile.filename),
          sandboxInputExtension: fileExtension(inputFilename),
        }),
      });
    }

    const sourceText = sourceContent.toString("utf8");
    const outputPattern = getSandboxOutputFilenamePattern(sourceFile.filename);
    const prefilledByLocale: Record<string, Record<string, string>> = {};

    for (const targetLocale of parsedInput.targetLocales) {
      let tmPrefilled: Record<string, string> = {};
      if (sourceEntries) {
        await captureFileAnalysisStep({
          organizationId,
          projectId: claim.job.projectId,
          jobId: claim.job.id,
          sourceLocale: parsedInput.sourceLocale,
          targetLocale,
          sourceEntries,
        });
        tmPrefilled = await reuseFileTranslationMemoryEntriesStep({
          projectId: claim.job.projectId,
          sourceLocale: parsedInput.sourceLocale,
          targetLocale,
          sourceEntries,
        });
        if (Object.keys(tmPrefilled).length > 0) {
          console.info("[file-translation-workflow] matched reusable translation memory entries", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            reusedEntryCount: Object.keys(tmPrefilled).length,
            sourceEntryCount: Object.keys(sourceEntries).length,
          });
        }
      }

      let existingPrefilled: Record<string, string> = {};
      let retryKeys: string[] = [];
      if (repositorySourcePath) {
        const projectPrefill = await loadProjectTranslationsAsPrefilledEntriesStep({
          organizationId,
          projectId: claim.job.projectId,
          sourcePath: repositorySourcePath,
          targetLocale,
        });
        existingPrefilled = projectPrefill.prefilled;
        retryKeys = projectPrefill.retryKeys;
        if (projectPrefill.truncated) {
          console.warn("[file-translation-workflow] project translation prefill truncated", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            loadedKeyCount: projectPrefill.loadedKeyCount,
            maxKeyCount: projectPrefill.maxKeyCount,
          });
        }
        if (Object.keys(existingPrefilled).length > 0) {
          console.info("[file-translation-workflow] loaded existing project translations", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            prefilledEntryCount: Object.keys(existingPrefilled).length,
          });
        }
        if (retryKeys.length > 0) {
          console.info("[file-translation-workflow] omitted same-as-source review prefills", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            omittedKeyCount: retryKeys.length,
          });
        }
      }

      const merged = mergeTranslationPrefills({
        tmPrefilled,
        projectPrefilled: existingPrefilled,
        retryKeys,
      });
      if (Object.keys(merged).length > 0) {
        prefilledByLocale[targetLocale] = merged;
      }
    }

    if (sourceEntries) {
      const pendingTranslationCount = countPendingFileTranslations(
        sourceEntries,
        parsedInput.targetLocales,
        prefilledByLocale,
      );
      translationSandboxTimeoutMs =
        calculateFileTranslationSandboxTimeoutMs(pendingTranslationCount);
      translationMaxPages = calculateFileTranslationMaxPages(pendingTranslationCount);
      await updateSandboxTimeoutStep(sandboxId, translationSandboxTimeoutMs);
      console.info("[file-translation-workflow] sandbox timeout updated", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        sourceEntryCount: Object.keys(sourceEntries).length,
        targetLocaleCount: parsedInput.targetLocales.length,
        pendingTranslationCount,
        translationMaxPages,
        translationSandboxTimeoutMs,
        sandboxId,
      });
    }

    const prefilledLocaleCount = Object.keys(prefilledByLocale).length;
    const prefilledEntryCount = Object.values(prefilledByLocale).reduce(
      (sum, entries) => sum + Object.keys(entries).length,
      0,
    );

    console.info("[file-translation-workflow] starting hl run", {
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      storedFileId: parsedInput.sourceFileId,
      fileFormat: parsedInput.fileFormat,
      targetLocales: parsedInput.targetLocales,
      sandboxInputExtension: fileExtension(inputFilename),
      sourceEntryCount: sourceEntries ? Object.keys(sourceEntries).length : null,
      prefilledLocaleCount,
      prefilledEntryCount,
      glossaryTermCount: context.glossaryTerms?.length ?? 0,
      sandboxId,
    });

    type LocaleGlossaryFailure = {
      targetLocale: string;
      failures: ReturnType<typeof validateGlossaryTermsInTranslation>;
    };

    const translatedByLocale = new Map<string, Buffer>();
    const runFailures: Array<{ locale: string; kind: string; exitCode: number }> = [];

    const persistReadableLocales = async (locales: string[], attempt: 1 | 2) => {
      if (!sourceEntries) {
        return;
      }
      for (const targetLocale of locales) {
        const outputFilename = getSandboxOutputFilename(sourceFile.filename, targetLocale);
        try {
          const targetEntries = await extractEntriesStep(sandboxId, outputFilename, {
            sourcePath: inputFilename,
          });
          await captureFileCompletionsStep({
            organizationId,
            jobId: claim.job.id,
            targetLocale,
            sourceEntries,
            targetEntries,
          });
          if (!repositorySourcePath) continue;
          await persistFileTranslationMemoryEntriesStep({
            projectId: claim.job.projectId,
            jobId: claim.job.id,
            sourceLocale: parsedInput.sourceLocale,
            targetLocale,
            sourcePath: repositorySourcePath,
            sourceFileHash: sourceFile.sha256,
            sourceEntries,
            targetEntries,
          });
          if (
            !isDocumentTranslationFileFormat(
              parsedInput.fileFormat as SupportedTranslationFileFormat,
            )
          ) {
            await persistFileProjectTranslationsStep({
              organizationId,
              projectId: claim.job.projectId,
              jobId: claim.job.id,
              sourcePath: repositorySourcePath,
              sourceLocale: parsedInput.sourceLocale,
              targetLocale,
              sourceEntries,
              targetEntries,
            });
          }
        } catch (error) {
          console.warn("[file-translation-workflow] incremental translation persistence failed", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            attempt,
            userFacingError: userFacingFailureReason(error, {
              fileFormat: parsedInput.fileFormat,
              sourceExtension: fileExtension(sourceFile.filename),
              sandboxInputExtension: fileExtension(inputFilename),
            }),
          });
        }
      }
    };

    const runHlForLocales = async (
      locales: string[],
      attempt: 1 | 2,
      options?: { retryFeedback?: string; force?: boolean; maxTranslations?: number },
    ) => {
      const force = options?.force ?? true;
      const retryFeedback = options?.retryFeedback;
      const maxTranslations =
        options?.maxTranslations ?? FILE_TRANSLATION_MAX_TRANSLATIONS_PER_SESSION;
      const localeSet = new Set(locales);
      const runContext =
        context.glossaryTerms != null
          ? {
              ...context,
              glossaryTerms: context.glossaryTerms.filter((term) =>
                localeSet.has(term.targetLocale),
              ),
            }
          : context;

      const runOnce = async (runForce: boolean) =>
        runTranslationStep(
          sandboxId,
          inputFilename,
          outputPattern,
          parsedInput.sourceLocale,
          locales,
          retryFeedback ? [instructions, retryFeedback].filter(Boolean).join("\n\n") : instructions,
          runContext,
          Object.fromEntries(
            locales
              .filter((locale) => prefilledByLocale[locale])
              .map((locale) => [locale, prefilledByLocale[locale]]),
          ),
          {
            force: runForce,
            maxTranslations,
            organizationId,
            projectId: claim.job.projectId,
            jobId: claim.job.id,
          },
        );

      let translation: Awaited<ReturnType<typeof runTranslationStep>>;
      try {
        translation = await runOnce(force);
      } catch (error) {
        const message = error instanceof Error ? error.message : "";
        if (!isSandboxDisconnectMessage(message)) {
          throw error;
        }

        // Prefer same-sandbox resume without --force so the lockfile can skip
        // completed tasks. Only recreate when that still cannot accept commands.
        console.warn("[file-translation-workflow] sandbox disconnect; retrying without force", {
          jobId: claim.job.id,
          projectId: claim.job.projectId,
          sandboxId,
          targetLocales: locales,
          attempt,
          error: message,
        });
        try {
          translation = await runOnce(false);
        } catch (retryError) {
          const retryMessage = retryError instanceof Error ? retryError.message : "";
          if (!isSandboxDisconnectMessage(retryMessage)) {
            throw retryError;
          }

          console.warn("[file-translation-workflow] sandbox disconnect; recreating sandbox", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            previousSandboxId: sandboxId,
            targetLocales: locales,
            attempt,
            error: retryMessage,
          });
          const recreated = await recreateSandboxWithSourceStep({
            previousSandboxId: sandboxId,
            filename: inputFilename,
            content: sourceContent,
            timeoutMs: translationSandboxTimeoutMs,
          });
          sandboxId = recreated.sandboxId;
          // Fresh sandbox has no lockfile — force is irrelevant; keep caller's intent.
          translation = await runOnce(force);
        }
      }

      const deferredByLimit = parseDeferredByLimit(translation.output);
      if (translation.exitCode !== 0) {
        const cliFailureKind = classifyCliFailureKind(translation.output);
        console.error("[file-translation-workflow] hl run failed", {
          jobId: claim.job.id,
          projectId: claim.job.projectId,
          storedFileId: parsedInput.sourceFileId,
          fileFormat: parsedInput.fileFormat,
          targetLocales: locales,
          attempt,
          force,
          maxTranslations,
          deferredByLimit,
          sandboxInputExtension: fileExtension(inputFilename),
          exitCode: translation.exitCode,
          cliOutputLength: translation.output.length,
          cliFailureKind,
          prefilledEntryCount,
          hasRetryFeedback: Boolean(retryFeedback),
          sandboxId,
        });
        return {
          ok: false as const,
          cliFailureKind,
          exitCode: translation.exitCode,
          deferredByLimit,
        };
      }

      console.info("[file-translation-workflow] hl run succeeded", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        fileFormat: parsedInput.fileFormat,
        targetLocales: locales,
        attempt,
        force,
        maxTranslations,
        deferredByLimit,
        exitCode: translation.exitCode,
      });
      return { ok: true as const, deferredByLimit };
    };

    const tryReadLocaleOutputs = async (locales: string[], attempt: 1 | 2) => {
      const readable: string[] = [];
      const missing: string[] = [];
      for (const targetLocale of locales) {
        const outputFilename = getSandboxOutputFilename(sourceFile.filename, targetLocale);
        try {
          const translatedContent = await readOutputStep(sandboxId, outputFilename, attempt);
          translatedByLocale.set(targetLocale, translatedContent);
          readable.push(targetLocale);
        } catch {
          missing.push(targetLocale);
        }
      }
      return { readable, missing };
    };

    // Paginate hl run so large files stay under sandbox/workflow timeouts and
    // project translations populate after each page.
    let deferredByLimit = 0;
    let page = 0;
    let localesNeedingWork = [...parsedInput.targetLocales];
    let batchFailed = false;

    while (page < translationMaxPages) {
      // Page 0 may use --force for a clean slate. Later pages omit it so the
      // lockfile skips completed tasks and advances through deferred work.
      const batchResult = await runHlForLocales(parsedInput.targetLocales, 1, {
        force: page === 0,
        maxTranslations: FILE_TRANSLATION_MAX_TRANSLATIONS_PER_SESSION,
      });
      deferredByLimit = batchResult.deferredByLimit;

      if (batchResult.ok) {
        const { readable, missing } = await tryReadLocaleOutputs(parsedInput.targetLocales, 1);
        localesNeedingWork = missing;
        if (readable.length > 0) {
          await persistReadableLocales(readable, 1);
        }
        console.info("[file-translation-workflow] hl run page completed", {
          jobId: claim.job.id,
          projectId: claim.job.projectId,
          page,
          deferredByLimit,
          readableLocaleCount: readable.length,
          missingLocaleCount: missing.length,
        });
        if (deferredByLimit <= 0) {
          break;
        }
        page += 1;
        continue;
      }

      const { readable, missing } = await tryReadLocaleOutputs(parsedInput.targetLocales, 1);
      console.warn("[file-translation-workflow] batch hl run failed; salvaging readable outputs", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        page,
        readableLocales: readable,
        missingLocales: missing,
        cliFailureKind: batchResult.cliFailureKind,
        exitCode: batchResult.exitCode,
        deferredByLimit,
      });
      if (readable.length > 0) {
        await persistReadableLocales(readable, 1);
      }
      localesNeedingWork = missing;
      for (const locale of missing) {
        runFailures.push({
          locale,
          kind: batchResult.cliFailureKind,
          exitCode: batchResult.exitCode,
        });
      }
      batchFailed = true;
      break;
    }

    if (!batchFailed && deferredByLimit > 0) {
      throw new Error(
        `translation pagination exceeded ${translationMaxPages} pages with deferred_by_limit=${deferredByLimit}`,
      );
    }

    // Retry missing locales individually so one bad locale cannot block the rest.
    // Page 0 uses --force; later pages omit it so deferred work advances.
    if (localesNeedingWork.length > 0) {
      const stillMissing: string[] = [];
      for (const targetLocale of localesNeedingWork) {
        let localeFailed = false;
        for (let localePage = 0; localePage < translationMaxPages; localePage += 1) {
          const localeResult = await runHlForLocales([targetLocale], 1, {
            force: localePage === 0,
            maxTranslations: FILE_TRANSLATION_MAX_TRANSLATIONS_PER_SESSION,
          });
          if (!localeResult.ok) {
            stillMissing.push(targetLocale);
            const idx = runFailures.findIndex((f) => f.locale === targetLocale);
            const failure = {
              locale: targetLocale,
              kind: localeResult.cliFailureKind,
              exitCode: localeResult.exitCode,
            };
            if (idx >= 0) {
              runFailures[idx] = failure;
            } else {
              runFailures.push(failure);
            }
            localeFailed = true;
            break;
          }

          const { readable, missing } = await tryReadLocaleOutputs([targetLocale], 1);
          if (missing.length > 0) {
            stillMissing.push(targetLocale);
            runFailures.push({
              locale: targetLocale,
              kind: "missing_output",
              exitCode: 0,
            });
            localeFailed = true;
            break;
          }
          if (readable.length > 0) {
            await persistReadableLocales(readable, 1);
          }
          if (localeResult.deferredByLimit <= 0) {
            break;
          }
          if (localePage === translationMaxPages - 1) {
            stillMissing.push(targetLocale);
            runFailures.push({
              locale: targetLocale,
              kind: "pagination_exhausted",
              exitCode: 0,
            });
            localeFailed = true;
          }
        }
        if (!localeFailed) {
          const idx = runFailures.findIndex((f) => f.locale === targetLocale);
          if (idx >= 0) {
            runFailures.splice(idx, 1);
          }
        }
      }
      localesNeedingWork = stillMissing;
    }

    const glossaryFailuresByLocale: LocaleGlossaryFailure[] = [];
    for (const targetLocale of parsedInput.targetLocales) {
      const translatedContent = translatedByLocale.get(targetLocale);
      if (!translatedContent) {
        continue;
      }

      const localeTerms = (context.glossaryTerms ?? []).filter(
        (term) => term.targetLocale === targetLocale,
      );
      const glossaryFailures = validateGlossaryTermsInTranslation({
        sourceText,
        translatedText: translatedContent.toString("utf8"),
        terms: localeTerms.map((term) => ({
          sourceTerm: term.sourceTerm,
          targetTerm: term.targetTerm,
          targetLocale: term.targetLocale,
          forbidden: term.forbidden,
          caseSensitive: term.caseSensitive,
        })),
      });
      if (glossaryFailures.length > 0) {
        glossaryFailuresByLocale.push({ targetLocale, failures: glossaryFailures });
      }
    }

    if (glossaryFailuresByLocale.length > 0) {
      console.warn("[file-translation-workflow] glossary validation failed; retrying", {
        jobId: claim.job.id,
        projectId: claim.job.projectId,
        failedLocales: glossaryFailuresByLocale.map((item) => item.targetLocale),
        failedTermCount: glossaryFailuresByLocale.reduce(
          (sum, item) => sum + item.failures.length,
          0,
        ),
        attempt: 1,
      });

      // Retry each failed locale individually with locale-specific feedback so
      // one locale's glossary constraints cannot contaminate another.
      // Page through --max-translations so a successful retry cannot leave
      // deferred keys untranslated.
      const stillFailing: LocaleGlossaryFailure[] = [];
      for (const { targetLocale, failures } of glossaryFailuresByLocale) {
        const feedback = [
          `Glossary validation failed for locale ${targetLocale}. Fix these term constraints exactly and regenerate:`,
          ...failures.map((failure) =>
            failure.forbidden
              ? `- Forbidden term violation for source "${failure.sourceTerm}": do not use "${failure.targetTerm}"`
              : `- Missing preferred term for source "${failure.sourceTerm}": must include "${failure.targetTerm}"`,
          ),
        ].join("\n");

        let retryFailed = false;
        for (let retryPage = 0; retryPage < translationMaxPages; retryPage += 1) {
          const retryResult = await runHlForLocales([targetLocale], 2, {
            retryFeedback: feedback,
            // Page 0 forces a clean rewrite; later pages omit --force so the
            // lockfile advances through deferred work.
            force: retryPage === 0,
            maxTranslations: FILE_TRANSLATION_MAX_TRANSLATIONS_PER_SESSION,
          });
          if (!retryResult.ok) {
            translatedByLocale.delete(targetLocale);
            stillFailing.push({ targetLocale, failures });
            retryFailed = true;
            break;
          }

          const { readable, missing } = await tryReadLocaleOutputs([targetLocale], 2);
          if (missing.length > 0) {
            translatedByLocale.delete(targetLocale);
            stillFailing.push({ targetLocale, failures });
            retryFailed = true;
            break;
          }
          if (readable.length > 0) {
            await persistReadableLocales(readable, 2);
          }
          if (retryResult.deferredByLimit <= 0) {
            break;
          }
          if (retryPage === translationMaxPages - 1) {
            translatedByLocale.delete(targetLocale);
            stillFailing.push({ targetLocale, failures });
            retryFailed = true;
          }
        }
        if (retryFailed) {
          continue;
        }

        const translatedContent = translatedByLocale.get(targetLocale);
        if (!translatedContent) {
          stillFailing.push({ targetLocale, failures });
          continue;
        }

        const localeTerms = (context.glossaryTerms ?? []).filter(
          (term) => term.targetLocale === targetLocale,
        );
        const glossaryFailures = validateGlossaryTermsInTranslation({
          sourceText,
          translatedText: translatedContent.toString("utf8"),
          terms: localeTerms.map((term) => ({
            sourceTerm: term.sourceTerm,
            targetTerm: term.targetTerm,
            targetLocale: term.targetLocale,
            forbidden: term.forbidden,
            caseSensitive: term.caseSensitive,
          })),
        });
        if (glossaryFailures.length > 0) {
          translatedByLocale.delete(targetLocale);
          stillFailing.push({ targetLocale, failures: glossaryFailures });
        }
      }

      if (stillFailing.length > 0) {
        runFailures.push(
          ...stillFailing.map(({ targetLocale }) => ({
            locale: targetLocale,
            kind: "glossary_validation",
            exitCode: 0,
          })),
        );
      }
    }

    // Persist every locale we successfully translated. If any locale is still
    // missing after salvage/retry, persist the good ones then fail the job.
    const missingLocales: string[] = [];
    for (const targetLocale of parsedInput.targetLocales) {
      const outputFilename = getSandboxOutputFilename(sourceFile.filename, targetLocale);
      const translatedContent = translatedByLocale.get(targetLocale);
      if (!translatedContent) {
        missingLocales.push(targetLocale);
        continue;
      }

      await logDiagnosticsStep(
        claim.job.id,
        sourceFile.filename,
        targetLocale,
        translatedContent,
        outputFilename,
      );

      const storedOutput = await storeOutputFileStep({
        organizationId,
        projectId: claim.job.projectId,
        jobId: claim.job.id,
        filename: outputFilename,
        contentType: sourceFile.contentType,
        content: translatedContent,
      });

      if (
        isDocumentTranslationFileFormat(parsedInput.fileFormat as SupportedTranslationFileFormat) &&
        repositorySourcePath
      ) {
        await persistDocumentVariantBytesStep({
          organizationId,
          projectId: claim.job.projectId,
          sourcePath: repositorySourcePath,
          targetLocale,
          content: translatedContent,
          contentType: sourceFile.contentType,
          filename: outputFilename,
          sourceJobId: claim.job.id,
        });
      }

      if (sourceEntries) {
        const targetEntries = await extractEntriesStep(sandboxId, outputFilename, {
          sourcePath: inputFilename,
        });
        await captureFileCompletionsStep({
          organizationId,
          jobId: claim.job.id,
          targetLocale,
          sourceEntries,
          targetEntries,
        });
      }

      if (sourceEntries && repositorySourcePath) {
        try {
          const targetEntries = await extractEntriesStep(sandboxId, outputFilename, {
            sourcePath: inputFilename,
          });
          await persistFileTranslationMemoryEntriesStep({
            projectId: claim.job.projectId,
            jobId: claim.job.id,
            sourceLocale: parsedInput.sourceLocale,
            targetLocale,
            sourcePath: repositorySourcePath,
            sourceFileHash: sourceFile.sha256,
            sourceEntries,
            targetEntries,
          });
          if (
            !isDocumentTranslationFileFormat(
              parsedInput.fileFormat as SupportedTranslationFileFormat,
            )
          ) {
            await persistFileProjectTranslationsStep({
              organizationId,
              projectId: claim.job.projectId,
              jobId: claim.job.id,
              sourcePath: repositorySourcePath,
              sourceLocale: parsedInput.sourceLocale,
              targetLocale,
              sourceEntries,
              targetEntries,
            });
          }
        } catch (error) {
          console.warn("[file-translation-workflow] target TM persistence failed", {
            jobId: claim.job.id,
            projectId: claim.job.projectId,
            targetLocale,
            userFacingError: userFacingFailureReason(error, {
              fileFormat: parsedInput.fileFormat,
              sourceExtension: fileExtension(sourceFile.filename),
              sandboxInputExtension: fileExtension(inputFilename),
            }),
          });
        }
      } else if (sourceEntries && !repositorySourcePath) {
        console.warn("[file-translation-workflow] skipped native translation persistence", {
          jobId: claim.job.id,
          projectId: claim.job.projectId,
          storedFileId: parsedInput.sourceFileId,
        });
      }

      outputFiles.push({
        fileId: storedOutput.id,
        locale: targetLocale,
        filename: outputFilename,
      });
    }

    if (missingLocales.length > 0) {
      const failedKinds = new Map(runFailures.map((f) => [f.locale, f.kind]));
      throw new Error(
        `translation failed for locales: ${missingLocales
          .map((locale) => `${locale}(${failedKinds.get(locale) ?? "missing_output"})`)
          .join(",")}`,
      );
    }

    await completeFileTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      outputFiles,
    });

    return outputFiles;
  } catch (error) {
    const reason = userFacingFailureReason(error, {
      fileFormat: parsedInput.fileFormat,
      sourceExtension: fileExtension(sourceFile.filename),
      sandboxInputExtension: fileExtension(inputFilename),
    });
    console.error("[file-translation-workflow] file translation failed", {
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      storedFileId: parsedInput.sourceFileId,
      fileFormat: parsedInput.fileFormat,
      sourceExtension: fileExtension(sourceFile.filename),
      sandboxInputExtension: fileExtension(inputFilename),
      byteLength: sourceContent.byteLength,
      hasRepositorySourcePath: Boolean(repositorySourcePath),
      targetLocales: parsedInput.targetLocales,
      sandboxId,
      error: reason,
    });
    await failTranslationJobStep({
      jobId: claim.job.id,
      projectId: claim.job.projectId,
      workflowRunId: claim.job.workflowRunId,
      code: "file_translation_failed",
      message: reason,
    });
    throw error;
  } finally {
    await stopSandboxStep(sandboxId);
  }
}

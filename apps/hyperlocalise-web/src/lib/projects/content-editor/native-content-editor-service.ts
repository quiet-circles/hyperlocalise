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
import { captureAnalysis, captureCompletions } from "@/lib/reporting/capture";
import { and, eq } from "drizzle-orm";

import type {
  ProjectFileContentEditorComment,
  ProjectFileContentEditorQueueFile,
  ProjectFileContentEditorSegment,
  ProjectFileContentEditorTranslation,
} from "@/api/routes/project/project.schema";
import { legacyNativeContentEditorSegmentLimit } from "@/api/routes/project/project.schema";
import { db, schema } from "@/lib/database/client";
import { getLatestRepositorySourceFileVersion } from "@/lib/file-storage/records";
import { NativeContentEditorCommentService } from "@/lib/projects/content-editor/native-content-editor-comment-service";
import {
  CAT_ALL_FILES_FILENAME,
  CONTENT_EDITOR_ALL_FILES_SOURCE_PATH,
  isContentEditorAllFilesSourcePath,
} from "@/lib/projects/content-editor-all-files";
import {
  buildCatFilePagination,
  type ProjectFileContentEditorPaginationInput,
} from "@/lib/projects/content-editor/project-file-content-editor-pagination";
import { getImageVariant, projectImageAssetPath } from "@/lib/projects/files/image-variant-service";
import {
  IMAGE_URL_CONTENT_KIND,
  isImageUrlContentKind,
} from "@/lib/projects/files/image-url-translation-service";
import { getVideoVariant, projectVideoAssetPath } from "@/lib/projects/files/video-variant-service";
import {
  VIDEO_URL_CONTENT_KIND,
  isVideoUrlContentKind,
} from "@/lib/projects/files/video-url-translation-service";
import { ProjectServiceBase } from "@/lib/projects/project-service-base";
import { ProjectTranslationService } from "@/lib/projects/translations/project-translation-service";
import {
  inferSupportedDocumentTranslationFileFormat,
  inferSupportedImageTranslationFileFormat,
  inferSupportedOfficeTranslationFileFormat,
  inferSupportedVideoTranslationFileFormat,
  inferSupportedWholeFileTranslationFileFormat,
  looksLikeImageUrl,
  looksLikeVideoUrl,
} from "@/lib/translation/file-formats";

function filenameFromSourcePath(sourcePath: string) {
  return sourcePath.split("/").at(-1) ?? sourcePath;
}

export function fileBackedCatSegmentIds(
  sourceFileId: string | null | undefined,
  sourcePath: string,
) {
  const ids = new Set<string>();
  if (sourceFileId) {
    ids.add(sourceFileId);
  }
  ids.add(`binary:${sourcePath}`);
  ids.add(`image:${sourcePath}`);
  ids.add(`video:${sourcePath}`);
  return [...ids];
}

function binaryFileExternalStringId(sourceFileId: string, sourcePath: string) {
  return sourceFileId || `binary:${sourcePath}`;
}

function imageFileExternalStringId(sourceFileId: string, sourcePath: string) {
  return binaryFileExternalStringId(sourceFileId, sourcePath);
}

function videoFileExternalStringId(sourceFileId: string, sourcePath: string) {
  return binaryFileExternalStringId(sourceFileId, sourcePath);
}

function officeFileExternalStringId(sourceFileId: string, sourcePath: string) {
  return binaryFileExternalStringId(sourceFileId, sourcePath);
}

function documentFileExternalStringId(sourceFileId: string, sourcePath: string) {
  return binaryFileExternalStringId(sourceFileId, sourcePath);
}

function toCatTranslation(row: {
  id: string;
  text: string;
  status: "draft" | "needs_review" | "approved" | "rejected";
  contentKind?: ProjectFileContentEditorTranslation["contentKind"];
  targetAssetUrl?: string | null;
  imageVariantId?: string | null;
}): ProjectFileContentEditorTranslation {
  return {
    text: row.text,
    externalTranslationId: row.id,
    isApproved: row.status === "approved",
    ...(row.contentKind ? { contentKind: row.contentKind } : {}),
    ...(row.targetAssetUrl !== undefined ? { targetAssetUrl: row.targetAssetUrl } : {}),
    ...(row.imageVariantId !== undefined ? { imageVariantId: row.imageVariantId } : {}),
    status: row.status,
  };
}

function mapTextSegment(
  key: {
    id: string;
    key: string;
    sourceText: string;
    context: string | null;
    type: string | null;
    maxLength: number | null;
    metadata: Record<string, unknown> | null;
    isHidden?: boolean;
    sourcePath?: string;
  },
  options?: { includeSourcePath?: boolean },
): ProjectFileContentEditorSegment {
  const isVideoUrl = isVideoUrlContentKind(key.metadata);
  const isImageUrl = isImageUrlContentKind(key.metadata);
  const contentKind = isVideoUrl
    ? VIDEO_URL_CONTENT_KIND
    : isImageUrl
      ? IMAGE_URL_CONTENT_KIND
      : undefined;
  const looksLikeImage = looksLikeImageUrl(key.sourceText);
  const looksLikeVideo = looksLikeVideoUrl(key.sourceText);

  return {
    externalStringId: key.id,
    key: key.key,
    sourceText: key.sourceText,
    context: key.context,
    type: key.type,
    ...(key.maxLength != null && key.maxLength > 0 ? { maxLength: key.maxLength } : {}),
    ...(key.isHidden ? { isHidden: true as const } : {}),
    ...(contentKind ? { contentKind } : {}),
    ...(contentKind === IMAGE_URL_CONTENT_KIND || contentKind === VIDEO_URL_CONTENT_KIND
      ? { sourceAssetUrl: key.sourceText }
      : {}),
    ...(looksLikeImage || isImageUrl ? { looksLikeImageUrl: looksLikeImage || isImageUrl } : {}),
    ...(looksLikeVideo || isVideoUrl ? { looksLikeVideoUrl: looksLikeVideo || isVideoUrl } : {}),
    ...(options?.includeSourcePath && key.sourcePath ? { sourcePath: key.sourcePath } : {}),
  };
}

export class NativeContentEditorService extends ProjectServiceBase {
  private readonly translations: ProjectTranslationService;
  private readonly comments: NativeContentEditorCommentService;

  constructor(
    database: typeof db = db,
    translations: ProjectTranslationService = new ProjectTranslationService(database),
    comments?: NativeContentEditorCommentService,
  ) {
    super(database, "projects.cat");
    this.translations = translations;
    this.comments = comments ?? new NativeContentEditorCommentService(database, translations);
  }

  async getCatFile(input: {
    organizationId: string;
    projectId: string;
    sourcePath: string;
    targetLocale: string;
    canEditTranslations: boolean;
    organizationSlug: string;
    pagination?: ProjectFileContentEditorPaginationInput;
    sourcePaths?: readonly string[] | null;
  }): Promise<ProjectFileContentEditorQueueFile | null> {
    if (isContentEditorAllFilesSourcePath(input.sourcePath)) {
      return this.getAllFilesCatQueue(input);
    }

    const sourceFile = await this.translations.getRepositorySourceFileByPath({
      organizationId: input.organizationId,
      projectId: input.projectId,
      sourcePath: input.sourcePath,
    });

    if (!sourceFile) {
      return null;
    }

    if (inferSupportedImageTranslationFileFormat(input.sourcePath)) {
      return this.buildImageCatFileResponse({
        input,
        sourceFileId: sourceFile.id,
      });
    }

    if (inferSupportedVideoTranslationFileFormat(input.sourcePath)) {
      return this.buildVideoCatFileResponse({
        input,
        sourceFileId: sourceFile.id,
      });
    }

    if (inferSupportedOfficeTranslationFileFormat(input.sourcePath)) {
      return this.buildOfficeCatFileResponse({
        input,
        sourceFileId: sourceFile.id,
      });
    }

    if (inferSupportedDocumentTranslationFileFormat(input.sourcePath)) {
      return this.buildDocumentCatFileResponse({
        input,
        sourceFileId: sourceFile.id,
      });
    }

    const paginationInput = input.pagination ?? {
      offset: 0,
      limit: legacyNativeContentEditorSegmentLimit,
      search: undefined,
      queueFilter: "all",
      queueSort: "file_order",
      paginated: false,
    };

    if (!paginationInput.paginated) {
      const keys = await this.translations.listKeysForFile({
        organizationId: input.organizationId,
        projectId: input.projectId,
        repositorySourceFileId: sourceFile.id,
        limit: legacyNativeContentEditorSegmentLimit + 1,
      });

      const truncated = keys.length > legacyNativeContentEditorSegmentLimit;
      const visibleKeys = truncated ? keys.slice(0, legacyNativeContentEditorSegmentLimit) : keys;

      return this.buildCatFileResponse({
        input,
        visibleKeys,
        truncated,
        pagination: undefined,
      });
    }

    const [totalCount, keys] = await Promise.all([
      this.translations.countKeysForFile({
        organizationId: input.organizationId,
        projectId: input.projectId,
        repositorySourceFileId: sourceFile.id,
        targetLocale: input.targetLocale,
        search: paginationInput.search,
        queueFilter: paginationInput.queueFilter,
      }),
      this.translations.listKeysForFile({
        organizationId: input.organizationId,
        projectId: input.projectId,
        repositorySourceFileId: sourceFile.id,
        targetLocale: input.targetLocale,
        limit: paginationInput.limit,
        offset: paginationInput.offset,
        search: paginationInput.search,
        queueFilter: paginationInput.queueFilter,
        queueSort: paginationInput.queueSort,
      }),
    ]);

    const pagination = buildCatFilePagination({
      offset: paginationInput.offset,
      limit: paginationInput.limit,
      returnedCount: keys.length,
      totalCount,
    });

    return this.buildCatFileResponse({
      input,
      visibleKeys: keys,
      truncated: pagination.hasMore,
      pagination,
    });
  }

  private async buildImageCatFileResponse(input: {
    input: {
      organizationId: string;
      projectId: string;
      sourcePath: string;
      targetLocale: string;
      canEditTranslations: boolean;
      organizationSlug: string;
    };
    sourceFileId: string;
  }): Promise<ProjectFileContentEditorQueueFile> {
    const [latestVersion, variant] = await Promise.all([
      getLatestRepositorySourceFileVersion({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        db: this.database,
      }),
      getImageVariant({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        targetLocale: input.input.targetLocale,
        db: this.database,
      }),
    ]);

    const sourceStoredFileId = latestVersion?.storedFileId ?? null;
    const targetStoredFileId = variant?.storedFileId ?? null;
    const sourceAssetUrl = sourceStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: sourceStoredFileId,
        })
      : null;
    const targetAssetUrl = targetStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: targetStoredFileId,
        })
      : null;

    return {
      sourcePath: input.input.sourcePath,
      filename: filenameFromSourcePath(input.input.sourcePath),
      provider: null,
      targetLocale: input.input.targetLocale,
      canEditTranslations: input.input.canEditTranslations,
      truncated: false,
      segments: [
        {
          externalStringId: imageFileExternalStringId(input.sourceFileId, input.input.sourcePath),
          key: input.input.sourcePath,
          sourceText: input.input.sourcePath,
          context: null,
          type: null,
          contentKind: "image_file",
          sourceAssetUrl,
          targetAssetUrl,
          imageVariantId: variant?.id ?? null,
        },
      ],
    };
  }

  private async buildVideoCatFileResponse(input: {
    input: {
      organizationId: string;
      projectId: string;
      sourcePath: string;
      targetLocale: string;
      canEditTranslations: boolean;
      organizationSlug: string;
    };
    sourceFileId: string;
  }): Promise<ProjectFileContentEditorQueueFile> {
    const [latestVersion, variant] = await Promise.all([
      getLatestRepositorySourceFileVersion({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        db: this.database,
      }),
      getVideoVariant({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        targetLocale: input.input.targetLocale,
        db: this.database,
      }),
    ]);

    const sourceStoredFileId = latestVersion?.storedFileId ?? null;
    const targetStoredFileId = variant?.storedFileId ?? null;
    const sourceAssetUrl = sourceStoredFileId
      ? projectVideoAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: sourceStoredFileId,
        })
      : null;
    const targetAssetUrl = targetStoredFileId
      ? projectVideoAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: targetStoredFileId,
        })
      : null;

    return {
      sourcePath: input.input.sourcePath,
      filename: filenameFromSourcePath(input.input.sourcePath),
      provider: null,
      targetLocale: input.input.targetLocale,
      canEditTranslations: input.input.canEditTranslations,
      truncated: false,
      segments: [
        {
          externalStringId: videoFileExternalStringId(input.sourceFileId, input.input.sourcePath),
          key: input.input.sourcePath,
          sourceText: input.input.sourcePath,
          context: null,
          type: null,
          contentKind: "video_file",
          sourceAssetUrl,
          targetAssetUrl,
          imageVariantId: variant?.id ?? null,
        },
      ],
    };
  }

  private async buildOfficeCatFileResponse(input: {
    input: {
      organizationId: string;
      projectId: string;
      sourcePath: string;
      targetLocale: string;
      canEditTranslations: boolean;
      organizationSlug: string;
    };
    sourceFileId: string;
  }): Promise<ProjectFileContentEditorQueueFile> {
    const [latestVersion, variant] = await Promise.all([
      getLatestRepositorySourceFileVersion({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        db: this.database,
      }),
      getImageVariant({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        targetLocale: input.input.targetLocale,
        db: this.database,
      }),
    ]);

    const sourceStoredFileId = latestVersion?.storedFileId ?? null;
    const targetStoredFileId = variant?.storedFileId ?? null;
    const sourceAssetUrl = sourceStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: sourceStoredFileId,
        })
      : null;
    const targetAssetUrl = targetStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: targetStoredFileId,
        })
      : null;

    return {
      sourcePath: input.input.sourcePath,
      filename: filenameFromSourcePath(input.input.sourcePath),
      provider: null,
      targetLocale: input.input.targetLocale,
      canEditTranslations: input.input.canEditTranslations,
      truncated: false,
      segments: [
        {
          externalStringId: officeFileExternalStringId(input.sourceFileId, input.input.sourcePath),
          key: input.input.sourcePath,
          sourceText: input.input.sourcePath,
          context: null,
          type: null,
          contentKind: "office_file",
          sourceAssetUrl,
          targetAssetUrl,
          imageVariantId: variant?.id ?? null,
        },
      ],
    };
  }

  private async buildDocumentCatFileResponse(input: {
    input: {
      organizationId: string;
      projectId: string;
      sourcePath: string;
      targetLocale: string;
      canEditTranslations: boolean;
      organizationSlug: string;
    };
    sourceFileId: string;
  }): Promise<ProjectFileContentEditorQueueFile> {
    const [latestVersion, variant] = await Promise.all([
      getLatestRepositorySourceFileVersion({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        db: this.database,
      }),
      getImageVariant({
        organizationId: input.input.organizationId,
        projectId: input.input.projectId,
        sourcePath: input.input.sourcePath,
        targetLocale: input.input.targetLocale,
        db: this.database,
      }),
    ]);

    const sourceStoredFileId = latestVersion?.storedFileId ?? null;
    const targetStoredFileId = variant?.storedFileId ?? null;
    const sourceAssetUrl = sourceStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: sourceStoredFileId,
        })
      : null;
    const targetAssetUrl = targetStoredFileId
      ? projectImageAssetPath({
          organizationSlug: input.input.organizationSlug,
          projectId: input.input.projectId,
          fileId: targetStoredFileId,
        })
      : null;

    return {
      sourcePath: input.input.sourcePath,
      filename: filenameFromSourcePath(input.input.sourcePath),
      provider: null,
      targetLocale: input.input.targetLocale,
      canEditTranslations: input.input.canEditTranslations,
      truncated: false,
      segments: [
        {
          externalStringId: documentFileExternalStringId(
            input.sourceFileId,
            input.input.sourcePath,
          ),
          key: input.input.sourcePath,
          sourceText: input.input.sourcePath,
          context: null,
          type: null,
          contentKind: "document",
          sourceAssetUrl,
          targetAssetUrl,
          imageVariantId: variant?.id ?? null,
        },
      ],
    };
  }

  private async getAllFilesCatQueue(input: {
    organizationId: string;
    projectId: string;
    targetLocale: string;
    canEditTranslations: boolean;
    pagination?: ProjectFileContentEditorPaginationInput;
    sourcePaths?: readonly string[] | null;
  }): Promise<ProjectFileContentEditorQueueFile> {
    const paginationInput = input.pagination ?? {
      offset: 0,
      limit: legacyNativeContentEditorSegmentLimit,
      search: undefined,
      queueFilter: "all",
      queueSort: "file_order",
      paginated: true,
    };

    const limit = paginationInput.paginated
      ? paginationInput.limit
      : legacyNativeContentEditorSegmentLimit + 1;
    const offset = paginationInput.paginated ? paginationInput.offset : 0;

    const [totalCount, keys] = await Promise.all([
      this.translations.countKeysForProject({
        organizationId: input.organizationId,
        projectId: input.projectId,
        targetLocale: input.targetLocale,
        search: paginationInput.search,
        queueFilter: paginationInput.queueFilter,
        sourcePaths: input.sourcePaths,
      }),
      this.translations.listKeysForProject({
        organizationId: input.organizationId,
        projectId: input.projectId,
        targetLocale: input.targetLocale,
        limit,
        offset,
        search: paginationInput.search,
        queueFilter: paginationInput.queueFilter,
        queueSort: paginationInput.queueSort,
        sourcePaths: input.sourcePaths,
      }),
    ]);

    const visibleKeys = paginationInput.paginated
      ? keys
      : keys.slice(0, legacyNativeContentEditorSegmentLimit);
    const truncated = paginationInput.paginated
      ? offset + visibleKeys.length < totalCount
      : keys.length > legacyNativeContentEditorSegmentLimit;

    const pagination = paginationInput.paginated
      ? buildCatFilePagination({
          offset,
          limit: paginationInput.limit,
          returnedCount: visibleKeys.length,
          totalCount,
        })
      : undefined;

    return {
      sourcePath: CONTENT_EDITOR_ALL_FILES_SOURCE_PATH,
      filename: CAT_ALL_FILES_FILENAME,
      provider: null,
      targetLocale: input.targetLocale,
      canEditTranslations: input.canEditTranslations,
      truncated,
      pagination,
      segments: visibleKeys.map((key) => mapTextSegment(key, { includeSourcePath: true })),
    };
  }

  private async buildCatFileResponse(input: {
    input: {
      sourcePath: string;
      targetLocale: string;
      canEditTranslations: boolean;
      organizationId: string;
      projectId: string;
    };
    visibleKeys: Awaited<ReturnType<ProjectTranslationService["listKeysForFile"]>>;
    truncated: boolean;
    pagination: ReturnType<typeof buildCatFilePagination> | undefined;
  }): Promise<ProjectFileContentEditorQueueFile> {
    return {
      sourcePath: input.input.sourcePath,
      filename: filenameFromSourcePath(input.input.sourcePath),
      provider: null,
      targetLocale: input.input.targetLocale,
      canEditTranslations: input.input.canEditTranslations,
      truncated: input.truncated,
      pagination: input.pagination,
      segments: input.visibleKeys.map((key) => mapTextSegment(key)),
    };
  }

  async setKeysHidden(input: {
    organizationId: string;
    projectId: string;
    translationKeyIds: string[];
    isHidden: boolean;
    sourcePath?: string;
  }) {
    let repositorySourceFileId: string | undefined;
    if (input.sourcePath && !isContentEditorAllFilesSourcePath(input.sourcePath)) {
      const sourceFile = await this.translations.getRepositorySourceFileByPath({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
      });
      if (!sourceFile) {
        return { updatedCount: 0 };
      }
      repositorySourceFileId = sourceFile.id;
    }

    return this.translations.setKeysHidden({
      organizationId: input.organizationId,
      projectId: input.projectId,
      translationKeyIds: input.translationKeyIds,
      isHidden: input.isHidden,
      repositorySourceFileId,
    });
  }

  async setKeyMaxLength(input: {
    organizationId: string;
    projectId: string;
    translationKeyId: string;
    maxLength: number | null;
    sourcePath?: string;
  }) {
    let repositorySourceFileId: string | undefined;
    if (input.sourcePath && !isContentEditorAllFilesSourcePath(input.sourcePath)) {
      const sourceFile = await this.translations.getRepositorySourceFileByPath({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
      });
      if (!sourceFile) {
        return { updated: false, maxLength: null };
      }
      repositorySourceFileId = sourceFile.id;
    }

    return this.translations.setKeyMaxLength({
      organizationId: input.organizationId,
      projectId: input.projectId,
      translationKeyId: input.translationKeyId,
      maxLength: input.maxLength,
      repositorySourceFileId,
    });
  }

  async saveComment(input: Parameters<NativeContentEditorCommentService["save"]>[0]) {
    return this.comments.save(input);
  }

  async resolveLegacyIssueComment(
    input: Parameters<NativeContentEditorCommentService["resolveLegacyIssue"]>[0],
  ) {
    return this.comments.resolveLegacyIssue(input);
  }

  async saveTranslation(input: {
    organizationId: string;
    projectId: string;
    sourcePath: string;
    targetLocale: string;
    translationKeyId: string;
    text: string;
    approve?: boolean;
    actorUserId?: string;
    provenance?: "manual" | "translation_job" | "import" | "agent";
    sourceJobId?: string;
  }): Promise<ProjectFileContentEditorTranslation | null> {
    const sourceFile = await this.translations.getRepositorySourceFileByPath({
      organizationId: input.organizationId,
      projectId: input.projectId,
      sourcePath: input.sourcePath,
    });

    if (!sourceFile) {
      return null;
    }

    const [key] = await this.database
      .select({
        id: schema.projectTranslationKeys.id,
        sourceText: schema.projectTranslationKeys.sourceText,
      })
      .from(schema.projectTranslationKeys)
      .where(
        and(
          eq(schema.projectTranslationKeys.id, input.translationKeyId),
          eq(schema.projectTranslationKeys.projectId, input.projectId),
          eq(schema.projectTranslationKeys.repositorySourceFileId, sourceFile.id),
        ),
      )
      .limit(1);

    if (!key) {
      return null;
    }

    if (input.sourceJobId) {
      const [project] = await this.database
        .select({ sourceLocale: schema.projects.sourceLocale })
        .from(schema.projects)
        .where(eq(schema.projects.id, input.projectId));
      await captureAnalysis({
        organizationId: input.organizationId,
        projectId: input.projectId,
        jobId: input.sourceJobId,
        sourceLocale: project?.sourceLocale ?? "en",
        targetLocale: input.targetLocale,
        sourceEntries: { [key.id]: key.sourceText },
        billable: (input.provenance ?? "manual") === "manual",
        step: input.approve ? "review" : "translation",
      });
    }
    const status = input.approve ? "approved" : "draft";
    const provenance = input.provenance ?? "manual";
    const reviewedAt = input.approve ? new Date() : null;
    const reviewedByUserId = input.approve ? (input.actorUserId ?? null) : null;

    const [saved] = await this.database
      .insert(schema.projectTranslations)
      .values({
        organizationId: input.organizationId,
        projectId: input.projectId,
        translationKeyId: key.id,
        targetLocale: input.targetLocale,
        text: input.text,
        status,
        provenance,
        sourceJobId: input.sourceJobId ?? null,
        reviewedByUserId,
        reviewedAt,
      })
      .onConflictDoUpdate({
        target: [
          schema.projectTranslations.translationKeyId,
          schema.projectTranslations.targetLocale,
        ],
        set: {
          text: input.text,
          status,
          provenance,
          sourceJobId: input.sourceJobId ?? null,
          reviewedByUserId,
          reviewedAt,
          updatedAt: new Date(),
        },
      })
      .returning({
        id: schema.projectTranslations.id,
        text: schema.projectTranslations.text,
        status: schema.projectTranslations.status,
      });

    if (!saved) {
      return null;
    }

    this.log.debug(
      {
        organizationId: input.organizationId,
        projectId: input.projectId,
        translationKeyId: input.translationKeyId,
        status,
      },
      "saved native CAT translation",
    );

    if (input.sourceJobId && input.text.trim())
      await captureCompletions({
        organizationId: input.organizationId,
        jobId: input.sourceJobId,
        targetLocale: input.targetLocale,
        sourceEntries: { [key.id]: key.sourceText },
        provenance: (input.provenance ?? "manual") === "manual" ? "human" : "automated",
        step: input.approve ? "review" : "translation",
      });
    return toCatTranslation(saved);
  }

  async updateTranslationStatus(input: {
    organizationId: string;
    projectId: string;
    translationKeyId: string;
    targetLocale: string;
    status: "needs_review" | "approved" | "rejected";
    actorUserId?: string;
  }) {
    const reviewedAt =
      input.status === "approved" || input.status === "rejected" ? new Date() : null;
    const reviewedByUserId =
      input.status === "approved" || input.status === "rejected"
        ? (input.actorUserId ?? null)
        : null;

    const [updated] = await this.database
      .update(schema.projectTranslations)
      .set({
        status: input.status,
        reviewedAt,
        reviewedByUserId,
      })
      .where(
        and(
          eq(schema.projectTranslations.organizationId, input.organizationId),
          eq(schema.projectTranslations.projectId, input.projectId),
          eq(schema.projectTranslations.translationKeyId, input.translationKeyId),
          eq(schema.projectTranslations.targetLocale, input.targetLocale),
        ),
      )
      .returning({
        id: schema.projectTranslations.id,
        text: schema.projectTranslations.text,
        status: schema.projectTranslations.status,
      });

    if (updated) {
      this.log.debug(
        {
          organizationId: input.organizationId,
          projectId: input.projectId,
          translationKeyId: input.translationKeyId,
          status: input.status,
        },
        "updated native CAT translation status",
      );
    }

    return updated ? toCatTranslation(updated) : null;
  }

  private async findTranslationKeyForSegment(input: {
    organizationId: string;
    projectId: string;
    sourcePath: string;
    externalStringId: string;
  }): Promise<{
    id: string;
    metadata: Record<string, unknown> | null;
  } | null> {
    // All-files CAT uses the sentinel sourcePath `*`. Keys are scoped to the
    // project only — callers may also send the real per-segment path.
    if (isContentEditorAllFilesSourcePath(input.sourcePath)) {
      const [key] = await this.database
        .select({
          id: schema.projectTranslationKeys.id,
          metadata: schema.projectTranslationKeys.metadata,
        })
        .from(schema.projectTranslationKeys)
        .where(
          and(
            eq(schema.projectTranslationKeys.id, input.externalStringId),
            eq(schema.projectTranslationKeys.organizationId, input.organizationId),
            eq(schema.projectTranslationKeys.projectId, input.projectId),
          ),
        )
        .limit(1);

      return key ?? null;
    }

    const sourceFile = await this.translations.getRepositorySourceFileByPath({
      organizationId: input.organizationId,
      projectId: input.projectId,
      sourcePath: input.sourcePath,
    });

    if (!sourceFile) {
      return null;
    }

    const [key] = await this.database
      .select({
        id: schema.projectTranslationKeys.id,
        metadata: schema.projectTranslationKeys.metadata,
      })
      .from(schema.projectTranslationKeys)
      .where(
        and(
          eq(schema.projectTranslationKeys.id, input.externalStringId),
          eq(schema.projectTranslationKeys.organizationId, input.organizationId),
          eq(schema.projectTranslationKeys.projectId, input.projectId),
          eq(schema.projectTranslationKeys.repositorySourceFileId, sourceFile.id),
        ),
      )
      .limit(1);

    return key ?? null;
  }

  async getSegmentTarget(input: {
    organizationId: string;
    projectId: string;
    sourcePath: string;
    targetLocale: string;
    externalStringId: string;
    organizationSlug: string;
  }): Promise<ProjectFileContentEditorTranslation | null | "not_found"> {
    if (
      !isContentEditorAllFilesSourcePath(input.sourcePath) &&
      inferSupportedWholeFileTranslationFileFormat(input.sourcePath)
    ) {
      const sourceFile = await this.translations.getRepositorySourceFileByPath({
        organizationId: input.organizationId,
        projectId: input.projectId,
        sourcePath: input.sourcePath,
      });

      if (!sourceFile) {
        return "not_found";
      }

      const isVideo = Boolean(inferSupportedVideoTranslationFileFormat(input.sourcePath));
      const isOffice = Boolean(inferSupportedOfficeTranslationFileFormat(input.sourcePath));
      const isDocument = Boolean(inferSupportedDocumentTranslationFileFormat(input.sourcePath));
      const contentKind = isVideo
        ? ("video_file" as const)
        : isOffice
          ? ("office_file" as const)
          : isDocument
            ? ("document" as const)
            : ("image_file" as const);
      const expectedIds = new Set(fileBackedCatSegmentIds(sourceFile.id, input.sourcePath));
      if (!expectedIds.has(input.externalStringId)) {
        return "not_found";
      }

      const variant = isVideo
        ? await getVideoVariant({
            organizationId: input.organizationId,
            projectId: input.projectId,
            sourcePath: input.sourcePath,
            targetLocale: input.targetLocale,
            db: this.database,
          })
        : await getImageVariant({
            organizationId: input.organizationId,
            projectId: input.projectId,
            sourcePath: input.sourcePath,
            targetLocale: input.targetLocale,
            db: this.database,
          });

      const targetAssetUrl = variant?.storedFileId
        ? (isVideo ? projectVideoAssetPath : projectImageAssetPath)({
            organizationSlug: input.organizationSlug,
            projectId: input.projectId,
            fileId: variant.storedFileId,
          })
        : null;

      if (!variant) {
        return {
          text: "",
          externalTranslationId: null,
          isApproved: false,
          contentKind,
          targetAssetUrl: null,
          imageVariantId: null,
          status: "draft",
        };
      }

      return toCatTranslation({
        id: variant.id,
        text: targetAssetUrl ?? "",
        status: variant.status,
        contentKind,
        targetAssetUrl,
        imageVariantId: variant.id,
      });
    }

    const key = await this.findTranslationKeyForSegment({
      organizationId: input.organizationId,
      projectId: input.projectId,
      sourcePath: input.sourcePath,
      externalStringId: input.externalStringId,
    });

    if (!key) {
      return "not_found";
    }

    const translation = (
      await this.translations.getTranslationsByKeyIds({
        organizationId: input.organizationId,
        projectId: input.projectId,
        translationKeyIds: [key.id],
        targetLocale: input.targetLocale,
      })
    )[0];

    if (!translation) {
      return null;
    }

    const contentKind = isVideoUrlContentKind(key.metadata)
      ? VIDEO_URL_CONTENT_KIND
      : isImageUrlContentKind(key.metadata)
        ? IMAGE_URL_CONTENT_KIND
        : undefined;

    return toCatTranslation({
      ...translation,
      ...(contentKind
        ? {
            contentKind,
            targetAssetUrl: translation.text,
          }
        : {}),
    });
  }

  async getSegmentComments(input: {
    organizationId: string;
    projectId: string;
    sourcePath: string;
    targetLocale: string;
    externalStringId: string;
  }): Promise<ProjectFileContentEditorComment[]> {
    if (
      !isContentEditorAllFilesSourcePath(input.sourcePath) &&
      inferSupportedWholeFileTranslationFileFormat(input.sourcePath)
    ) {
      return [];
    }

    const key = await this.findTranslationKeyForSegment({
      organizationId: input.organizationId,
      projectId: input.projectId,
      sourcePath: input.sourcePath,
      externalStringId: input.externalStringId,
    });

    if (!key) {
      return [];
    }

    const commentsByKeyId = await this.comments.listByKeyIds({
      organizationId: input.organizationId,
      projectId: input.projectId,
      translationKeyIds: [key.id],
      targetLocale: input.targetLocale,
    });

    return commentsByKeyId.get(key.id) ?? [];
  }
}

export const nativeCatService = new NativeContentEditorService();

export const getNativeProjectContentEditorFile = (
  input: Parameters<NativeContentEditorService["getCatFile"]>[0],
) => nativeCatService.getCatFile(input);

export const getNativeProjectContentEditorSegmentTarget = (
  input: Parameters<NativeContentEditorService["getSegmentTarget"]>[0],
) => nativeCatService.getSegmentTarget(input);

export const getNativeProjectContentEditorSegmentComments = (
  input: Parameters<NativeContentEditorService["getSegmentComments"]>[0],
) => nativeCatService.getSegmentComments(input);

export const saveNativeProjectContentEditorTranslation = (
  input: Parameters<NativeContentEditorService["saveTranslation"]>[0],
) => nativeCatService.saveTranslation(input);

export const saveNativeProjectContentEditorComment = (
  input: Parameters<NativeContentEditorService["saveComment"]>[0],
) => nativeCatService.saveComment(input);

export const resolveNativeProjectContentEditorLegacyIssueComment = (
  input: Parameters<NativeContentEditorService["resolveLegacyIssueComment"]>[0],
) => nativeCatService.resolveLegacyIssueComment(input);

export const updateNativeProjectTranslationStatus = (
  input: Parameters<NativeContentEditorService["updateTranslationStatus"]>[0],
) => nativeCatService.updateTranslationStatus(input);

export const setNativeProjectContentEditorStringsHidden = (
  input: Parameters<NativeContentEditorService["setKeysHidden"]>[0],
) => nativeCatService.setKeysHidden(input);

export const setNativeProjectContentEditorKeyMaxLength = (
  input: Parameters<NativeContentEditorService["setKeyMaxLength"]>[0],
) => nativeCatService.setKeyMaxLength(input);

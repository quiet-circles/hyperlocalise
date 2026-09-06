WITH ranked AS (
	SELECT id,
		first_value(id) OVER (
			PARTITION BY organization_id, job_id, segment_id, source_revision, target_locale, step
			ORDER BY is_current DESC, created_at DESC, id DESC
		) AS keeper_id
	FROM reporting_analyses
)
UPDATE reporting_activity
SET analysis_id = ranked.keeper_id
FROM ranked
WHERE reporting_activity.analysis_id = ranked.id AND ranked.id <> ranked.keeper_id;--> statement-breakpoint
DELETE FROM reporting_analyses
USING (
	SELECT id,
		first_value(id) OVER (
			PARTITION BY organization_id, job_id, segment_id, source_revision, target_locale, step
			ORDER BY is_current DESC, created_at DESC, id DESC
		) AS keeper_id
	FROM reporting_analyses
) ranked
WHERE reporting_analyses.id = ranked.id AND ranked.id <> ranked.keeper_id;--> statement-breakpoint
DROP INDEX "reporting_analysis_identity";--> statement-breakpoint
ALTER TABLE "reporting_analyses" ADD CONSTRAINT "reporting_analysis_identity" UNIQUE NULLS NOT DISTINCT("organization_id","job_id","segment_id","source_revision","target_locale","step");
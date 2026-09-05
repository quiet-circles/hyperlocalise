CREATE TABLE "reporting_activity" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"project_id" text,
	"job_id" text,
	"operation_key" text NOT NULL,
	"kind" text NOT NULL,
	"step" text NOT NULL,
	"target_locale" text,
	"analysis_id" uuid,
	"status" text,
	"duration_ms" integer,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_analyses" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"project_id" text,
	"job_id" text,
	"step" text DEFAULT 'translation' NOT NULL,
	"segment_id" text NOT NULL,
	"source_revision" text NOT NULL,
	"source_locale" text NOT NULL,
	"target_locale" text NOT NULL,
	"words" integer,
	"billable" boolean DEFAULT false NOT NULL,
	"is_current" boolean DEFAULT true NOT NULL,
	"match_score" integer,
	"bucket" text NOT NULL,
	"algorithm_version" integer DEFAULT 1 NOT NULL,
	"rate_id" uuid,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_audit" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"actor_id" uuid NOT NULL,
	"resource_id" text NOT NULL,
	"action" text NOT NULL,
	"before" jsonb,
	"after" jsonb,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_budgets" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"project_id" text,
	"budget" numeric(24, 8) NOT NULL,
	"rate_card_name" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_costs" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"project_id" text,
	"job_id" text,
	"operation_key" text NOT NULL,
	"kind" text NOT NULL,
	"step" text NOT NULL,
	"target_locale" text,
	"amount_usd" numeric(24, 8),
	"basis" text NOT NULL,
	"rate_id" uuid,
	"time_entry_id" uuid,
	"provider" text,
	"model" text,
	"input_tokens" integer,
	"output_tokens" integer,
	"token_categories" jsonb,
	"pricing_version" text,
	"note" text,
	"voided" boolean DEFAULT false NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_rates" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"name" text NOT NULL,
	"source_locale" text NOT NULL,
	"target_locale" text NOT NULL,
	"step" text NOT NULL,
	"basis" text NOT NULL,
	"rate" numeric(24, 8) NOT NULL,
	"percentages" jsonb DEFAULT '{}'::jsonb NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_rollout" (
	"id" integer PRIMARY KEY DEFAULT 1 NOT NULL,
	"started_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_task_rates" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"job_id" text,
	"step" text NOT NULL,
	"rate_id" uuid,
	"estimated_minutes" integer,
	"override_usd" numeric(24, 8),
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "reporting_time_entries" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"project_id" text,
	"job_id" text,
	"contributor_id" uuid NOT NULL,
	"step" text NOT NULL,
	"target_locale" text NOT NULL,
	"work_date" timestamp with time zone NOT NULL,
	"minutes" integer NOT NULL,
	"note" text,
	"rate_id" uuid,
	"voided" boolean DEFAULT false NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "reporting_activity" ADD CONSTRAINT "reporting_activity_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_activity" ADD CONSTRAINT "reporting_activity_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_activity" ADD CONSTRAINT "reporting_activity_job_id_jobs_id_fk" FOREIGN KEY ("job_id") REFERENCES "public"."jobs"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_activity" ADD CONSTRAINT "reporting_activity_analysis_id_reporting_analyses_id_fk" FOREIGN KEY ("analysis_id") REFERENCES "public"."reporting_analyses"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_analyses" ADD CONSTRAINT "reporting_analyses_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_analyses" ADD CONSTRAINT "reporting_analyses_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_analyses" ADD CONSTRAINT "reporting_analyses_job_id_jobs_id_fk" FOREIGN KEY ("job_id") REFERENCES "public"."jobs"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_analyses" ADD CONSTRAINT "reporting_analyses_rate_id_reporting_rates_id_fk" FOREIGN KEY ("rate_id") REFERENCES "public"."reporting_rates"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_audit" ADD CONSTRAINT "reporting_audit_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_audit" ADD CONSTRAINT "reporting_audit_actor_id_users_id_fk" FOREIGN KEY ("actor_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_budgets" ADD CONSTRAINT "reporting_budgets_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_budgets" ADD CONSTRAINT "reporting_budgets_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_costs" ADD CONSTRAINT "reporting_costs_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_costs" ADD CONSTRAINT "reporting_costs_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_costs" ADD CONSTRAINT "reporting_costs_job_id_jobs_id_fk" FOREIGN KEY ("job_id") REFERENCES "public"."jobs"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_costs" ADD CONSTRAINT "reporting_costs_rate_id_reporting_rates_id_fk" FOREIGN KEY ("rate_id") REFERENCES "public"."reporting_rates"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_costs" ADD CONSTRAINT "reporting_costs_time_entry_id_reporting_time_entries_id_fk" FOREIGN KEY ("time_entry_id") REFERENCES "public"."reporting_time_entries"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_rates" ADD CONSTRAINT "reporting_rates_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_task_rates" ADD CONSTRAINT "reporting_task_rates_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_task_rates" ADD CONSTRAINT "reporting_task_rates_job_id_jobs_id_fk" FOREIGN KEY ("job_id") REFERENCES "public"."jobs"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_task_rates" ADD CONSTRAINT "reporting_task_rates_rate_id_reporting_rates_id_fk" FOREIGN KEY ("rate_id") REFERENCES "public"."reporting_rates"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_time_entries" ADD CONSTRAINT "reporting_time_entries_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_time_entries" ADD CONSTRAINT "reporting_time_entries_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_time_entries" ADD CONSTRAINT "reporting_time_entries_job_id_jobs_id_fk" FOREIGN KEY ("job_id") REFERENCES "public"."jobs"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_time_entries" ADD CONSTRAINT "reporting_time_entries_contributor_id_users_id_fk" FOREIGN KEY ("contributor_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "reporting_time_entries" ADD CONSTRAINT "reporting_time_entries_rate_id_reporting_rates_id_fk" FOREIGN KEY ("rate_id") REFERENCES "public"."reporting_rates"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
CREATE UNIQUE INDEX "reporting_activity_operation" ON "reporting_activity" USING btree ("organization_id","operation_key");--> statement-breakpoint
CREATE INDEX "reporting_activity_filter" ON "reporting_activity" USING btree ("organization_id","project_id","created_at");--> statement-breakpoint
CREATE UNIQUE INDEX "reporting_analysis_identity" ON "reporting_analyses" USING btree ("organization_id","job_id","segment_id","source_revision","target_locale","step");--> statement-breakpoint
CREATE INDEX "reporting_analysis_project" ON "reporting_analyses" USING btree ("organization_id","project_id","created_at");--> statement-breakpoint
CREATE UNIQUE INDEX "reporting_budgets_project" ON "reporting_budgets" USING btree ("organization_id","project_id");--> statement-breakpoint
CREATE UNIQUE INDEX "reporting_cost_operation" ON "reporting_costs" USING btree ("organization_id","operation_key");--> statement-breakpoint
CREATE INDEX "reporting_cost_filter" ON "reporting_costs" USING btree ("organization_id","project_id","created_at");--> statement-breakpoint
CREATE INDEX "reporting_rates_org" ON "reporting_rates" USING btree ("organization_id");--> statement-breakpoint
CREATE UNIQUE INDEX "reporting_task_rates_job_step" ON "reporting_task_rates" USING btree ("organization_id","job_id","step");--> statement-breakpoint
CREATE INDEX "reporting_time_filter" ON "reporting_time_entries" USING btree ("organization_id","project_id","work_date");
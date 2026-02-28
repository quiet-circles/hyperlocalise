package syncsvc

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	PolicyConservativeCurationPull = "conservative_curation_pull"
)

type LocalReadRequest struct {
	Locales []string
}

type PullOptions struct {
	DryRun                bool
	FailOnConflict        bool
	ApplyCuratedOverDraft bool
	Policy                string
}

type PushOptions struct {
	DryRun         bool
	FailOnConflict bool
	ForceConflicts bool
}

type LocalStore interface {
	ReadSnapshot(ctx context.Context, req LocalReadRequest) (storage.CatalogSnapshot, error)
	ApplyPull(ctx context.Context, plan ApplyPullPlan) (ApplyResult, error)
	BuildPushSnapshot(ctx context.Context, req LocalReadRequest) (storage.CatalogSnapshot, error)
}

type Service struct {
	now func() time.Time
}

func New() *Service {
	return &Service{now: time.Now}
}

type PullInput struct {
	Adapter storage.StorageAdapter
	Local   LocalStore
	Request storage.PullRequest
	Read    LocalReadRequest
	Options PullOptions
}

type PushInput struct {
	Adapter storage.StorageAdapter
	Local   LocalStore
	Read    LocalReadRequest
	Options PushOptions
}

type Report struct {
	Action    string             `json:"action"`
	Creates   []storage.Entry    `json:"creates,omitempty"`
	Updates   []storage.Entry    `json:"updates,omitempty"`
	Unchanged []storage.EntryID  `json:"unchanged,omitempty"`
	Conflicts []storage.Conflict `json:"conflicts,omitempty"`
	Warnings  []storage.Warning  `json:"warnings,omitempty"`
	Applied   []storage.EntryID  `json:"applied,omitempty"`
	Skipped   []storage.EntryID  `json:"skipped,omitempty"`
}

type ApplyPullPlan struct {
	Creates []storage.Entry
	Updates []storage.Entry
}

type ApplyResult struct {
	Applied []storage.EntryID
}

func (s *Service) Pull(ctx context.Context, in PullInput) (Report, error) {
	localSnapshot, err := in.Local.ReadSnapshot(ctx, in.Read)
	if err != nil {
		return Report{}, fmt.Errorf("read local snapshot: %w", err)
	}

	remoteResult, err := in.Adapter.Pull(ctx, in.Request)
	if err != nil {
		return Report{}, fmt.Errorf("pull remote snapshot: %w", err)
	}

	report := buildPullReport(localSnapshot, remoteResult.Snapshot, in.Options)
	report.Action = "pull"
	report.Warnings = append(report.Warnings, remoteResult.Warnings...)

	if in.Options.FailOnConflict && len(report.Conflicts) > 0 {
		return report, fmt.Errorf("pull conflicts detected: %d", len(report.Conflicts))
	}

	if in.Options.DryRun {
		return report, nil
	}

	apply, err := in.Local.ApplyPull(ctx, ApplyPullPlan{
		Creates: report.Creates,
		Updates: report.Updates,
	})
	if err != nil {
		return report, fmt.Errorf("apply pull plan: %w", err)
	}
	report.Applied = apply.Applied

	return report, nil
}

func (s *Service) Push(ctx context.Context, in PushInput) (Report, error) {
	localSnapshot, err := in.Local.BuildPushSnapshot(ctx, in.Read)
	if err != nil {
		return Report{}, fmt.Errorf("build local push snapshot: %w", err)
	}

	remoteResult, err := in.Adapter.Pull(ctx, storage.PullRequest{Locales: in.Read.Locales})
	if err != nil {
		return Report{}, fmt.Errorf("pull remote baseline: %w", err)
	}

	report, pushReq := buildPushReport(localSnapshot, remoteResult.Snapshot, in.Options)
	report.Action = "push"
	report.Warnings = append(report.Warnings, remoteResult.Warnings...)

	if in.Options.FailOnConflict && len(report.Conflicts) > 0 {
		return report, fmt.Errorf("push conflicts detected: %d", len(report.Conflicts))
	}
	if in.Options.DryRun || len(pushReq.Entries) == 0 {
		return report, nil
	}

	pushResult, err := in.Adapter.Push(ctx, pushReq)
	if err != nil {
		return report, fmt.Errorf("push remote changes: %w", err)
	}

	report.Applied = append(report.Applied, pushResult.Applied...)
	report.Skipped = append(report.Skipped, pushResult.Skipped...)
	report.Conflicts = append(report.Conflicts, pushResult.Conflicts...)
	report.Warnings = append(report.Warnings, pushResult.Warnings...)

	return report, nil
}

func buildPullReport(local, remote storage.CatalogSnapshot, opts PullOptions) Report {
	localIndex := indexEntries(local.Entries)
	remoteIndex := indexEntries(remote.Entries)
	report := Report{}

	for id, remoteEntry := range remoteIndex {
		localEntry, exists := localIndex[id]
		if !exists {
			report.Creates = append(report.Creates, markRemoteCurated(remoteEntry))
			continue
		}

		if localEntry.Value == remoteEntry.Value {
			report.Unchanged = append(report.Unchanged, id)
			continue
		}

		conflict, update := decidePullDiff(localEntry, remoteEntry, opts)
		if conflict != nil {
			report.Conflicts = append(report.Conflicts, *conflict)
			continue
		}
		if diags := validateEntryInvariant(*update, localEntry); len(diags) > 0 {
			report.Conflicts = append(report.Conflicts, storage.Conflict{
				ID:          update.ID(),
				Reason:      conflictReasonInvariantViolation,
				LocalValue:  localEntry.Value,
				RemoteValue: update.Value,
				LocalState:  localEntry.Provenance.State,
				RemoteState: update.Provenance.State,
			})
			report.Warnings = append(report.Warnings, storage.Warning{
				Code: "invariant_violation",
				Message: formatInvariantWarning(
					"pull blocked by invariant validation",
					update.ID(),
					diags,
				),
			})
			continue
		}
		report.Updates = append(report.Updates, *update)
	}

	sortReport(&report)
	return report
}

func buildPushReport(local, remote storage.CatalogSnapshot, opts PushOptions) (Report, storage.PushRequest) {
	localIndex := indexEntries(local.Entries)
	remoteIndex := indexEntries(remote.Entries)
	report := Report{}
	pushReq := storage.PushRequest{}

	for id, localEntry := range localIndex {
		remoteEntry, exists := remoteIndex[id]
		if !exists {
			report.Creates = append(report.Creates, localEntry)
			pushReq.Entries = append(pushReq.Entries, localEntry)
			continue
		}

		if localEntry.Value == remoteEntry.Value {
			report.Unchanged = append(report.Unchanged, id)
			continue
		}

		if diags := validateEntryInvariant(localEntry, remoteEntry); len(diags) > 0 {
			report.Conflicts = append(report.Conflicts, storage.Conflict{
				ID:          id,
				Reason:      conflictReasonInvariantViolation,
				LocalValue:  localEntry.Value,
				RemoteValue: remoteEntry.Value,
				LocalState:  localEntry.Provenance.State,
				RemoteState: remoteEntry.Provenance.State,
			})
			report.Warnings = append(report.Warnings, storage.Warning{
				Code:    "invariant_violation",
				Message: formatInvariantWarning("push blocked by invariant validation", id, diags),
			})
			continue
		}

		if strings.EqualFold(localEntry.Provenance.State, storage.StateDraft) &&
			strings.EqualFold(remoteEntry.Provenance.State, storage.StateCurated) {
			if !opts.ForceConflicts {
				report.Conflicts = append(report.Conflicts, storage.Conflict{
					ID:          id,
					Reason:      "draft_vs_curated_remote",
					LocalValue:  localEntry.Value,
					RemoteValue: remoteEntry.Value,
					LocalState:  localEntry.Provenance.State,
					RemoteState: remoteEntry.Provenance.State,
				})
				continue
			}
		}

		localOrigin := strings.ToLower(strings.TrimSpace(localEntry.Provenance.Origin))
		if (localOrigin == "" || localOrigin == storage.OriginUnknown) && !opts.ForceConflicts {
			report.Conflicts = append(report.Conflicts, storage.Conflict{
				ID:          id,
				Reason:      "missing_provenance_value_mismatch",
				LocalValue:  localEntry.Value,
				RemoteValue: remoteEntry.Value,
				LocalState:  localEntry.Provenance.State,
				RemoteState: remoteEntry.Provenance.State,
			})
			continue
		}

		report.Updates = append(report.Updates, localEntry)
		pushReq.Entries = append(pushReq.Entries, localEntry)
	}

	sortReport(&report)
	return report, pushReq
}

func decidePullDiff(localEntry, remoteEntry storage.Entry, opts PullOptions) (*storage.Conflict, *storage.Entry) {
	localState := strings.ToLower(strings.TrimSpace(localEntry.Provenance.State))
	localOrigin := strings.ToLower(strings.TrimSpace(localEntry.Provenance.Origin))

	shouldApplyCuratedOverDraft := opts.ApplyCuratedOverDraft
	if strings.EqualFold(opts.Policy, PolicyConservativeCurationPull) {
		shouldApplyCuratedOverDraft = true
	}

	if localState == storage.StateCurated {
		return &storage.Conflict{
			ID:          localEntry.ID(),
			Reason:      "curated_value_mismatch",
			LocalValue:  localEntry.Value,
			RemoteValue: remoteEntry.Value,
			LocalState:  localEntry.Provenance.State,
			RemoteState: storage.StateCurated,
		}, nil
	}

	if localOrigin == storage.OriginLLM && localState == storage.StateDraft && shouldApplyCuratedOverDraft {
		update := markRemoteCurated(remoteEntry)
		return nil, &update
	}
	if localOrigin == storage.OriginLLM && localState == storage.StateDraft && !shouldApplyCuratedOverDraft {
		return &storage.Conflict{
			ID:          localEntry.ID(),
			Reason:      "curated_over_draft_disabled",
			LocalValue:  localEntry.Value,
			RemoteValue: remoteEntry.Value,
			LocalState:  localEntry.Provenance.State,
			RemoteState: storage.StateCurated,
		}, nil
	}

	if localOrigin == "" || localOrigin == storage.OriginUnknown {
		return &storage.Conflict{
			ID:          localEntry.ID(),
			Reason:      "missing_provenance_value_mismatch",
			LocalValue:  localEntry.Value,
			RemoteValue: remoteEntry.Value,
			LocalState:  localEntry.Provenance.State,
			RemoteState: storage.StateCurated,
		}, nil
	}

	update := markRemoteCurated(remoteEntry)
	return nil, &update
}

func markRemoteCurated(entry storage.Entry) storage.Entry {
	now := time.Now().UTC()
	entry.Provenance.Origin = storage.OriginHuman
	entry.Provenance.State = storage.StateCurated
	if entry.Provenance.UpdatedAt.IsZero() {
		entry.Provenance.UpdatedAt = now
	}
	return entry
}

func indexEntries(entries []storage.Entry) map[storage.EntryID]storage.Entry {
	out := make(map[storage.EntryID]storage.Entry, len(entries))
	for _, entry := range entries {
		out[entry.ID()] = entry
	}
	return out
}

func sortReport(report *Report) {
	slices.SortFunc(report.Unchanged, compareEntryID)
	slices.SortFunc(report.Applied, compareEntryID)
	slices.SortFunc(report.Skipped, compareEntryID)
	slices.SortFunc(report.Creates, func(a, b storage.Entry) int { return compareEntryID(a.ID(), b.ID()) })
	slices.SortFunc(report.Updates, func(a, b storage.Entry) int { return compareEntryID(a.ID(), b.ID()) })
	slices.SortFunc(report.Conflicts, func(a, b storage.Conflict) int { return compareEntryID(a.ID, b.ID) })
}

func compareEntryID(a, b storage.EntryID) int {
	if c := strings.Compare(a.Locale, b.Locale); c != 0 {
		return c
	}
	if c := strings.Compare(a.Key, b.Key); c != 0 {
		return c
	}
	return strings.Compare(a.Context, b.Context)
}

package syncsvc

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

type fakeLocalStore struct {
	readSnapshot storage.CatalogSnapshot
	applied      ApplyPullPlan
}

func (f *fakeLocalStore) ReadSnapshot(_ context.Context, _ LocalReadRequest) (storage.CatalogSnapshot, error) {
	return f.readSnapshot, nil
}

func (f *fakeLocalStore) BuildPushSnapshot(_ context.Context, _ LocalReadRequest) (storage.CatalogSnapshot, error) {
	return f.readSnapshot, nil
}

func (f *fakeLocalStore) ApplyPull(_ context.Context, plan ApplyPullPlan) (ApplyResult, error) {
	f.applied = plan
	applied := make([]storage.EntryID, 0, len(plan.Creates)+len(plan.Updates))
	for _, e := range plan.Creates {
		applied = append(applied, e.ID())
	}
	for _, e := range plan.Updates {
		applied = append(applied, e.ID())
	}
	return ApplyResult{Applied: applied}, nil
}

type fakeAdapter struct {
	pullResult storage.PullResult
	pushReq    storage.PushRequest
	pushResult storage.PushResult
}

func (f *fakeAdapter) Name() string                       { return "fake" }
func (f *fakeAdapter) Capabilities() storage.Capabilities { return storage.Capabilities{} }
func (f *fakeAdapter) Pull(_ context.Context, _ storage.PullRequest) (storage.PullResult, error) {
	return f.pullResult, nil
}

func (f *fakeAdapter) Push(_ context.Context, req storage.PushRequest) (storage.PushResult, error) {
	f.pushReq = req
	return f.pushResult, nil
}

func TestPullUpdatesDraftFromCuratedRemote(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour brouillon",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour",
				}},
			},
		},
	}

	report, err := svc.Pull(context.Background(), PullInput{
		Adapter: adapter,
		Local:   local,
		Options: PullOptions{
			DryRun:                false,
			ApplyCuratedOverDraft: true,
		},
	})
	if err != nil {
		t.Fatalf("pull sync: %v", err)
	}
	if got := len(report.Updates); got != 1 {
		t.Fatalf("expected 1 update, got %d", got)
	}
	if got := len(local.applied.Updates); got != 1 {
		t.Fatalf("expected local apply 1 update, got %d", got)
	}
	if got := local.applied.Updates[0].Provenance.State; got != storage.StateCurated {
		t.Fatalf("expected curated state, got %q", got)
	}
}

func TestPullConflictForCuratedLocalMismatch(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginHuman,
					State:  storage.StateCurated,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{Key: "hello", Locale: "fr", Value: "bonjour remote"}},
			},
		},
	}

	report, err := svc.Pull(context.Background(), PullInput{
		Adapter: adapter,
		Local:   local,
		Options: PullOptions{
			DryRun:                true,
			ApplyCuratedOverDraft: true,
		},
	})
	if err != nil {
		t.Fatalf("pull sync dry-run: %v", err)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
}

func TestPushSkipsDraftAgainstCuratedRemote(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local draft",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour curated",
					Provenance: storage.EntryProvenance{
						Origin: storage.OriginHuman,
						State:  storage.StateCurated,
					},
				}},
			},
		},
	}

	report, err := svc.Push(context.Background(), PushInput{
		Adapter: adapter,
		Local:   local,
		Options: PushOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("push sync dry-run: %v", err)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
	if got := len(adapter.pushReq.Entries); got != 0 {
		t.Fatalf("expected no pushed entries, got %d", got)
	}
}

func TestPushUpdatesRemoteWhenMismatchIsSafe(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local curated",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginHuman,
					State:  storage.StateCurated,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour remote draft",
					Provenance: storage.EntryProvenance{
						Origin: storage.OriginLLM,
						State:  storage.StateDraft,
					},
				}},
			},
		},
	}

	report, err := svc.Push(context.Background(), PushInput{
		Adapter: adapter,
		Local:   local,
		Options: PushOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("push sync dry-run: %v", err)
	}
	if got := len(report.Updates); got != 1 {
		t.Fatalf("expected 1 update, got %d", got)
	}
	if got := len(report.Conflicts); got != 0 {
		t.Fatalf("expected 0 conflicts, got %d", got)
	}
}

func TestPushForceConflictsAllowsDraftOverwrite(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local draft",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour curated",
					Provenance: storage.EntryProvenance{
						Origin: storage.OriginHuman,
						State:  storage.StateCurated,
					},
				}},
			},
		},
	}

	report, err := svc.Push(context.Background(), PushInput{
		Adapter: adapter,
		Local:   local,
		Options: PushOptions{
			DryRun:         true,
			ForceConflicts: true,
		},
	})
	if err != nil {
		t.Fatalf("push sync dry-run: %v", err)
	}
	if got := len(report.Updates); got != 1 {
		t.Fatalf("expected 1 update, got %d", got)
	}
	if got := len(report.Conflicts); got != 0 {
		t.Fatalf("expected 0 conflicts, got %d", got)
	}
}

func TestPushConflictsOnUnknownProvenanceMismatch(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginUnknown,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour remote",
					Provenance: storage.EntryProvenance{
						Origin: storage.OriginHuman,
						State:  storage.StateCurated,
					},
				}},
			},
		},
	}

	report, err := svc.Push(context.Background(), PushInput{
		Adapter: adapter,
		Local:   local,
		Options: PushOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("push sync dry-run: %v", err)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
	if got := report.Conflicts[0].Reason; got != "missing_provenance_value_mismatch" {
		t.Fatalf("unexpected conflict reason: %q", got)
	}
}

func TestPullDoesNotApplyCuratedOverDraftWhenDisabled(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "hello",
				Locale: "fr",
				Value:  "bonjour local draft",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "hello",
					Locale: "fr",
					Value:  "bonjour curated",
				}},
			},
		},
	}

	report, err := svc.Pull(context.Background(), PullInput{
		Adapter: adapter,
		Local:   local,
		Options: PullOptions{
			DryRun:                true,
			ApplyCuratedOverDraft: false,
		},
	})
	if err != nil {
		t.Fatalf("pull sync dry-run: %v", err)
	}
	if got := len(report.Updates); got != 0 {
		t.Fatalf("expected 0 updates, got %d", got)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
}

func TestPullBlocksPlaceholderRegression(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "greeting",
				Locale: "fr",
				Value:  "Bonjour {{name}}",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "greeting",
					Locale: "fr",
					Value:  "Bonjour",
				}},
			},
		},
	}

	report, err := svc.Pull(context.Background(), PullInput{
		Adapter: adapter,
		Local:   local,
		Options: PullOptions{
			DryRun:                true,
			ApplyCuratedOverDraft: true,
		},
	})
	if err != nil {
		t.Fatalf("pull sync dry-run: %v", err)
	}
	if got := len(report.Updates); got != 0 {
		t.Fatalf("expected no updates, got %d", got)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
	if report.Conflicts[0].Reason != conflictReasonInvariantViolation {
		t.Fatalf("unexpected conflict reason: %q", report.Conflicts[0].Reason)
	}
	if len(report.Warnings) == 0 || !strings.Contains(report.Warnings[0].Message, "placeholder parity mismatch") {
		t.Fatalf("expected placeholder parity warning, got %+v", report.Warnings)
	}
}

func TestPullBlocksICUParityRegression(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "cart.items",
				Locale: "fr",
				Value:  "{count, plural, one {# article} other {# articles}}",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "cart.items",
					Locale: "fr",
					Value:  "{count, plural, one {# article}}",
				}},
			},
		},
	}

	report, err := svc.Pull(context.Background(), PullInput{
		Adapter: adapter,
		Local:   local,
		Options: PullOptions{
			DryRun:                true,
			ApplyCuratedOverDraft: true,
		},
	})
	if err != nil {
		t.Fatalf("pull sync dry-run: %v", err)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
	if len(report.Warnings) == 0 || !strings.Contains(report.Warnings[0].Message, "ICU parity mismatch") {
		t.Fatalf("expected ICU parity warning, got %+v", report.Warnings)
	}
}

func TestPushBlocksUnbalancedICURegression(t *testing.T) {
	svc := New()
	local := &fakeLocalStore{
		readSnapshot: storage.CatalogSnapshot{
			Entries: []storage.Entry{{
				Key:    "cart.items",
				Locale: "fr",
				Value:  "{count, plural, one {# article} other {# articles}",
			}},
		},
	}
	adapter := &fakeAdapter{
		pullResult: storage.PullResult{
			Snapshot: storage.CatalogSnapshot{
				Entries: []storage.Entry{{
					Key:    "cart.items",
					Locale: "fr",
					Value:  "{count, plural, one {# article} other {# articles}}",
				}},
			},
		},
	}

	report, err := svc.Push(context.Background(), PushInput{
		Adapter: adapter,
		Local:   local,
		Options: PushOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("push sync dry-run: %v", err)
	}
	if got := len(report.Conflicts); got != 1 {
		t.Fatalf("expected 1 conflict, got %d", got)
	}
	if got := len(adapter.pushReq.Entries); got != 0 {
		t.Fatalf("expected no pushed entries, got %d", got)
	}
	if len(report.Warnings) == 0 || !strings.Contains(report.Warnings[0].Message, "invalid ICU/braces structure") {
		t.Fatalf("expected invalid ICU structure warning, got %+v", report.Warnings)
	}
}

func TestBuildPushSnapshotSortedReportUsesCompareEntryID(t *testing.T) {
	report, _ := buildPushReport(
		storage.CatalogSnapshot{
			Entries: []storage.Entry{
				{Key: "b", Locale: "fr", Value: "2"},
				{Key: "a", Locale: "de", Value: "1"},
				{Key: "a", Locale: "fr", Value: "1"},
			},
		},
		storage.CatalogSnapshot{},
		PushOptions{},
	)

	got := make([]storage.EntryID, 0, len(report.Creates))
	for _, e := range report.Creates {
		got = append(got, e.ID())
	}
	want := []storage.EntryID{
		{Key: "a", Locale: "de"},
		{Key: "a", Locale: "fr"},
		{Key: "b", Locale: "fr"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected sorted create order: got %+v want %+v", got, want)
	}
}

func TestCompareEntryIDOrdersLocaleThenKeyThenContext(t *testing.T) {
	tests := []struct {
		name string
		a    storage.EntryID
		b    storage.EntryID
		want int // sign only
	}{
		{
			name: "locale first",
			a:    storage.EntryID{Locale: "de", Key: "x"},
			b:    storage.EntryID{Locale: "fr", Key: "a"},
			want: -1,
		},
		{
			name: "key second",
			a:    storage.EntryID{Locale: "fr", Key: "a"},
			b:    storage.EntryID{Locale: "fr", Key: "b"},
			want: -1,
		},
		{
			name: "context third",
			a:    storage.EntryID{Locale: "fr", Key: "a", Context: "1"},
			b:    storage.EntryID{Locale: "fr", Key: "a", Context: "2"},
			want: -1,
		},
		{
			name: "equal",
			a:    storage.EntryID{Locale: "fr", Key: "a", Context: "x"},
			b:    storage.EntryID{Locale: "fr", Key: "a", Context: "x"},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareEntryID(tt.a, tt.b)
			switch {
			case tt.want == 0 && got != 0:
				t.Fatalf("expected equal ordering, got %d", got)
			case tt.want < 0 && got >= 0:
				t.Fatalf("expected negative ordering, got %d", got)
			case tt.want > 0 && got <= 0:
				t.Fatalf("expected positive ordering, got %d", got)
			}
		})
	}
}

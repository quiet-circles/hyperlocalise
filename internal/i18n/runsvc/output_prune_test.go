package runsvc

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildPlannedTargetKeySet(t *testing.T) {
	planned := []Task{
		{TargetPath: "/tmp/a.json", EntryKey: "hello"},
		{TargetPath: "/tmp/a.json", EntryKey: "bye"},
		{TargetPath: "/tmp/b.json", EntryKey: "hello"},
	}

	got := buildPlannedTargetKeySet(planned)
	if _, ok := got["/tmp/a.json"]["hello"]; !ok {
		t.Fatalf("missing key hello for target a")
	}
	if _, ok := got["/tmp/a.json"]["bye"]; !ok {
		t.Fatalf("missing key bye for target a")
	}
	if _, ok := got["/tmp/b.json"]["hello"]; !ok {
		t.Fatalf("missing key hello for target b")
	}
}

func TestBuildPlannedTargetMetadata(t *testing.T) {
	planned := []Task{
		{TargetPath: "/tmp/a.json", SourcePath: "/tmp/en.json", SourceLocale: "en", TargetLocale: "fr"},
		{TargetPath: "/tmp/a.json", SourcePath: "/tmp/en.json", SourceLocale: "en", TargetLocale: "fr"},
		{TargetPath: "/tmp/b.json", SourcePath: "/tmp/en-2.json", SourceLocale: "en", TargetLocale: "de"},
	}

	got, err := buildPlannedTargetMetadata(planned)
	if err != nil {
		t.Fatalf("build planned target metadata: %v", err)
	}
	if got["/tmp/a.json"].sourcePath != "/tmp/en.json" || got["/tmp/a.json"].targetLocale != "fr" {
		t.Fatalf("unexpected metadata for a: %+v", got["/tmp/a.json"])
	}
	if got["/tmp/b.json"].sourcePath != "/tmp/en-2.json" || got["/tmp/b.json"].targetLocale != "de" {
		t.Fatalf("unexpected metadata for b: %+v", got["/tmp/b.json"])
	}
}

func TestPlanPruneCandidatesSorted(t *testing.T) {
	svc := newTestService()
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case "/tmp/a.json":
			return []byte(`{"b":"B","c":"C"}`), nil
		case "/tmp/b.json":
			return []byte(`{"a":"A","z":"Z"}`), nil
		default:
			return nil, errors.New("unexpected read path")
		}
	}

	keep := map[string]map[string]struct{}{
		"/tmp/b.json": {"z": {}},
		"/tmp/a.json": {"c": {}},
	}

	got, err := svc.planPruneCandidates(keep)
	if err != nil {
		t.Fatalf("plan prune candidates: %v", err)
	}

	want := []PruneCandidate{
		{TargetPath: "/tmp/a.json", EntryKey: "b"},
		{TargetPath: "/tmp/b.json", EntryKey: "a"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestValidatePruneLimit(t *testing.T) {
	tests := []struct {
		name       string
		in         Input
		candidates int
		wantErr    bool
	}{
		{name: "disabled", in: Input{Prune: false}, candidates: 1000, wantErr: false},
		{name: "dry run", in: Input{Prune: true, DryRun: true}, candidates: 1000, wantErr: false},
		{name: "force", in: Input{Prune: true, PruneForce: true}, candidates: 1000, wantErr: false},
		{name: "default limit passes", in: Input{Prune: true}, candidates: defaultPruneLimit, wantErr: false},
		{name: "default limit fails", in: Input{Prune: true}, candidates: defaultPruneLimit + 1, wantErr: true},
		{name: "custom limit passes", in: Input{Prune: true, PruneLimit: 2}, candidates: 2, wantErr: false},
		{name: "custom limit fails", in: Input{Prune: true, PruneLimit: 2}, candidates: 3, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePruneLimit(tc.in, tc.candidates)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

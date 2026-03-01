package fileworkflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubAPI struct {
	startImportFn      func(context.Context, StartRequest) (JobRef, error)
	startExportFn      func(context.Context, StartRequest) (JobRef, error)
	getJobStatusFn     func(context.Context, JobRef) (JobStatus, error)
	downloadArtifactFn func(context.Context, ArtifactRef) ([]byte, error)
}

func (s stubAPI) StartImport(ctx context.Context, req StartRequest) (JobRef, error) {
	return s.startImportFn(ctx, req)
}

func (s stubAPI) StartExport(ctx context.Context, req StartRequest) (JobRef, error) {
	return s.startExportFn(ctx, req)
}

func (s stubAPI) GetJobStatus(ctx context.Context, job JobRef) (JobStatus, error) {
	return s.getJobStatusFn(ctx, job)
}

func (s stubAPI) DownloadArtifact(ctx context.Context, artifact ArtifactRef) ([]byte, error) {
	return s.downloadArtifactFn(ctx, artifact)
}

func TestRunnerRunExportSuccess(t *testing.T) {
	t.Parallel()
	statuses := []JobStatus{
		{State: JobStateQueued},
		{State: JobStateRunning},
		{State: JobStateSuccess, Artifact: &ArtifactRef{ID: "artifact-1"}},
	}
	statusIdx := 0

	runner, err := NewRunner(stubAPI{
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{ID: "job-1"}, nil
		},
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			current := statuses[statusIdx]
			statusIdx++
			return current, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return []byte("zip-data"), nil
		},
	}, Options{
		PollInterval: time.Millisecond,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	result, runErr := runner.RunExport(context.Background(), StartRequest{IdempotencyKey: "abc"})
	if runErr != nil {
		t.Fatalf("run export: %v", runErr)
	}
	if result.Job.ID != "job-1" {
		t.Fatalf("job id = %q, want %q", result.Job.ID, "job-1")
	}
	if string(result.Artifact) != "zip-data" {
		t.Fatalf("artifact = %q, want %q", string(result.Artifact), "zip-data")
	}
}

func TestRunnerRunImportTimeoutNormalized(t *testing.T) {
	t.Parallel()
	runner, err := NewRunner(stubAPI{
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{ID: "job-2"}, nil
		},
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected export")
		},
		getJobStatusFn: func(ctx context.Context, job JobRef) (JobStatus, error) {
			<-ctx.Done()
			return JobStatus{}, ctx.Err()
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return nil, nil
		},
	}, Options{
		Timeout:      10 * time.Millisecond,
		PollInterval: time.Millisecond,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	_, runErr := runner.RunImport(context.Background(), StartRequest{})
	if !IsCode(runErr, ErrorCodeTimeout) {
		t.Fatalf("error = %v, want timeout code", runErr)
	}
}

func TestRunnerRunExportRemoteRejected(t *testing.T) {
	t.Parallel()
	runner, err := NewRunner(stubAPI{
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{ID: "job-3"}, nil
		},
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			return JobStatus{State: JobStateFailed, Message: "invalid locale mapping"}, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return nil, nil
		},
	}, Options{})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	_, runErr := runner.RunExport(context.Background(), StartRequest{})
	if !IsCode(runErr, ErrorCodeRemoteRejected) {
		t.Fatalf("error = %v, want remote_rejected code", runErr)
	}
}

func TestRunnerRunExportPartialSuccess(t *testing.T) {
	t.Parallel()
	runner, err := NewRunner(stubAPI{
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{ID: "job-4"}, nil
		},
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			return JobStatus{State: JobStatePartial, Message: "2 keys skipped", Artifact: &ArtifactRef{ID: "a-4"}}, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return []byte("partial-artifact"), nil
		},
	}, Options{})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	result, runErr := runner.RunExport(context.Background(), StartRequest{})
	if !IsCode(runErr, ErrorCodePartialSuccess) {
		t.Fatalf("error = %v, want partial_success", runErr)
	}
	if string(result.Artifact) != "partial-artifact" {
		t.Fatalf("artifact = %q, want %q", string(result.Artifact), "partial-artifact")
	}
}

func TestRunnerRetriesRetryableErrors(t *testing.T) {
	t.Parallel()
	attempts := 0
	sleeps := 0
	retryableErr := errors.New("retry")

	runner, err := NewRunner(stubAPI{
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			attempts++
			if attempts == 1 {
				return JobRef{}, retryableErr
			}
			return JobRef{ID: "job-5"}, nil
		},
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			return JobStatus{State: JobStateSuccess}, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return nil, nil
		},
	}, Options{
		Retry: RetryConfig{MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: 3 * time.Millisecond},
		IsRetryable: func(err error) bool {
			return errors.Is(err, retryableErr)
		},
		Sleep: func(context.Context, time.Duration) error {
			sleeps++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	_, runErr := runner.RunExport(context.Background(), StartRequest{})
	if runErr != nil {
		t.Fatalf("run export: %v", runErr)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if sleeps == 0 {
		t.Fatalf("expected sleep to be called for retry")
	}
}

func TestRunnerRetryMaxAttemptsCapsTotalCalls(t *testing.T) {
	t.Parallel()
	retryableErr := errors.New("retry")
	attempts := 0

	runner, err := NewRunner(stubAPI{
		startExportFn: func(context.Context, StartRequest) (JobRef, error) {
			attempts++
			return JobRef{}, retryableErr
		},
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			return JobStatus{State: JobStateSuccess}, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return nil, nil
		},
	}, Options{
		Retry: RetryConfig{MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
		IsRetryable: func(err error) bool {
			return errors.Is(err, retryableErr)
		},
		Sleep: func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	_, runErr := runner.RunExport(context.Background(), StartRequest{})
	if runErr == nil {
		t.Fatalf("expected error, got nil")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRunnerPassesIdempotencyKey(t *testing.T) {
	t.Parallel()
	gotKey := ""

	runner, err := NewRunner(stubAPI{
		startImportFn: func(context.Context, StartRequest) (JobRef, error) {
			return JobRef{}, errors.New("unexpected import")
		},
		startExportFn: func(_ context.Context, req StartRequest) (JobRef, error) {
			gotKey = req.IdempotencyKey
			return JobRef{ID: "job-6"}, nil
		},
		getJobStatusFn: func(context.Context, JobRef) (JobStatus, error) {
			return JobStatus{State: JobStateSuccess}, nil
		},
		downloadArtifactFn: func(context.Context, ArtifactRef) ([]byte, error) {
			return nil, nil
		},
	}, Options{})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	_, runErr := runner.RunExport(context.Background(), StartRequest{IdempotencyKey: "idem-key-1"})
	if runErr != nil {
		t.Fatalf("run export: %v", runErr)
	}
	if gotKey != "idem-key-1" {
		t.Fatalf("idempotency key = %q, want %q", gotKey, "idem-key-1")
	}
}

package fileworkflow

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	defaultTimeout      = 2 * time.Minute
	defaultPollInterval = time.Second
	defaultRetryBase    = 200 * time.Millisecond
	defaultRetryMax     = 5 * time.Second
	defaultMultiplier   = 2.0
)

type Operation string

const (
	OperationImport Operation = "import"
	OperationExport Operation = "export"
)

type JobState string

const (
	JobStateQueued   JobState = "queued"
	JobStateRunning  JobState = "running"
	JobStateSuccess  JobState = "success"
	JobStateFailed   JobState = "failed"
	JobStateCanceled JobState = "canceled"
	JobStatePartial  JobState = "partial"
)

func (s JobState) terminal() bool {
	switch s {
	case JobStateSuccess, JobStateFailed, JobStateCanceled, JobStatePartial:
		return true
	default:
		return false
	}
}

type ErrorCode string

const (
	ErrorCodeTimeout        ErrorCode = "timeout"
	ErrorCodeRemoteRejected ErrorCode = "remote_rejected"
	ErrorCodePartialSuccess ErrorCode = "partial_success"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code ErrorCode) bool {
	var wfErr *Error
	if !errors.As(err, &wfErr) {
		return false
	}
	return wfErr.Code == code
}

type StartRequest struct {
	IdempotencyKey string
}

type JobRef struct {
	ID string
}

type ArtifactRef struct {
	ID string
}

type JobStatus struct {
	State    JobState
	Message  string
	Artifact *ArtifactRef
}

type RunResult struct {
	Operation Operation
	Job       JobRef
	Status    JobStatus
	Artifact  []byte
}

type Starter interface {
	StartImport(ctx context.Context, req StartRequest) (JobRef, error)
	StartExport(ctx context.Context, req StartRequest) (JobRef, error)
}

type JobPoller interface {
	GetJobStatus(ctx context.Context, job JobRef) (JobStatus, error)
}

type ArtifactDownloader interface {
	DownloadArtifact(ctx context.Context, artifact ArtifactRef) ([]byte, error)
}

type API interface {
	Starter
	JobPoller
	ArtifactDownloader
}

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

func (r RetryConfig) normalize() RetryConfig {
	if r.MaxAttempts < 0 {
		r.MaxAttempts = 0
	}
	if r.InitialDelay <= 0 {
		r.InitialDelay = defaultRetryBase
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = defaultRetryMax
	}
	if r.Multiplier < 1 {
		r.Multiplier = defaultMultiplier
	}
	return r
}

type Options struct {
	Timeout      time.Duration
	PollInterval time.Duration
	Retry        RetryConfig
	IsRetryable  func(error) bool
	Sleep        func(ctx context.Context, d time.Duration) error
}

func (o Options) normalize() Options {
	if o.Timeout <= 0 {
		o.Timeout = defaultTimeout
	}
	if o.PollInterval <= 0 {
		o.PollInterval = defaultPollInterval
	}
	o.Retry = o.Retry.normalize()
	if o.IsRetryable == nil {
		o.IsRetryable = func(error) bool { return false }
	}
	if o.Sleep == nil {
		o.Sleep = sleepWithContext
	}
	return o
}

type Runner struct {
	api  API
	opts Options
}

func NewRunner(api API, opts Options) (*Runner, error) {
	if api == nil {
		return nil, fmt.Errorf("file workflow: api must not be nil")
	}
	return &Runner{api: api, opts: opts.normalize()}, nil
}

func (r *Runner) RunImport(ctx context.Context, req StartRequest) (RunResult, error) {
	return r.run(ctx, OperationImport, req)
}

func (r *Runner) RunExport(ctx context.Context, req StartRequest) (RunResult, error) {
	return r.run(ctx, OperationExport, req)
}

func (r *Runner) run(ctx context.Context, op Operation, req StartRequest) (RunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.opts.Timeout)
	defer cancel()

	job, err := withRetry(ctx, r.opts, func(callCtx context.Context) (JobRef, error) {
		switch op {
		case OperationImport:
			return r.api.StartImport(callCtx, req)
		case OperationExport:
			return r.api.StartExport(callCtx, req)
		default:
			return JobRef{}, fmt.Errorf("unsupported operation %q", op)
		}
	})
	if err != nil {
		return RunResult{}, normalizeError(fmt.Errorf("start %s job: %w", op, err))
	}

	status, err := r.waitForTerminal(ctx, job)
	if err != nil {
		return RunResult{Operation: op, Job: job}, err
	}

	result := RunResult{Operation: op, Job: job, Status: status}
	if status.Artifact != nil {
		artifact, downloadErr := withRetry(ctx, r.opts, func(callCtx context.Context) ([]byte, error) {
			return r.api.DownloadArtifact(callCtx, *status.Artifact)
		})
		if downloadErr != nil {
			return result, normalizeError(fmt.Errorf("download artifact for job %q: %w", job.ID, downloadErr))
		}
		result.Artifact = artifact
	}

	switch status.State {
	case JobStateSuccess:
		return result, nil
	case JobStatePartial:
		return result, &Error{Code: ErrorCodePartialSuccess, Message: status.Message}
	default:
		return result, &Error{Code: ErrorCodeRemoteRejected, Message: status.Message}
	}
}

func (r *Runner) waitForTerminal(ctx context.Context, job JobRef) (JobStatus, error) {
	for {
		status, err := withRetry(ctx, r.opts, func(callCtx context.Context) (JobStatus, error) {
			return r.api.GetJobStatus(callCtx, job)
		})
		if err != nil {
			return JobStatus{}, normalizeError(fmt.Errorf("poll job %q status: %w", job.ID, err))
		}
		if status.State.terminal() {
			return status, nil
		}
		if err := r.opts.Sleep(ctx, r.opts.PollInterval); err != nil {
			return JobStatus{}, normalizeError(fmt.Errorf("poll job %q status: %w", job.ID, err))
		}
	}
}

func withRetry[T any](ctx context.Context, opts Options, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	attempt := 0
	for {
		value, err := fn(ctx)
		if err == nil {
			return value, nil
		}
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if !opts.IsRetryable(err) {
			return zero, err
		}
		if opts.Retry.MaxAttempts > 0 && attempt+1 >= opts.Retry.MaxAttempts {
			return zero, err
		}
		delay := retryDelay(opts.Retry, attempt)
		attempt++
		if sleepErr := opts.Sleep(ctx, delay); sleepErr != nil {
			return zero, sleepErr
		}
	}
}

func retryDelay(cfg RetryConfig, attempt int) time.Duration {
	factor := math.Pow(cfg.Multiplier, float64(attempt))
	delay := time.Duration(float64(cfg.InitialDelay) * factor)
	if delay > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return delay
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizeError(err error) error {
	if err == nil {
		return nil
	}
	if IsCode(err, ErrorCodeTimeout) || IsCode(err, ErrorCodeRemoteRejected) || IsCode(err, ErrorCodePartialSuccess) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: ErrorCodeTimeout, Message: err.Error(), Cause: err}
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return err
}

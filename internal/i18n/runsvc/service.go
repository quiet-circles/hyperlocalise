package runsvc

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	tokenSource = "{{source}}"
	tokenTarget = "{{target}}"
	tokenInput  = "{{input}}"
)

type Input struct {
	ConfigPath string
	DryRun     bool
	Prune      bool
	PruneLimit int
	PruneForce bool
	LockPath   string
}

const defaultPruneLimit = 100

type Task struct {
	SourceLocale string `json:"sourceLocale"`
	TargetLocale string `json:"targetLocale"`
	SourcePath   string `json:"sourcePath"`
	TargetPath   string `json:"targetPath"`
	EntryKey     string `json:"entryKey"`
	SourceText   string `json:"sourceText"`
	ProfileName  string `json:"profileName"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
}

type Failure struct {
	TargetPath string `json:"targetPath"`
	EntryKey   string `json:"entryKey"`
	Reason     string `json:"reason"`
}

type Report struct {
	PlannedTotal    int              `json:"plannedTotal"`
	SkippedByLock   int              `json:"skippedByLock"`
	ExecutableTotal int              `json:"executableTotal"`
	Succeeded       int              `json:"succeeded"`
	Failed          int              `json:"failed"`
	PersistedToLock int              `json:"persistedToLock"`
	Failures        []Failure        `json:"failures,omitempty"`
	Executable      []Task           `json:"executable,omitempty"`
	Skipped         []Task           `json:"skipped,omitempty"`
	PruneCandidates []PruneCandidate `json:"pruneCandidates,omitempty"`
	PruneApplied    int              `json:"pruneApplied"`
}

type PruneCandidate struct {
	TargetPath string `json:"targetPath"`
	EntryKey   string `json:"entryKey"`
}

type Service struct {
	loadConfig func(path string) (*config.I18NConfig, error)
	loadLock   func(path string) (*lockfile.File, error)
	saveLock   func(path string, f lockfile.File) error
	readFile   func(path string) ([]byte, error)
	writeFile  func(path string, content []byte) error
	translate  func(ctx context.Context, req translator.Request) (string, error)
	newParser  func() *translationfileparser.Strategy
	now        func() time.Time
	numCPU     func() int
}

func New() *Service {
	return &Service{
		loadConfig: config.Load,
		loadLock:   lockfile.Load,
		saveLock:   lockfile.Save,
		readFile:   os.ReadFile,
		writeFile: func(path string, content []byte) error {
			return writeBytesAtomic(path, content)
		},
		translate: translator.Translate,
		newParser: translationfileparser.NewDefaultStrategy,
		now:       func() time.Time { return time.Now().UTC() },
		numCPU:    runtime.NumCPU,
	}
}

func Run(ctx context.Context, in Input) (Report, error) {
	return New().Run(ctx, in)
}

func (s *Service) Run(ctx context.Context, in Input) (Report, error) {
	cfg, err := s.loadConfig(in.ConfigPath)
	if err != nil {
		return Report{}, fmt.Errorf("load config: %w", err)
	}

	planned, err := s.planTasks(cfg)
	if err != nil {
		return Report{}, err
	}

	state, err := s.loadLock(in.LockPath)
	if err != nil {
		return Report{}, fmt.Errorf("load lock state: %w", err)
	}

	if state.RunCompleted == nil {
		state.RunCompleted = map[string]lockfile.RunCompletion{}
	}

	report, executable := applyLockFilter(planned, state.RunCompleted)

	pruneTargets := map[string]map[string]struct{}{}
	if in.Prune {
		pruneTargets = buildPlannedTargetKeySet(planned)
		report.PruneCandidates, err = s.planPruneCandidates(pruneTargets)
		if err != nil {
			return report, err
		}
		if err := validatePruneLimit(in, len(report.PruneCandidates)); err != nil {
			return report, err
		}
	}

	if in.DryRun {
		return report, nil
	}
	if len(executable) == 0 && len(report.PruneCandidates) == 0 {
		return report, nil
	}

	staged, execReport, err := s.executePool(ctx, executable, in.LockPath, state)
	report.Succeeded = execReport.Succeeded
	report.Failed = execReport.Failed
	report.PersistedToLock = execReport.PersistedToLock
	report.Failures = append(report.Failures, execReport.Failures...)
	if err != nil {
		return report, err
	}

	if err := s.flushOutputs(staged, pruneTargets); err != nil {
		return report, err
	}
	report.PruneApplied = len(report.PruneCandidates)

	return report, nil
}

func (s *Service) planTasks(cfg *config.I18NConfig) ([]Task, error) {
	parser := s.newParser()
	groups := sortedGroupNames(cfg.Groups)
	buckets := sortedBucketNames(cfg.Buckets)

	tasks := make([]Task, 0)

	for _, groupName := range groups {
		group := cfg.Groups[groupName]
		profileName, profile, err := resolveProfile(cfg, groupName)
		if err != nil {
			return nil, err
		}

		targets := group.Targets
		if len(targets) == 0 {
			targets = append([]string(nil), cfg.Locales.Targets...)
		}
		slices.Sort(targets)

		selectedBuckets := group.Buckets
		if len(selectedBuckets) == 0 {
			selectedBuckets = append([]string(nil), buckets...)
		}

		for _, bucketName := range selectedBuckets {
			bucket, ok := cfg.Buckets[bucketName]
			if !ok {
				return nil, fmt.Errorf("planning tasks: group %q references unknown bucket %q", groupName, bucketName)
			}

			for _, file := range bucket.Files {
				sourcePath := expandLocalePath(file.From, cfg.Locales.Source, "")
				sourceEntries, err := s.loadSourceEntries(parser, sourcePath)
				if err != nil {
					return nil, err
				}
				keys := sortedEntryKeys(sourceEntries)
				for _, target := range targets {
					targetPath := expandLocalePath(file.To, cfg.Locales.Source, target)
					for _, key := range keys {
						sourceText := sourceEntries[key]
						tasks = append(tasks, Task{
							SourceLocale: cfg.Locales.Source,
							TargetLocale: target,
							SourcePath:   sourcePath,
							TargetPath:   targetPath,
							EntryKey:     key,
							SourceText:   sourceText,
							ProfileName:  profileName,
							Provider:     profile.Provider,
							Model:        profile.Model,
							Prompt:       renderPrompt(profile.Prompt, cfg.Locales.Source, target, sourceText),
						})
					}
				}
			}
		}
	}

	return tasks, nil
}

func (s *Service) loadSourceEntries(parser *translationfileparser.Strategy, sourcePath string) (map[string]string, error) {
	content, err := s.readFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("planning tasks: source file %q does not exist", sourcePath)
		}
		return nil, fmt.Errorf("planning tasks: read source file %q: %w", sourcePath, err)
	}

	entries, err := parser.Parse(sourcePath, content)
	if err != nil {
		return nil, fmt.Errorf("planning tasks: parse source file %q: %w", sourcePath, err)
	}

	return entries, nil
}

func applyLockFilter(planned []Task, completed map[string]lockfile.RunCompletion) (Report, []Task) {
	report := Report{PlannedTotal: len(planned)}

	executable := make([]Task, 0, len(planned))
	for _, task := range planned {
		identity := taskIdentity(task.TargetPath, task.EntryKey)
		sourceHash := hashSourceText(task.SourceText)
		if c, ok := completed[identity]; ok && c.SourceHash == sourceHash {
			report.SkippedByLock++
			report.Skipped = append(report.Skipped, task)
			continue
		}

		report.Executable = append(report.Executable, task)
		executable = append(executable, task)
	}

	report.ExecutableTotal = len(executable)
	return report, executable
}

type executionReport struct {
	Succeeded       int
	Failed          int
	PersistedToLock int
	Failures        []Failure
}

type taskCompletion struct {
	identity   string
	sourceHash string
}

type stagedOutput struct {
	entries    map[string]string
	sourcePath string
}

func (s *Service) executePool(ctx context.Context, tasks []Task, lockPath string, state *lockfile.File) (map[string]stagedOutput, executionReport, error) {
	report := executionReport{}
	staged := map[string]stagedOutput{}

	workerCount := s.numCPU()
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan Task)
	completions := make(chan taskCompletion)
	fatalLockErr := make(chan error, 1)

	var (
		reportMu sync.Mutex
		stageMu  sync.Mutex
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lockWriterDone := make(chan struct{})
	go s.runLockWriter(completions, lockWriterDone, state, lockPath, fatalLockErr, cancel, &report, &reportMu)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go s.runWorker(ctx, jobs, completions, staged, &stageMu, &report, &reportMu, &wg)
	}

	go s.feedJobs(ctx, jobs, tasks)

	wg.Wait()
	close(completions)
	<-lockWriterDone

	select {
	case err := <-fatalLockErr:
		return staged, report, err
	default:
	}

	return staged, report, nil
}

func (s *Service) runLockWriter(
	completions <-chan taskCompletion,
	lockWriterDone chan<- struct{},
	state *lockfile.File,
	lockPath string,
	fatalLockErr chan<- error,
	cancel context.CancelFunc,
	report *executionReport,
	reportMu *sync.Mutex,
) {
	defer close(lockWriterDone)
	for completion := range completions {
		state.RunCompleted[completion.identity] = lockfile.RunCompletion{CompletedAt: s.now(), SourceHash: completion.sourceHash}
		if err := s.saveLock(lockPath, *state); err != nil {
			select {
			case fatalLockErr <- fmt.Errorf("persist lock state: %w", err):
			default:
			}
			cancel()
			return
		}
		reportMu.Lock()
		report.PersistedToLock++
		reportMu.Unlock()
	}
}

func (s *Service) runWorker(
	ctx context.Context,
	jobs <-chan Task,
	completions chan<- taskCompletion,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	report *executionReport,
	reportMu *sync.Mutex,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-jobs:
			if !ok {
				return
			}
			s.processTask(ctx, task, staged, stageMu, completions, report, reportMu)
		}
	}
}

func (s *Service) processTask(
	ctx context.Context,
	task Task,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	completions chan<- taskCompletion,
	report *executionReport,
	reportMu *sync.Mutex,
) {
	translated, err := s.translate(ctx, translator.Request{
		Source:         task.SourceText,
		TargetLanguage: task.TargetLocale,
		Context:        task.EntryKey,
		ModelProvider:  task.Provider,
		Model:          task.Model,
		Prompt:         task.Prompt,
	})
	if err != nil {
		recordTaskFailure(report, reportMu, task, err)
		return
	}

	if err := stageTaskOutput(staged, task.TargetPath, task.SourcePath, task.EntryKey, translated, stageMu); err != nil {
		recordTaskFailure(report, reportMu, task, err)
		return
	}

	select {
	case completions <- taskCompletion{identity: taskIdentity(task.TargetPath, task.EntryKey), sourceHash: hashSourceText(task.SourceText)}:
		reportMu.Lock()
		report.Succeeded++
		reportMu.Unlock()
	case <-ctx.Done():
		return
	}
}

func (s *Service) feedJobs(ctx context.Context, jobs chan<- Task, tasks []Task) {
	defer close(jobs)
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		case jobs <- task:
		}
	}
}

func recordTaskFailure(report *executionReport, reportMu *sync.Mutex, task Task, err error) {
	reportMu.Lock()
	defer reportMu.Unlock()
	report.Failed++
	report.Failures = append(report.Failures, Failure{TargetPath: task.TargetPath, EntryKey: task.EntryKey, Reason: err.Error()})
}

func stageTaskOutput(staged map[string]stagedOutput, targetPath, sourcePath, entryKey, value string, stageMu *sync.Mutex) error {
	stageMu.Lock()
	defer stageMu.Unlock()

	bucket, ok := staged[targetPath]
	if !ok {
		bucket = stagedOutput{entries: map[string]string{}, sourcePath: sourcePath}
		staged[targetPath] = bucket
	} else if bucket.sourcePath != sourcePath {
		return fmt.Errorf("output staging conflict: %s has conflicting source paths", targetPath)
	}

	if existing, exists := bucket.entries[entryKey]; exists && existing != value {
		return fmt.Errorf("output staging conflict: %s already staged with different value", taskIdentity(targetPath, entryKey))
	}

	bucket.entries[entryKey] = value
	staged[targetPath] = bucket
	return nil
}

func (s *Service) flushOutputs(staged map[string]stagedOutput, pruneTargets map[string]map[string]struct{}) error {
	targetPaths := make([]string, 0, len(staged))
	for path := range staged {
		targetPaths = append(targetPaths, path)
	}
	for path := range pruneTargets {
		if _, ok := staged[path]; ok {
			continue
		}
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)
	targetPaths = slices.Compact(targetPaths)

	for _, targetPath := range targetPaths {
		output := staged[targetPath]
		values, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return err
		}

		if keep := pruneTargets[targetPath]; keep != nil {
			for key := range values {
				if _, ok := keep[key]; ok {
					continue
				}
				delete(values, key)
			}
		}

		maps.Copy(values, output.entries)

		content, err := s.marshalTargetFile(targetPath, output.sourcePath, values)
		if err != nil {
			return err
		}
		if err := s.writeFile(targetPath, content); err != nil {
			return fmt.Errorf("flush outputs: write %q: %w", targetPath, err)
		}
	}

	return nil
}

func buildPlannedTargetKeySet(planned []Task) map[string]map[string]struct{} {
	keep := map[string]map[string]struct{}{}
	for _, task := range planned {
		bucket := keep[task.TargetPath]
		if bucket == nil {
			bucket = map[string]struct{}{}
			keep[task.TargetPath] = bucket
		}
		bucket[task.EntryKey] = struct{}{}
	}
	return keep
}

func (s *Service) planPruneCandidates(pruneTargets map[string]map[string]struct{}) ([]PruneCandidate, error) {
	candidates := make([]PruneCandidate, 0)
	targetPaths := make([]string, 0, len(pruneTargets))
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)

	for _, targetPath := range targetPaths {
		existing, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return nil, err
		}
		keys := sortedEntryKeys(existing)
		for _, key := range keys {
			if _, ok := pruneTargets[targetPath][key]; ok {
				continue
			}
			candidates = append(candidates, PruneCandidate{TargetPath: targetPath, EntryKey: key})
		}
	}

	return candidates, nil
}

func validatePruneLimit(in Input, candidates int) error {
	if !in.Prune || in.DryRun || in.PruneForce {
		return nil
	}
	limit := in.PruneLimit
	if limit <= 0 {
		limit = defaultPruneLimit
	}
	if candidates <= limit {
		return nil
	}
	return fmt.Errorf("prune safety limit exceeded: %d keys scheduled for deletion (limit %d). rerun with --prune-max-deletions %d or --prune-force", candidates, limit, candidates)
}

func (s *Service) loadExistingTarget(path string) (map[string]string, error) {
	content, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}

	entries, err := s.newParser().Parse(path, content)
	if err != nil {
		return nil, fmt.Errorf("flush outputs: parse target file %q: %w", path, err)
	}

	return entries, nil
}

func (s *Service) marshalTargetFile(path, sourcePath string, values map[string]string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".mdx" {
		template, err := s.loadMarkdownTemplate(path, sourcePath)
		if err != nil {
			return nil, err
		}
		return translationfileparser.MarshalMarkdown(template, values), nil
	}

	if ext != ".json" {
		return nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}

	payload := map[string]any{}
	keys := sortedEntryKeys(values)
	for _, key := range keys {
		setNestedValue(payload, key, values[key])
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
	}
	return append(content, '\n'), nil
}

func (s *Service) loadMarkdownTemplate(targetPath, sourcePath string) ([]byte, error) {
	content, err := s.readFile(targetPath)
	if err == nil {
		return content, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", targetPath, err)
	}

	template, srcErr := s.readFile(sourcePath)
	if srcErr != nil {
		return nil, fmt.Errorf("flush outputs: read markdown template source %q: %w", sourcePath, srcErr)
	}
	return template, nil
}

func setNestedValue(payload map[string]any, dottedKey, value string) {
	parts := strings.Split(dottedKey, ".")
	current := payload
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		next, ok := current[part]
		if !ok {
			nested := map[string]any{}
			current[part] = nested
			current = nested
			continue
		}

		nested, ok := next.(map[string]any)
		if !ok {
			nested = map[string]any{}
			current[part] = nested
		}
		current = nested
	}
}

func writeBytesAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}

func resolveProfile(cfg *config.I18NConfig, groupName string) (string, config.LLMProfile, error) {
	bestPriority := -1
	bestProfile := ""

	for _, rule := range cfg.LLM.Rules {
		if rule.Group != groupName {
			continue
		}
		if rule.Priority > bestPriority {
			bestPriority = rule.Priority
			bestProfile = rule.Profile
		}
	}

	if strings.TrimSpace(bestProfile) == "" {
		bestProfile = "default"
	}

	profile, ok := cfg.LLM.Profiles[bestProfile]
	if !ok {
		return "", config.LLMProfile{}, fmt.Errorf("planning tasks: unresolvable profile %q for group %q", bestProfile, groupName)
	}

	return bestProfile, profile, nil
}

func sortedGroupNames(groups map[string]config.GroupConfig) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedBucketNames(buckets map[string]config.BucketConfig) []string {
	names := make([]string, 0, len(buckets))
	for name := range buckets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedEntryKeys(entries map[string]string) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func expandLocalePath(pattern, sourceLocale, targetLocale string) string {
	path := strings.ReplaceAll(pattern, tokenSource, sourceLocale)
	path = strings.ReplaceAll(path, tokenTarget, targetLocale)
	return path
}

func renderPrompt(prompt, sourceLocale, targetLocale, sourceText string) string {
	rendered := strings.ReplaceAll(prompt, tokenSource, sourceLocale)
	rendered = strings.ReplaceAll(rendered, tokenTarget, targetLocale)
	rendered = strings.ReplaceAll(rendered, tokenInput, sourceText)
	return rendered
}

func taskIdentity(targetPath, entryKey string) string {
	return targetPath + "::" + entryKey
}

func hashSourceText(source string) string {
	sum := sha512.Sum512([]byte(source))
	return fmt.Sprintf("%x", sum)
}

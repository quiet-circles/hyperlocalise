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
	LockPath   string
}

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
	PlannedTotal    int       `json:"plannedTotal"`
	SkippedByLock   int       `json:"skippedByLock"`
	ExecutableTotal int       `json:"executableTotal"`
	Succeeded       int       `json:"succeeded"`
	Failed          int       `json:"failed"`
	PersistedToLock int       `json:"persistedToLock"`
	Failures        []Failure `json:"failures,omitempty"`
	Executable      []Task    `json:"executable,omitempty"`
	Skipped         []Task    `json:"skipped,omitempty"`
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

	if in.DryRun || len(executable) == 0 {
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

	if err := s.flushOutputs(staged); err != nil {
		return report, err
	}

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

func (s *Service) executePool(ctx context.Context, tasks []Task, lockPath string, state *lockfile.File) (map[string]map[string]string, executionReport, error) {
	report := executionReport{}
	staged := map[string]map[string]string{}

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
	go func() {
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
	}()

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-jobs:
					if !ok {
						return
					}

					translated, err := s.translate(ctx, translator.Request{
						Source:         task.SourceText,
						TargetLanguage: task.TargetLocale,
						Context:        task.EntryKey,
						ModelProvider:  task.Provider,
						Model:          task.Model,
						Prompt:         task.Prompt,
					})
					if err != nil {
						reportMu.Lock()
						report.Failed++
						report.Failures = append(report.Failures, Failure{
							TargetPath: task.TargetPath,
							EntryKey:   task.EntryKey,
							Reason:     err.Error(),
						})
						reportMu.Unlock()
						continue
					}

					if err := stageTaskOutput(staged, task.TargetPath, task.EntryKey, translated, &stageMu); err != nil {
						reportMu.Lock()
						report.Failed++
						report.Failures = append(report.Failures, Failure{
							TargetPath: task.TargetPath,
							EntryKey:   task.EntryKey,
							Reason:     err.Error(),
						})
						reportMu.Unlock()
						continue
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
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- task:
			}
		}
	}()

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

func stageTaskOutput(staged map[string]map[string]string, targetPath, entryKey, value string, stageMu *sync.Mutex) error {
	stageMu.Lock()
	defer stageMu.Unlock()

	bucket := staged[targetPath]
	if bucket == nil {
		bucket = map[string]string{}
		staged[targetPath] = bucket
	}

	if existing, exists := bucket[entryKey]; exists && existing != value {
		return fmt.Errorf("output staging conflict: %s already staged with different value", taskIdentity(targetPath, entryKey))
	}

	bucket[entryKey] = value
	return nil
}

func (s *Service) flushOutputs(staged map[string]map[string]string) error {
	targetPaths := make([]string, 0, len(staged))
	for path := range staged {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)

	for _, targetPath := range targetPaths {
		values, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return err
		}

		maps.Copy(values, staged[targetPath])

		content, err := marshalTargetFile(targetPath, values)
		if err != nil {
			return err
		}
		if err := s.writeFile(targetPath, content); err != nil {
			return fmt.Errorf("flush outputs: write %q: %w", targetPath, err)
		}
	}

	return nil
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

func marshalTargetFile(path string, values map[string]string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
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

package runsvc

import (
	"context"
	"crypto/sha512"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/pathresolver"
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
	Workers    int
	OnEvent    func(Event)
}

const defaultPruneLimit = 100

type EventKind string

const (
	EventPhase     EventKind = "phase"
	EventPlanned   EventKind = "planned"
	EventTaskDone  EventKind = "task_done"
	EventPersisted EventKind = "persisted"
	EventCompleted EventKind = "completed"
)

const (
	PhasePlanning         = "planning"
	PhaseScanningPrune    = "scanning_prune"
	PhaseExecuting        = "executing"
	PhaseFinalizingOutput = "finalizing_output"
)

type Event struct {
	Kind            EventKind `json:"kind"`
	Phase           string    `json:"phase,omitempty"`
	PlannedTotal    int       `json:"plannedTotal,omitempty"`
	SkippedByLock   int       `json:"skippedByLock,omitempty"`
	ExecutableTotal int       `json:"executableTotal,omitempty"`
	Succeeded       int       `json:"succeeded,omitempty"`
	Failed          int       `json:"failed,omitempty"`
	PersistedToLock int       `json:"persistedToLock,omitempty"`
	PruneCandidates int       `json:"pruneCandidates,omitempty"`
	PruneApplied    int       `json:"pruneApplied,omitempty"`
	TaskSucceeded   bool      `json:"taskSucceeded,omitempty"`
	TargetPath      string    `json:"targetPath,omitempty"`
	EntryKey        string    `json:"entryKey,omitempty"`
	FailureReason   string    `json:"failureReason,omitempty"`
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
				sourcePattern := pathresolver.ResolveSourcePath(file.From, cfg.Locales.Source)
				sources, err := resolveSourcePaths(sourcePattern)
				if err != nil {
					return nil, fmt.Errorf("planning tasks: resolve source paths for %q: %w", sourcePattern, err)
				}
				if len(sources) == 0 {
					return nil, fmt.Errorf("planning tasks: source pattern %q matched no files", sourcePattern)
				}

				for _, sourcePath := range sources {
					if shouldIgnoreSourcePath(sourcePath, cfg.Locales.Targets) {
						continue
					}
					sourceEntries, err := s.loadSourceEntries(parser, sourcePath)
					if err != nil {
						return nil, err
					}
					keys := sortedEntryKeys(sourceEntries)
					for _, target := range targets {
						resolvedTargetPattern := pathresolver.ResolveTargetPath(file.To, cfg.Locales.Source, target)
						targetPath, err := resolveTargetPath(sourcePattern, resolvedTargetPattern, sourcePath)
						if err != nil {
							return nil, fmt.Errorf("planning tasks: resolve target path for source %q: %w", sourcePath, err)
						}
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

type eventEmitter struct {
	notify func(Event)
	mu     sync.Mutex
}

func newEventEmitter(onEvent func(Event)) *eventEmitter {
	if onEvent == nil {
		return nil
	}

	return &eventEmitter{notify: onEvent}
}

func (e *eventEmitter) emit(ev Event) {
	if e == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.notify(ev)
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

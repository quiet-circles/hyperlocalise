package runsvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	contextMemorySourceMaxChars  = 12000
	contextMemoryPerGroupTimeout = 45 * time.Second
	contextMemorySyntheticKey    = "__context_memory__"
	contextTaskBurstPerKey       = 1
)

type contextMemoryPlan struct {
	Enabled  bool
	Scope    string
	MaxChars int
	Total    int
	Groups   map[string]contextMemoryGroup
}

type contextMemoryGroup struct {
	Key                string
	Source             string
	Seed               Task
	DisplayTarget      string
	SingleTargetLocale string
}

func normalizeContextMemoryOptions(in Input) (scope string, maxChars int, err error) {
	scope = strings.ToLower(strings.TrimSpace(in.ContextMemoryScope))
	if scope == "" {
		scope = ContextMemoryScopeFile
	}
	switch scope {
	case ContextMemoryScopeFile, ContextMemoryScopeBucket, ContextMemoryScopeGroup:
	default:
		return "", 0, fmt.Errorf("invalid context memory scope %q: must be one of %q, %q, %q", scope, ContextMemoryScopeFile, ContextMemoryScopeBucket, ContextMemoryScopeGroup)
	}

	maxChars = in.ContextMemoryMaxChars
	if maxChars <= 0 {
		maxChars = defaultContextMemoryChars
	}

	return scope, maxChars, nil
}

func buildContextMemoryPlan(tasks []Task, scope string, maxChars int) contextMemoryPlan {
	if len(tasks) == 0 {
		return contextMemoryPlan{}
	}
	grouped := make(map[string][]int)
	order := make([]string, 0)
	for i := range tasks {
		key := contextMemoryKey(tasks[i], scope)
		tasks[i].ContextKey = key
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], i)
	}
	slices.Sort(order)

	groups := make(map[string]contextMemoryGroup, len(order))
	for _, key := range order {
		indexes := grouped[key]
		if len(indexes) == 0 {
			continue
		}
		seed := tasks[indexes[0]]
		singleLocale, _ := singleTargetLocale(tasks, indexes)
		groups[key] = contextMemoryGroup{
			Key:                key,
			Source:             buildContextMemorySource(tasks, indexes),
			Seed:               seed,
			DisplayTarget:      contextMemoryDisplayTarget(seed, scope),
			SingleTargetLocale: singleLocale,
		}
	}

	return contextMemoryPlan{
		Enabled:  true,
		Scope:    scope,
		MaxChars: maxChars,
		Total:    len(groups),
		Groups:   groups,
	}
}

func (s *Service) resolveTaskContextMemory(ctx context.Context, task Task, state *executorState, emitter *eventEmitter) string {
	if !state.contextPlan.Enabled {
		return ""
	}
	key := strings.TrimSpace(task.ContextKey)
	if key == "" {
		return ""
	}
	group, ok := state.contextPlan.Groups[key]
	if !ok {
		return ""
	}

	state.contextMu.Lock()
	if slot, exists := state.contextSlots[key]; exists {
		done := slot.done
		state.contextMu.Unlock()
		select {
		case <-done:
			return slot.memory
		case <-ctx.Done():
			return ""
		}
	}
	slot := &contextMemorySlot{done: make(chan struct{})}
	state.contextSlots[key] = slot
	state.contextMu.Unlock()

	s.emitContextMemoryStart(state, emitter, group)
	usage := translator.Usage{}
	groupCtx, cancel := context.WithTimeout(ctx, contextMemoryPerGroupTimeout)
	request := translator.Request{
		Source:         group.Source,
		TargetLanguage: group.Seed.TargetLocale,
		Context:        fmt.Sprintf("Scope: %s\nSource locale: %s\nTarget locale: %s\nSource identifier: %s", state.contextPlan.Scope, group.Seed.SourceLocale, group.Seed.TargetLocale, contextScopeValue(group.Seed, state.contextPlan.Scope)),
		ModelProvider:  group.Seed.Provider,
		Model:          group.Seed.Model,
		Prompt:         buildContextMemoryPrompt(group.Seed.SourceLocale, group.Seed.TargetLocale),
	}
	memory, err := s.translateRequestWithRetry(translator.WithUsageCollector(groupCtx, &usage), request)
	cancel()
	memory = normalizeContextMemory(memory, state.contextPlan.MaxChars)

	var warning, failureReason string
	success := err == nil && memory != ""
	if !success {
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") {
				warning = fmt.Sprintf("context memory generation timed out for %q after %s; continuing without shared memory", key, contextMemoryPerGroupTimeout)
			} else {
				warning = fmt.Sprintf("context memory generation failed for %q: %v", key, err)
			}
			failureReason = "summary request failed"
		} else {
			warning = fmt.Sprintf("context memory generation produced empty summary for %q", key)
			failureReason = "empty summary"
		}
	}

	state.contextMu.Lock()
	slot.memory = memory
	close(slot.done)
	state.contextMu.Unlock()

	state.reportMu.Lock()
	if success {
		state.report.ContextMemoryGenerated++
		usageToken := toRunTokenUsage(usage)
		state.report.TokenUsage = addTokenUsage(state.report.TokenUsage, usageToken)
		if locale := strings.TrimSpace(group.SingleTargetLocale); locale != "" {
			state.report.LocaleUsage[locale] = addTokenUsage(state.report.LocaleUsage[locale], usageToken)
		}
	} else {
		state.report.ContextMemoryFallbackGroups++
		if warning != "" {
			state.report.Warnings = append(state.report.Warnings, warning)
		}
	}
	generated := state.report.ContextMemoryGenerated
	fallbacks := state.report.ContextMemoryFallbackGroups
	state.reportMu.Unlock()

	s.emitContextMemoryDone(state, emitter, group, success, failureReason, generated, fallbacks)
	if success {
		return memory
	}
	return ""
}

func (s *Service) emitContextMemoryStart(state *executorState, emitter *eventEmitter, group contextMemoryGroup) {
	state.reportMu.Lock()
	generated := state.report.ContextMemoryGenerated
	fallbacks := state.report.ContextMemoryFallbackGroups
	total := state.contextPlan.Total
	state.reportMu.Unlock()
	emitter.emit(Event{
		Kind:                   EventContextMemory,
		ContextMemoryState:     ContextMemoryStateStart,
		ContextMemoryTotal:     total,
		ContextMemoryProcessed: generated + fallbacks,
		ContextMemoryFallbacks: fallbacks,
		TargetPath:             group.DisplayTarget,
		EntryKey:               contextMemorySyntheticKey,
		Message:                fmt.Sprintf("building context memory for %s", group.DisplayTarget),
	})
}

func (s *Service) emitContextMemoryDone(state *executorState, emitter *eventEmitter, group contextMemoryGroup, success bool, failureReason string, generated int, fallbacks int) {
	emitter.emit(Event{
		Kind:                   EventContextMemory,
		ContextMemoryState:     ContextMemoryStateProgress,
		ContextMemoryTotal:     state.contextPlan.Total,
		ContextMemoryProcessed: generated + fallbacks,
		ContextMemoryFallbacks: fallbacks,
		TargetPath:             group.DisplayTarget,
		EntryKey:               contextMemorySyntheticKey,
		Message:                fmt.Sprintf("context memory progress for %s", group.DisplayTarget),
	})
	emitter.emit(Event{
		Kind:                   EventContextMemory,
		ContextMemoryState:     ContextMemoryStateDone,
		ContextMemoryTotal:     state.contextPlan.Total,
		ContextMemoryProcessed: generated + fallbacks,
		ContextMemoryFallbacks: fallbacks,
		TaskSucceeded:          success,
		TargetPath:             group.DisplayTarget,
		EntryKey:               contextMemorySyntheticKey,
		FailureReason:          failureReason,
		Message:                fmt.Sprintf("context memory %s for %s", ternary(success, "generated", "fallback"), group.DisplayTarget),
	})
}

func ternary(ok bool, left, right string) string {
	if ok {
		return left
	}
	return right
}

func contextMemoryKey(task Task, scope string) string {
	return fmt.Sprintf("%s|scope_value=%s", scope, contextScopeValue(task, scope))
}

func contextScopeValue(task Task, scope string) string {
	switch scope {
	case ContextMemoryScopeBucket:
		return task.BucketName
	case ContextMemoryScopeGroup:
		return task.GroupName
	default:
		return task.SourcePath
	}
}

func buildContextMemorySource(tasks []Task, indexes []int) string {
	unique := make(map[string]string)
	for _, idx := range indexes {
		task := tasks[idx]
		if _, exists := unique[task.EntryKey]; exists {
			continue
		}
		unique[task.EntryKey] = task.SourceText
	}

	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var b strings.Builder
	b.WriteString("Representative source entries:\n")
	for _, key := range keys {
		text := strings.TrimSpace(unique[key])
		if text == "" {
			continue
		}
		line := fmt.Sprintf("- key=%s\n  text=%s\n", key, strings.ReplaceAll(text, "\n", "\\n"))
		if b.Len()+len(line) > contextMemorySourceMaxChars {
			break
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}

func buildContextMemoryPrompt(sourceLocale, targetLocale string) string {
	return "You produce compact translation memory notes for consistent localization. " +
		"Generate structured plain text under these headings: Terminology, Tone, Formatting, Do-not-translate. " +
		"Do not quote long source passages, do not include secrets, and do not output markdown code fences. " +
		fmt.Sprintf("The source language is %s and target language is %s. ", sourceLocale, targetLocale) +
		"Keep the output concise and directly useful for translating related keys."
}

func normalizeContextMemory(memory string, maxChars int) string {
	trimmed := strings.TrimSpace(memory)
	if trimmed == "" {
		return ""
	}
	if maxChars > 0 {
		runes := []rune(trimmed)
		if len(runes) > maxChars {
			trimmed = strings.TrimSpace(string(runes[:maxChars]))
		}
	}
	return trimmed
}

func contextMemoryDisplayTarget(task Task, scope string) string {
	switch scope {
	case ContextMemoryScopeBucket:
		return "bucket:" + task.BucketName
	case ContextMemoryScopeGroup:
		return "group:" + task.GroupName
	default:
		return task.SourcePath
	}
}

func singleTargetLocale(tasks []Task, indexes []int) (string, bool) {
	if len(indexes) == 0 {
		return "", false
	}
	first := tasks[indexes[0]].TargetLocale
	if strings.TrimSpace(first) == "" {
		return "", false
	}
	for _, idx := range indexes[1:] {
		if tasks[idx].TargetLocale != first {
			return "", false
		}
	}
	return first, true
}

func mergeLocaleUsage(base map[string]TokenUsage, delta map[string]TokenUsage) map[string]TokenUsage {
	if len(delta) == 0 {
		return base
	}
	if base == nil {
		base = make(map[string]TokenUsage, len(delta))
	}
	for locale, usage := range delta {
		base[locale] = addTokenUsage(base[locale], usage)
	}
	return base
}

func interleaveTasksByContextKey(tasks []Task) []Task {
	if len(tasks) <= 1 {
		return tasks
	}

	grouped := make(map[string][]Task)
	order := make([]string, 0)
	for _, task := range tasks {
		key := strings.TrimSpace(task.ContextKey)
		if key == "" {
			key = task.SourcePath
		}
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], task)
	}
	if len(order) <= 1 {
		return tasks
	}
	slices.Sort(order)

	indexes := make(map[string]int, len(order))
	out := make([]Task, 0, len(tasks))
	for len(out) < len(tasks) {
		progressed := false
		for _, key := range order {
			idx := indexes[key]
			queue := grouped[key]
			if idx >= len(queue) {
				continue
			}
			remaining := len(queue) - idx
			take := contextTaskBurstPerKey
			if take > remaining {
				take = remaining
			}
			out = append(out, queue[idx:idx+take]...)
			indexes[key] = idx + take
			progressed = true
		}
		if !progressed {
			break
		}
	}
	if len(out) != len(tasks) {
		return tasks
	}
	return out
}

package runsvc

import "testing"

func baseCacheTask() Task {
	return Task{
		SourceLocale:    "en-US",
		TargetLocale:    "fr-FR",
		SourceText:      "Hello",
		Provider:        "openai",
		Model:           "gpt-5.2",
		ProfileName:     "default",
		PromptVersion:   "p1",
		GlossaryVersion: "g1",
		ParserMode:      "json",
		RAGSnapshot:     "r1",
		ContextKey:      "file:a",
		ContextMemory:   "memory-A",
	}
}

func TestExactCacheKeyChangesWhenSourceLocaleChanges(t *testing.T) {
	base := baseCacheTask()
	other := base
	other.SourceLocale = "en-GB"
	if exactCacheKey(base) == exactCacheKey(other) {
		t.Fatal("expected source locale to affect exact cache key")
	}
}

func TestExactCacheKeyNormalizesSourceText(t *testing.T) {
	base := baseCacheTask()
	other := base
	other.SourceText = "  Hello\r\n"
	if exactCacheKey(base) != exactCacheKey(other) {
		t.Fatal("expected equivalent normalized source text to yield same key")
	}
}

func TestExactCacheKeyChangesWhenContextMemoryChanges(t *testing.T) {
	base := baseCacheTask()
	other := base
	other.ContextMemory = "memory-B"
	if exactCacheKey(base) == exactCacheKey(other) {
		t.Fatal("expected context memory to affect exact cache key")
	}
}

func TestExactCacheKeyChangesAcrossDimensions(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(task *Task)
	}{
		{
			name: "source text",
			mutate: func(task *Task) {
				task.SourceText = "Hello there"
			},
		},
		{
			name: "target locale",
			mutate: func(task *Task) {
				task.TargetLocale = "de-DE"
			},
		},
		{
			name: "provider",
			mutate: func(task *Task) {
				task.Provider = "anthropic"
			},
		},
		{
			name: "model",
			mutate: func(task *Task) {
				task.Model = "claude-sonnet-4"
			},
		},
		{
			name: "profile name",
			mutate: func(task *Task) {
				task.ProfileName = "high-quality"
			},
		},
		{
			name: "prompt version",
			mutate: func(task *Task) {
				task.PromptVersion = "p2"
			},
		},
		{
			name: "glossary version",
			mutate: func(task *Task) {
				task.GlossaryVersion = "g2"
			},
		},
		{
			name: "parser mode",
			mutate: func(task *Task) {
				task.ParserMode = "formatjs"
			},
		},
		{
			name: "context key",
			mutate: func(task *Task) {
				task.ContextKey = "file:b"
			},
		},
		{
			name: "retrieval snapshot",
			mutate: func(task *Task) {
				task.RAGSnapshot = "r2"
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			base := baseCacheTask()
			other := base
			tc.mutate(&other)
			if exactCacheKey(base) == exactCacheKey(other) {
				t.Fatalf("expected %s to affect exact cache key", tc.name)
			}
		})
	}
}

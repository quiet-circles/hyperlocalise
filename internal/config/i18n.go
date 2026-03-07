package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/jsonc"
)

const (
	defaultConfigPath                 = "i18n.jsonc"
	DefaultCacheDBPath                = ".hyperlocalise/cache/cache.sqlite"
	DefaultCacheMaxOpen               = 1
	DefaultCacheMaxIdle               = 1
	DefaultCacheMaxLifeSec            = 300
	DefaultCacheL1MaxItems            = 50000
	DefaultCacheL2AutoAcceptThreshold = 0.90
	DefaultCacheRAGTopK               = 5
	llmProviderOpenAI                 = "openai"
	llmProviderAzureOpenAI            = "azure_openai"
	llmProviderAnthropic              = "anthropic"
	llmProviderLMStudio               = "lmstudio"
	llmProviderGroq                   = "groq"
	llmProviderMistral                = "mistral"
	llmProviderOllama                 = "ollama"
	llmProviderGemini                 = "gemini"
	llmProviderBedrock                = "bedrock"
	llmDefaultProfile                 = "default"
)

// I18NConfig defines the i18n configuration file structure.
type I18NConfig struct {
	Locales LocaleConfig            `json:"locales" jsonschema:"required"`
	Buckets map[string]BucketConfig `json:"buckets" jsonschema:"required"`
	Groups  map[string]GroupConfig  `json:"groups" jsonschema:"required"`
	LLM     LLMConfig               `json:"llm" jsonschema:"required"`
	Storage *StorageConfig          `json:"storage,omitempty"`
	Cache   CacheConfig             `json:"cache,omitempty"`
}

// LocaleConfig configures source/target locales and fallback hierarchy.
type LocaleConfig struct {
	Source    string              `json:"source" jsonschema:"required"`
	Targets   []string            `json:"targets" jsonschema:"required"`
	Fallbacks map[string][]string `json:"fallbacks,omitempty"`
}

// BucketConfig defines file mappings for a bucket.
type BucketConfig struct {
	Files []BucketFileMapping `json:"files" jsonschema:"required"`
}

// BucketFileMapping defines source/target file paths for a bucket.
type BucketFileMapping struct {
	From string `json:"from" jsonschema:"required"`
	To   string `json:"to" jsonschema:"required"`
}

// GroupConfig selects locales and buckets.
type GroupConfig struct {
	Targets []string `json:"targets,omitempty"`
	Buckets []string `json:"buckets,omitempty"`
}

// LLMConfig defines model defaults, locale groups, and override rules.
type LLMConfig struct {
	Profiles      map[string]LLMProfile    `json:"profiles" jsonschema:"required"`
	ContextMemory *LLMContextMemoryProfile `json:"context_memory,omitempty"`
	Rules         []LLMRule                `json:"rules,omitempty"`
}

// LLMProfile contains provider/model prompt configuration.
type LLMProfile struct {
	Provider     string `json:"provider" jsonschema:"required"`
	Model        string `json:"model" jsonschema:"required"`
	Prompt       string `json:"prompt,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	UserPrompt   string `json:"user_prompt,omitempty"`
}

// LLMContextMemoryProfile configures provider/model for context memory building.
type LLMContextMemoryProfile struct {
	Provider string `json:"provider" jsonschema:"required"`
	Model    string `json:"model" jsonschema:"required"`
}

// LLMRule applies a profile for a specific group.
type LLMRule struct {
	Priority int    `json:"priority" jsonschema:"required"`
	Group    string `json:"group" jsonschema:"required"`
	Profile  string `json:"profile" jsonschema:"required"`
}

// StorageConfig configures remote storage adapter sync settings.
type StorageConfig struct {
	Adapter string          `json:"adapter" jsonschema:"required"`
	Config  json.RawMessage `json:"config,omitempty"`
}

// CacheConfig configures local cache foundation (L1/L2/RAG wiring + SQLite bootstrap).
type CacheConfig struct {
	Enabled                        bool                 `json:"enabled,omitempty"`
	DBPath                         string               `json:"db_path,omitempty"`
	SQLite                         CacheSQLiteConfig    `json:"sqlite,omitempty"`
	L1                             CacheTierConfig      `json:"l1,omitempty"`
	L2                             CacheTierConfig      `json:"l2,omitempty"`
	RAG                            CacheRetrieverConfig `json:"rag,omitempty"`
	GlossaryTermbaseVersion        string               `json:"glossary_termbase_version,omitempty"`
	RetrievalCorpusSnapshotVersion string               `json:"retrieval_corpus_snapshot_version,omitempty"`
}

// CacheSQLiteConfig controls sqlite connection pool tuning.
type CacheSQLiteConfig struct {
	MaxOpenConns    int `json:"max_open_conns,omitempty"`
	MaxIdleConns    int `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime int `json:"conn_max_lifetime_seconds,omitempty"`
}

// CacheTierConfig configures a cache tier toggle and capacity hints.
type CacheTierConfig struct {
	Enabled             bool    `json:"enabled,omitempty"`
	MaxItems            int     `json:"max_items,omitempty"`
	AutoAcceptThreshold float64 `json:"auto_accept_threshold,omitempty"`
}

// CacheRetrieverConfig configures optional retrieval augmentation settings.
type CacheRetrieverConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	TopK    int  `json:"top_k,omitempty"`
}

// Load parses and validates i18n configuration from path.
// When path is empty, it defaults to i18n.jsonc in the current working directory.
func Load(path string) (*I18NConfig, error) {
	if strings.TrimSpace(path) == "" {
		path = resolveDefaultPath()
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open i18n config: %w", err)
	}

	jsonContent := jsonc.ToJSON(content)

	decoder := json.NewDecoder(bytes.NewReader(jsonContent))
	decoder.DisallowUnknownFields()

	var cfg I18NConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode i18n config: %w", err)
	}

	if err := expectEOF(decoder); err != nil {
		return nil, err
	}

	autoAcceptThresholdSet, err := hasExplicitCacheL2AutoAcceptThreshold(jsonContent)
	if err != nil {
		return nil, err
	}
	cfg.applyDefaults(autoAcceptThresholdSet)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate i18n config: %w", err)
	}

	return &cfg, nil
}

func resolveDefaultPath() string {
	return defaultConfigPath
}

// Validate validates all cross-field i18n configuration semantics.
func (c I18NConfig) Validate() error {
	targetSet, err := c.validateLocales()
	if err != nil {
		return err
	}

	bucketSet, err := c.validateBuckets()
	if err != nil {
		return err
	}

	groupSet, err := c.validateGroups(targetSet, bucketSet)
	if err != nil {
		return err
	}

	if err := c.validateLLM(groupSet); err != nil {
		return err
	}

	if err := c.validateStorage(); err != nil {
		return err
	}
	if err := c.validateCache(); err != nil {
		return err
	}

	return nil
}

func hasExplicitCacheL2AutoAcceptThreshold(content []byte) (bool, error) {
	type cacheL2Config struct {
		Cache struct {
			L2 struct {
				AutoAcceptThreshold *float64 `json:"auto_accept_threshold"`
			} `json:"l2"`
		} `json:"cache"`
	}

	var cfg cacheL2Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return false, fmt.Errorf("decode i18n config defaults: %w", err)
	}
	return cfg.Cache.L2.AutoAcceptThreshold != nil, nil
}

func (c *I18NConfig) applyDefaults(autoAcceptThresholdSet bool) {
	c.Cache.applyDefaults(autoAcceptThresholdSet)
}

func (c *CacheConfig) applyDefaults(autoAcceptThresholdSet bool) {
	if strings.TrimSpace(c.DBPath) == "" {
		c.DBPath = DefaultCacheDBPath
	}
	if c.SQLite.MaxOpenConns == 0 {
		c.SQLite.MaxOpenConns = DefaultCacheMaxOpen
	}
	if c.SQLite.MaxIdleConns == 0 {
		c.SQLite.MaxIdleConns = DefaultCacheMaxIdle
	}
	if c.SQLite.ConnMaxLifetime == 0 {
		c.SQLite.ConnMaxLifetime = DefaultCacheMaxLifeSec
	}
	if c.L1.MaxItems == 0 {
		c.L1.MaxItems = DefaultCacheL1MaxItems
	}
	if c.L2.AutoAcceptThreshold < 0 || (!autoAcceptThresholdSet && c.L2.AutoAcceptThreshold == 0) {
		c.L2.AutoAcceptThreshold = DefaultCacheL2AutoAcceptThreshold
	}
	if c.RAG.TopK == 0 {
		c.RAG.TopK = DefaultCacheRAGTopK
	}
}

func expectEOF(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}

		return fmt.Errorf("decode trailing i18n config content: %w", err)
	}

	return fmt.Errorf("decode trailing i18n config content: unexpected trailing JSON value")
}

func (c I18NConfig) validateLocales() (map[string]struct{}, error) {
	if strings.TrimSpace(c.Locales.Source) == "" {
		return nil, fmt.Errorf("locales.source: must not be empty")
	}

	if len(c.Locales.Targets) == 0 {
		return nil, fmt.Errorf("locales.targets: must not be empty")
	}

	targetSet := make(map[string]struct{}, len(c.Locales.Targets))

	for i, target := range c.Locales.Targets {
		if strings.TrimSpace(target) == "" {
			return nil, fmt.Errorf("locales.targets[%d]: must not be empty", i)
		}

		if target == c.Locales.Source {
			return nil, fmt.Errorf("locales.targets[%d]: source locale %q is not allowed in targets", i, c.Locales.Source)
		}

		if _, exists := targetSet[target]; exists {
			return nil, fmt.Errorf("locales.targets[%d]: duplicate locale %q", i, target)
		}

		targetSet[target] = struct{}{}
	}

	if err := c.validateFallbacks(targetSet); err != nil {
		return nil, err
	}

	return targetSet, nil
}

func (c I18NConfig) validateFallbacks(targetSet map[string]struct{}) error {
	for locale, chain := range c.Locales.Fallbacks {
		if _, exists := targetSet[locale]; !exists {
			return fmt.Errorf("locales.fallbacks.%s: fallback key must exist in locales.targets", locale)
		}

		if len(chain) == 0 {
			return fmt.Errorf("locales.fallbacks.%s: fallback chain must not be empty", locale)
		}

		seen := make(map[string]struct{}, len(chain))

		for i, candidate := range chain {
			if strings.TrimSpace(candidate) == "" {
				return fmt.Errorf("locales.fallbacks.%s[%d]: must not be empty", locale, i)
			}

			if candidate == locale {
				return fmt.Errorf("locales.fallbacks.%s[%d]: self-reference is not allowed", locale, i)
			}

			if _, exists := seen[candidate]; exists {
				return fmt.Errorf("locales.fallbacks.%s[%d]: duplicate locale %q", locale, i, candidate)
			}

			if candidate != c.Locales.Source {
				if _, exists := targetSet[candidate]; !exists {
					return fmt.Errorf("locales.fallbacks.%s[%d]: locale %q must be in locales.targets or locales.source", locale, i, candidate)
				}
			}

			seen[candidate] = struct{}{}
		}
	}

	if err := c.validateFallbackCycles(); err != nil {
		return err
	}

	return nil
}

func (c I18NConfig) validateFallbackCycles() error {
	state := make(map[string]int, len(c.Locales.Fallbacks))

	var visit func(locale string) error
	visit = func(locale string) error {
		state[locale] = 1

		for _, candidate := range c.Locales.Fallbacks[locale] {
			_, hasFallback := c.Locales.Fallbacks[candidate]
			if !hasFallback {
				continue
			}

			switch state[candidate] {
			case 1:
				return fmt.Errorf("locales.fallbacks.%s: cycle detected through %q", locale, candidate)
			case 0:
				if err := visit(candidate); err != nil {
					return err
				}
			}
		}

		state[locale] = 2

		return nil
	}

	for locale := range c.Locales.Fallbacks {
		if state[locale] == 0 {
			if err := visit(locale); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c I18NConfig) validateBuckets() (map[string]struct{}, error) {
	if len(c.Buckets) == 0 {
		return nil, fmt.Errorf("buckets: must not be empty")
	}

	bucketSet := make(map[string]struct{}, len(c.Buckets))

	for name, bucket := range c.Buckets {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("buckets: bucket name must not be empty")
		}

		if _, exists := bucketSet[name]; exists {
			return nil, fmt.Errorf("buckets.%s: duplicate bucket name", name)
		}

		if err := validateBucket(name, bucket); err != nil {
			return nil, err
		}

		bucketSet[name] = struct{}{}
	}

	return bucketSet, nil
}

func validateBucket(name string, bucket BucketConfig) error {
	if len(bucket.Files) == 0 {
		return fmt.Errorf("buckets.%s.files: must not be empty", name)
	}

	for i, file := range bucket.Files {
		if strings.TrimSpace(file.From) == "" {
			return fmt.Errorf("buckets.%s.files[%d].from: must not be empty", name, i)
		}

		if strings.TrimSpace(file.To) == "" {
			return fmt.Errorf("buckets.%s.files[%d].to: must not be empty", name, i)
		}

		fromSuffix := getFileSuffix(file.From)
		toSuffix := getFileSuffix(file.To)
		if !containsPlaceholder(fromSuffix) && !containsPlaceholder(toSuffix) && fromSuffix != toSuffix {
			return fmt.Errorf("buckets.%s.files[%d]: file suffix mismatch: from=%q and to=%q must have the same extension", name, i, file.From, file.To)
		}
	}

	return nil
}

func getFileSuffix(path string) string {
	return filepath.Ext(path)
}

func containsPlaceholder(s string) bool {
	return strings.Contains(s, "[locale]") ||
		strings.Contains(s, "{{source}}") ||
		strings.Contains(s, "{{target}}") ||
		strings.Contains(s, "{{localeDir}}")
}

func (c I18NConfig) validateGroups(targetSet map[string]struct{}, bucketSet map[string]struct{}) (map[string]struct{}, error) {
	groupSet := make(map[string]struct{}, len(c.Groups))

	for groupName, group := range c.Groups {
		if strings.TrimSpace(groupName) == "" {
			return nil, fmt.Errorf("groups: group name must not be empty")
		}

		if len(group.Targets) == 0 && len(group.Buckets) == 0 {
			return nil, fmt.Errorf("groups.%s: targets and buckets cannot both be empty", groupName)
		}

		if err := validateGroupTargets(groupName, group.Targets, targetSet); err != nil {
			return nil, err
		}

		if err := validateGroupBuckets(groupName, group.Buckets, bucketSet); err != nil {
			return nil, err
		}

		groupSet[groupName] = struct{}{}
	}

	return groupSet, nil
}

func validateGroupTargets(groupName string, targets []string, targetSet map[string]struct{}) error {
	seen := make(map[string]struct{}, len(targets))

	for i, locale := range targets {
		if strings.TrimSpace(locale) == "" {
			return fmt.Errorf("groups.%s.targets[%d]: must not be empty", groupName, i)
		}

		if _, exists := targetSet[locale]; !exists {
			return fmt.Errorf("groups.%s.targets[%d]: locale %q must exist in locales.targets", groupName, i, locale)
		}

		if _, exists := seen[locale]; exists {
			return fmt.Errorf("groups.%s.targets[%d]: duplicate locale %q", groupName, i, locale)
		}

		seen[locale] = struct{}{}
	}

	return nil
}

func validateGroupBuckets(groupName string, buckets []string, bucketSet map[string]struct{}) error {
	seen := make(map[string]struct{}, len(buckets))

	for i, bucketName := range buckets {
		if strings.TrimSpace(bucketName) == "" {
			return fmt.Errorf("groups.%s.buckets[%d]: must not be empty", groupName, i)
		}

		if _, exists := bucketSet[bucketName]; !exists {
			return fmt.Errorf("groups.%s.buckets[%d]: bucket %q must exist in buckets", groupName, i, bucketName)
		}

		if _, exists := seen[bucketName]; exists {
			return fmt.Errorf("groups.%s.buckets[%d]: duplicate bucket %q", groupName, i, bucketName)
		}

		seen[bucketName] = struct{}{}
	}

	return nil
}

func (c I18NConfig) validateLLM(groupSet map[string]struct{}) error {
	if len(c.LLM.Profiles) == 0 {
		return fmt.Errorf("llm.profiles: must not be empty")
	}

	defaultProfile, exists := c.LLM.Profiles[llmDefaultProfile]
	if !exists {
		return fmt.Errorf("llm.profiles.%s: is required", llmDefaultProfile)
	}

	if err := validateProfile("llm.profiles.default", defaultProfile); err != nil {
		return err
	}

	for profileName, profile := range c.LLM.Profiles {
		if strings.TrimSpace(profileName) == "" {
			return fmt.Errorf("llm.profiles: profile name must not be empty")
		}

		if profileName == llmDefaultProfile {
			continue
		}

		if err := validateProfile("llm.profiles."+profileName, profile); err != nil {
			return err
		}
	}

	for i, rule := range c.LLM.Rules {
		if err := validateRule(i, rule, c.LLM.Profiles, groupSet); err != nil {
			return err
		}
	}
	if c.LLM.ContextMemory != nil {
		if err := validateContextMemoryProfile("llm.context_memory", *c.LLM.ContextMemory); err != nil {
			return err
		}
	}

	return nil
}

func validateProfile(fieldPrefix string, profile LLMProfile) error {
	provider := strings.ToLower(strings.TrimSpace(profile.Provider))
	if provider == "" {
		return fmt.Errorf("%s.provider: must not be empty", fieldPrefix)
	}

	switch provider {
	case llmProviderOpenAI, llmProviderAzureOpenAI, llmProviderAnthropic, llmProviderLMStudio, llmProviderGroq, llmProviderMistral, llmProviderOllama, llmProviderGemini, llmProviderBedrock:
	default:
		return fmt.Errorf("%s.provider: unsupported provider %q", fieldPrefix, profile.Provider)
	}

	if strings.TrimSpace(profile.Model) == "" {
		return fmt.Errorf("%s.model: must not be empty", fieldPrefix)
	}

	return nil
}

func validateContextMemoryProfile(fieldPrefix string, profile LLMContextMemoryProfile) error {
	return validateProfile(fieldPrefix, LLMProfile{
		Provider: profile.Provider,
		Model:    profile.Model,
	})
}

func validateRule(index int, rule LLMRule, profiles map[string]LLMProfile, groupSet map[string]struct{}) error {
	if rule.Priority < 0 {
		return fmt.Errorf("llm.rules[%d].priority: must be >= 0", index)
	}

	if strings.TrimSpace(rule.Group) == "" {
		return fmt.Errorf("llm.rules[%d].group: must not be empty", index)
	}

	if strings.TrimSpace(rule.Profile) == "" {
		return fmt.Errorf("llm.rules[%d].profile: must not be empty", index)
	}

	if _, exists := groupSet[rule.Group]; !exists {
		return fmt.Errorf("llm.rules[%d].group: unknown group %q", index, rule.Group)
	}

	if _, exists := profiles[rule.Profile]; !exists {
		return fmt.Errorf("llm.rules[%d].profile: unknown profile %q", index, rule.Profile)
	}

	return nil
}

func (c I18NConfig) validateStorage() error {
	if c.Storage == nil {
		return nil
	}

	if strings.TrimSpace(c.Storage.Adapter) == "" {
		return fmt.Errorf("storage.adapter: must not be empty")
	}

	return nil
}

func (c I18NConfig) validateCache() error {
	if strings.TrimSpace(c.Cache.DBPath) == "" {
		return fmt.Errorf("cache.db_path: must not be empty")
	}
	if c.Cache.SQLite.MaxOpenConns < 0 {
		return fmt.Errorf("cache.sqlite.max_open_conns: must be >= 0")
	}
	if c.Cache.SQLite.MaxIdleConns < 0 {
		return fmt.Errorf("cache.sqlite.max_idle_conns: must be >= 0")
	}
	if c.Cache.SQLite.ConnMaxLifetime < 0 {
		return fmt.Errorf("cache.sqlite.conn_max_lifetime_seconds: must be >= 0")
	}
	if c.Cache.L1.MaxItems < 0 {
		return fmt.Errorf("cache.l1.max_items: must be >= 0")
	}
	if c.Cache.L2.AutoAcceptThreshold < 0 || c.Cache.L2.AutoAcceptThreshold > 1 {
		return fmt.Errorf("cache.l2.auto_accept_threshold: must be between 0 and 1")
	}
	if c.Cache.RAG.TopK < 0 {
		return fmt.Errorf("cache.rag.top_k: must be >= 0")
	}
	return nil
}

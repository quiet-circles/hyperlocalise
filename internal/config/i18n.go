package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tidwall/jsonc"
)

const (
	defaultConfigPath = "i18n.jsonc"
	hiddenConfigPath  = ".i18n.jsonc"
	llmProviderOpenAI = "openai"
)

// I18NConfig defines the i18n configuration file structure.
type I18NConfig struct {
	Locale  LocaleConfig  `json:"locale" jsonschema:"required"`
	Buckets BucketsConfig `json:"buckets" jsonschema:"required"`
	LLM     LLMConfig     `json:"llm" jsonschema:"required"`
}

// LocaleConfig configures source/target locales and fallback hierarchy.
type LocaleConfig struct {
	Source   string              `json:"source" jsonschema:"required"`
	Targets  []string            `json:"targets" jsonschema:"required"`
	Fallback map[string][]string `json:"fallback,omitempty"`
}

// BucketsConfig configures input translation buckets by content type.
type BucketsConfig struct {
	JSON     *BucketConfig `json:"json,omitempty"`
	Markdown *BucketConfig `json:"markdown,omitempty"`
}

// BucketConfig defines glob include patterns for a bucket.
type BucketConfig struct {
	Include []string `json:"include" jsonschema:"required"`
}

// LLMConfig defines model defaults, locale groups, and override rules.
type LLMConfig struct {
	Default   LLMDefaultConfig    `json:"default" jsonschema:"required"`
	Groups    map[string][]string `json:"groups,omitempty"`
	Overrides []LLMOverrideConfig `json:"overrides,omitempty"`
}

// LLMDefaultConfig contains mandatory baseline provider/model/prompt.
type LLMDefaultConfig struct {
	Provider string `json:"provider" jsonschema:"required"`
	Model    string `json:"model" jsonschema:"required"`
	Prompt   string `json:"prompt" jsonschema:"required"`
}

// LLMOverrideConfig applies partial model/prompt overrides by match.
type LLMOverrideConfig struct {
	Match  LLMMatchConfig `json:"match" jsonschema:"required"`
	Model  string         `json:"model,omitempty"`
	Prompt string         `json:"prompt,omitempty"`
}

// LLMMatchConfig selects override targets by group or explicit locales.
type LLMMatchConfig struct {
	Group   string   `json:"group,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

// Load parses and validates i18n configuration from path.
// When path is empty, it prefers i18n.jsonc
func Load(path string) (*I18NConfig, error) {
	if strings.TrimSpace(path) == "" {
		path = resolveDefaultPath()
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open i18n config: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonc.ToJSON(content)))
	decoder.DisallowUnknownFields()

	var cfg I18NConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode i18n config: %w", err)
	}

	if err := expectEOF(decoder); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate i18n config: %w", err)
	}

	return &cfg, nil
}

func resolveDefaultPath() string {
	candidates := []string{
		defaultConfigPath,
		hiddenConfigPath,
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	return defaultConfigPath
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Validate validates all cross-field i18n configuration semantics.
func (c I18NConfig) Validate() error {
	targetSet, err := c.validateLocale()
	if err != nil {
		return err
	}

	if err := c.validateBuckets(); err != nil {
		return err
	}

	if err := c.validateLLM(targetSet); err != nil {
		return err
	}

	return nil
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

func (c I18NConfig) validateLocale() (map[string]struct{}, error) {
	if strings.TrimSpace(c.Locale.Source) == "" {
		return nil, fmt.Errorf("locale.source: must not be empty")
	}

	if len(c.Locale.Targets) == 0 {
		return nil, fmt.Errorf("locale.targets: must not be empty")
	}

	targetSet := make(map[string]struct{}, len(c.Locale.Targets))

	for i, target := range c.Locale.Targets {
		if strings.TrimSpace(target) == "" {
			return nil, fmt.Errorf("locale.targets[%d]: must not be empty", i)
		}

		if target == c.Locale.Source {
			return nil, fmt.Errorf("locale.targets[%d]: source locale %q is not allowed in targets", i, c.Locale.Source)
		}

		if _, exists := targetSet[target]; exists {
			return nil, fmt.Errorf("locale.targets[%d]: duplicate locale %q", i, target)
		}

		targetSet[target] = struct{}{}
	}

	if err := c.validateFallback(targetSet); err != nil {
		return nil, err
	}

	return targetSet, nil
}

func (c I18NConfig) validateFallback(targetSet map[string]struct{}) error {
	for locale, chain := range c.Locale.Fallback {
		if _, exists := targetSet[locale]; !exists {
			return fmt.Errorf("locale.fallback.%s: fallback key must exist in locale.targets", locale)
		}

		if len(chain) == 0 {
			return fmt.Errorf("locale.fallback.%s: fallback chain must not be empty", locale)
		}

		seen := make(map[string]struct{}, len(chain))

		for i, candidate := range chain {
			if strings.TrimSpace(candidate) == "" {
				return fmt.Errorf("locale.fallback.%s[%d]: must not be empty", locale, i)
			}

			if candidate == locale {
				return fmt.Errorf("locale.fallback.%s[%d]: self-reference is not allowed", locale, i)
			}

			if _, exists := seen[candidate]; exists {
				return fmt.Errorf("locale.fallback.%s[%d]: duplicate locale %q", locale, i, candidate)
			}

			if candidate != c.Locale.Source {
				if _, exists := targetSet[candidate]; !exists {
					return fmt.Errorf("locale.fallback.%s[%d]: locale %q must be in locale.targets or locale.source", locale, i, candidate)
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
	state := make(map[string]int, len(c.Locale.Fallback))

	var visit func(locale string) error
	visit = func(locale string) error {
		state[locale] = 1

		for _, candidate := range c.Locale.Fallback[locale] {
			_, hasFallback := c.Locale.Fallback[candidate]
			if !hasFallback {
				continue
			}

			switch state[candidate] {
			case 1:
				return fmt.Errorf("locale.fallback.%s: cycle detected through %q", locale, candidate)
			case 0:
				if err := visit(candidate); err != nil {
					return err
				}
			}
		}

		state[locale] = 2

		return nil
	}

	for locale := range c.Locale.Fallback {
		if state[locale] == 0 {
			if err := visit(locale); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c I18NConfig) validateBuckets() error {
	if c.Buckets.JSON == nil && c.Buckets.Markdown == nil {
		return fmt.Errorf("buckets: at least one of buckets.json or buckets.markdown must be configured")
	}

	if err := validateBucket("buckets.json", c.Buckets.JSON); err != nil {
		return err
	}

	if err := validateBucket("buckets.markdown", c.Buckets.Markdown); err != nil {
		return err
	}

	return nil
}

func validateBucket(field string, bucket *BucketConfig) error {
	if bucket == nil {
		return nil
	}

	if len(bucket.Include) == 0 {
		return fmt.Errorf("%s.include: must not be empty", field)
	}

	for i, pattern := range bucket.Include {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("%s.include[%d]: must not be empty", field, i)
		}
	}

	return nil
}

func (c I18NConfig) validateLLM(targetSet map[string]struct{}) error {
	if strings.TrimSpace(c.LLM.Default.Provider) == "" {
		return fmt.Errorf("llm.default.provider: must not be empty")
	}

	if c.LLM.Default.Provider != llmProviderOpenAI {
		return fmt.Errorf("llm.default.provider: unsupported provider %q", c.LLM.Default.Provider)
	}

	if strings.TrimSpace(c.LLM.Default.Model) == "" {
		return fmt.Errorf("llm.default.model: must not be empty")
	}

	if strings.TrimSpace(c.LLM.Default.Prompt) == "" {
		return fmt.Errorf("llm.default.prompt: must not be empty")
	}

	if err := validateGroups(c.LLM.Groups, targetSet); err != nil {
		return err
	}

	if err := validateOverrides(c.LLM.Overrides, c.LLM.Groups, targetSet); err != nil {
		return err
	}

	return nil
}

func validateGroups(groups map[string][]string, targetSet map[string]struct{}) error {
	for group, locales := range groups {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("llm.groups: group name must not be empty")
		}

		if len(locales) == 0 {
			return fmt.Errorf("llm.groups.%s: must not be empty", group)
		}

		seen := make(map[string]struct{}, len(locales))

		for i, locale := range locales {
			if strings.TrimSpace(locale) == "" {
				return fmt.Errorf("llm.groups.%s[%d]: must not be empty", group, i)
			}

			if _, exists := targetSet[locale]; !exists {
				return fmt.Errorf("llm.groups.%s[%d]: locale %q must exist in locale.targets", group, i, locale)
			}

			if _, exists := seen[locale]; exists {
				return fmt.Errorf("llm.groups.%s[%d]: duplicate locale %q", group, i, locale)
			}

			seen[locale] = struct{}{}
		}
	}

	return nil
}

func validateOverrides(overrides []LLMOverrideConfig, groups map[string][]string, targetSet map[string]struct{}) error {
	for i, override := range overrides {
		hasGroup := strings.TrimSpace(override.Match.Group) != ""
		hasTargets := len(override.Match.Targets) > 0

		if hasGroup == hasTargets {
			return fmt.Errorf("llm.overrides[%d].match: exactly one of group or targets must be set", i)
		}

		if strings.TrimSpace(override.Model) == "" && strings.TrimSpace(override.Prompt) == "" {
			return fmt.Errorf("llm.overrides[%d]: at least one of model or prompt must be set", i)
		}

		if hasGroup {
			if _, exists := groups[override.Match.Group]; !exists {
				return fmt.Errorf("llm.overrides[%d].match.group: unknown group %q", i, override.Match.Group)
			}
		}

		if hasTargets {
			if err := validateOverrideTargets(i, override.Match.Targets, targetSet); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateOverrideTargets(index int, targets []string, targetSet map[string]struct{}) error {
	if len(targets) == 0 {
		return fmt.Errorf("llm.overrides[%d].match.targets: must not be empty", index)
	}

	seen := make(map[string]struct{}, len(targets))

	for i, locale := range targets {
		if strings.TrimSpace(locale) == "" {
			return fmt.Errorf("llm.overrides[%d].match.targets[%d]: must not be empty", index, i)
		}

		if _, exists := targetSet[locale]; !exists {
			return fmt.Errorf("llm.overrides[%d].match.targets[%d]: locale %q must exist in locale.targets", index, i, locale)
		}

		if _, exists := seen[locale]; exists {
			return fmt.Errorf("llm.overrides[%d].match.targets[%d]: duplicate locale %q", index, i, locale)
		}

		seen[locale] = struct{}{}
	}

	return nil
}

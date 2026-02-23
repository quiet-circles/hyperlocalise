package config

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// JSONSchema returns the generated JSON schema for I18NConfig.
func JSONSchema() ([]byte, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(&I18NConfig{})
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal i18n json schema: %w", err)
	}

	return output, nil
}

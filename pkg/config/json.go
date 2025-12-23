package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadJSON loads configuration from a JSON file
func LoadJSON(path string, target interface{}) error {
	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read JSON file %s: %w", path, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// SaveJSON saves configuration to a JSON file
func SaveJSON(path string, config interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Use restrictive permissions by default since configs may contain secrets.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

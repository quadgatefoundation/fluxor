package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML loads configuration from a YAML file
func LoadYAML(path string, target interface{}) error {
	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return nil
}

// SaveYAML saves configuration to a YAML file
func SaveYAML(path string, config interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Use restrictive permissions by default since configs may contain secrets.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}

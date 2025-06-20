// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SiteConfig holds the configuration from the site.yaml file.
// The `yaml` tags are used by the parser to map file keys to struct fields.
type SiteConfig struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	BaseURL     string `yaml:"baseurl"`
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// LoadSiteConfig now uses a proper YAML parser for robust and safe config loading.
func LoadSiteConfig(path string) (SiteConfig, error) {
	cfg := SiteConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return SiteConfig{}, fmt.Errorf("could not read config file at %s: %w", path, err)
	}

	// Unmarshal the YAML data into the SiteConfig struct.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return SiteConfig{}, fmt.Errorf("could not parse config file %s: %w", path, err)
	}

	return cfg, nil
}


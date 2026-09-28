package mcpee

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var defaultEndpoint = "127.0.0.1:8080"

type Config struct {
	Listen   string          `yaml:"listen"`
	Backends []BackendConfig `yaml:"backends"`
}

type BackendConfig struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	if c.Listen == "" {
		c.Listen = defaultEndpoint
	}
	if len(c.Backends) == 0 {
		return Config{}, fmt.Errorf("no backends configured")
	}
	seen := map[string]bool{}
	for i, b := range c.Backends {
		if b.Name == "" || b.Command == "" {
			return Config{}, fmt.Errorf("backend %d: name and command are required", i)
		}
		if seen[b.Name] {
			return Config{}, fmt.Errorf("duplicate backend name %q", b.Name)
		}
		seen[b.Name] = true
	}
	return c, nil
}

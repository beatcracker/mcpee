package mcpee

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allenai/bytefmt"
	"gopkg.in/yaml.v3"
)

var defaultEndpoint = "127.0.0.1:8080"

type Config struct {
	Listen   string
	Backends []BackendConfig
}

type BackendConfig struct {
	Name         string
	Command      string
	Args         []string
	Cwd          string
	InheritEnv   bool
	Env          map[string]string
	MaxFrameSize int
}

type fileConfig struct {
	Listen   string              `yaml:"listen"`
	Backends []fileBackendConfig `yaml:"backends"`
}

type fileBackendConfig struct {
	Name       string            `yaml:"name"`
	Command    string            `yaml:"command"`
	Args       []string          `yaml:"args"`
	Cwd        string            `yaml:"cwd"`
	InheritEnv *bool             `yaml:"inherit_env"`
	Env        map[string]string `yaml:"env"`
	// bytefmt is used only for human-readable config parsing.
	// Runtime frame-limit semantics remain owned by the MCP SDK.
	MaxFrameSize bytefmt.Size `yaml:"max_frame_size"`
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var raw fileConfig
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return Config{}, err
	}
	if raw.Listen == "" {
		raw.Listen = defaultEndpoint
	}
	if len(raw.Backends) == 0 {
		return Config{}, fmt.Errorf("no backends configured")
	}

	c := Config{
		Listen:   raw.Listen,
		Backends: make([]BackendConfig, 0, len(raw.Backends)),
	}
	seen := map[string]bool{}
	for i, b := range raw.Backends {
		if b.Name == "" || b.Command == "" {
			return Config{}, fmt.Errorf("backend %d: name and command are required", i)
		}
		if seen[b.Name] {
			return Config{}, fmt.Errorf("duplicate backend name %q", b.Name)
		}
		seen[b.Name] = true

		cwd, err := effectiveCwd(b.Cwd)
		if err != nil {
			return Config{}, fmt.Errorf("backend %q cwd: %w", b.Name, err)
		}

		frameSize := b.MaxFrameSize.Int64()
		maxFrameSize := int(frameSize)
		if int64(maxFrameSize) != frameSize {
			return Config{}, fmt.Errorf("backend %q max_frame_size does not fit int", b.Name)
		}

		inheritEnv := true
		if b.InheritEnv != nil {
			inheritEnv = *b.InheritEnv
		}

		c.Backends = append(c.Backends, BackendConfig{
			Name:         b.Name,
			Command:      b.Command,
			Args:         b.Args,
			Cwd:          cwd,
			InheritEnv:   inheritEnv,
			Env:          b.Env,
			MaxFrameSize: maxFrameSize,
		})
	}
	return c, nil
}

func effectiveCwd(cwd string) (string, error) {
	if cwd == "" {
		return os.Getwd()
	}
	return filepath.Abs(cwd)
}

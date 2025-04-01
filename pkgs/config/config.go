package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Capability string

const (
	CapabilityQuery        Capability = "query"
	CapabilityMutation     Capability = "mutation"
	CapabilitySubscription Capability = "subscription"
)

type UpstreamServer struct {
	URL            string       `yaml:"url"`
	Capabilities   []Capability `yaml:"capabilities"`
	Weight         int          `yaml:"weight"`
	OperationNames []string     `yaml:"operation_names,omitempty"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type ServerConfig struct {
	ReadTimeout      time.Duration `yaml:"read_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`
	MaxIdleConns     int           `yaml:"max_idle_conns"`
	MaxIdleConnsHost int           `yaml:"max_idle_conns_host"`
	HandshakeTimeout time.Duration `yaml:"handshake_timeout"`
	ResponseTimeout  time.Duration `yaml:"response_timeout"`
}

type Config struct {
	Upstreams []UpstreamServer `yaml:"upstreams"`
	Logging   LogConfig        `yaml:"logging"`
	Server    ServerConfig     `yaml:"server"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Upstreams) == 0 {
		return fmt.Errorf("no upstream servers configured")
	}

	// Set default timeouts if not specified
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 90 * time.Second
	}
	if config.Server.MaxIdleConns == 0 {
		config.Server.MaxIdleConns = 100
	}
	if config.Server.MaxIdleConnsHost == 0 {
		config.Server.MaxIdleConnsHost = 10
	}
	if config.Server.HandshakeTimeout == 0 {
		config.Server.HandshakeTimeout = 10 * time.Second
	}
	if config.Server.ResponseTimeout == 0 {
		config.Server.ResponseTimeout = 30 * time.Second
	}

	for i, upstream := range config.Upstreams {
		if upstream.URL == "" {
			return fmt.Errorf("upstream #%d has empty URL", i+1)
		}
		if upstream.Weight < 1 {
			return fmt.Errorf("upstream #%d has invalid weight: %d", i+1, upstream.Weight)
		}
		if len(upstream.Capabilities) == 0 {
			return fmt.Errorf("upstream #%d has no capabilities", i+1)
		}
	}

	return nil
}

package config

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/ini.v1"
)

// Config holds all loaded host configurations
type Config struct {
	Hosts map[string]*HostConfig
}

// DefaultConfigPath returns the default config file path following XDG conventions
func DefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "hosts.ini"
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "hop", "hosts.ini")
}

// Load reads the configuration from the default path
func Load() (*Config, error) {
	return LoadFromPath(DefaultConfigPath())
}

// LoadFromPath reads the configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Hosts: make(map[string]*HostConfig),
	}

	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "DEFAULT" {
			continue
		}

		host := &HostConfig{
			Name:      name,
			Type:      section.Key("type").MustString("ssh"),
			Host:      section.Key("host").String(),
			User:      section.Key("user").String(),
			Port:      section.Key("port").MustInt(0),
			Identity:  section.Key("identity").String(),
			Container: section.Key("container").String(),
			Shell:     section.Key("shell").MustString("/bin/sh"),
			Namespace: section.Key("namespace").MustString("default"),
			Pod:       section.Key("pod").String(),
			Context:   section.Key("context").String(),
		}

		// Parse port forwarding
		if portStr := section.Key("local_port").String(); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				host.LocalPort = p
			}
		}
		if portStr := section.Key("remote_port").String(); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				host.RemotePort = p
			}
		}

		config.Hosts[name] = host
	}

	return config, nil
}

// Get retrieves a host configuration by name
func (c *Config) Get(name string) (*HostConfig, bool) {
	host, ok := c.Hosts[name]
	return host, ok
}

// Names returns all configured host names
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Hosts))
	for name := range c.Hosts {
		names = append(names, name)
	}
	return names
}

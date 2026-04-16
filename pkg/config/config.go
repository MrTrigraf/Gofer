package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Host              string `yaml:"host"`
	Port              string `yaml:"port"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	Name              string `yaml:"name"`
	SSLMode           string `yaml:"sslmode"`
	MaxConns          int32  `yaml:"max_conns"`
	MinConns          int32  `yaml:"min_conns"`
	MaxConnLifetime   string `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   string `yaml:"max_conn_idle_time"`
	HealthCheckPeriod string `yaml:"health_check_period"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	Refresh    string `yaml:"refresh"`
	AccessTTL  string `yaml:"access_ttl"`
	RefreshTTL string `yaml:"refresh_ttl"`
}

func Load() (*Config, error) {
	file, err := os.Open("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)

	if err = decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &cfg, nil
}

package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"SERVER_PORT"`
}

type DatabaseConfig struct {
	Host              string `yaml:"host" env:"DB_HOST"`
	Port              string `yaml:"port" env:"DB_PORT"`
	User              string `yaml:"user" env:"DB_USER"`
	Password          string `yaml:"password" env:"DB_PASSWORD"`
	Name              string `yaml:"name" env:"DB_NAME"`
	SSLMode           string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
	MaxConns          int32  `yaml:"max_conns" env:"DB_MAX_CONNS" env-default:"25"`
	MinConns          int32  `yaml:"min_conns" env:"DB_MIN_CONNS" env-default:"5"`
	MaxConnLifetime   string `yaml:"max_conn_lifetime" env:"DB_MAX_CONN_LIFETIME" env-default:"5m"`
	MaxConnIdleTime   string `yaml:"max_conn_idle_time" env:"DB_MAX_CONN_IDLE_TIME" env-default:"2m"`
	HealthCheckPeriod string `yaml:"health_check_period" env:"DB_HEALTH_CHECK_PERIOD" env-default:"1m"`
}

type JWTConfig struct {
	Secret     string `env:"JWT_SECRET"`
	AccessTTL  string `yaml:"access_ttl" env:"JWT_ACCESS_TTL"`
	RefreshTTL string `yaml:"refresh_ttl" env:"JWT_REFRESH_TTL"`
}

func Load() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig("config/config.yaml", &cfg)
	return &cfg, err
}

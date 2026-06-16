package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Polling  PollingConfig  `mapstructure:"polling"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type PollingConfig struct {
	IntervalSeconds      int  `mapstructure:"interval_seconds"`
	TimeoutSeconds       int  `mapstructure:"timeout_seconds"`
	RetryCount           int  `mapstructure:"retry_count"`
	ICMPEnabled          bool `mapstructure:"icmp_enabled"`
	HistoryRetentionDays int  `mapstructure:"history_retention_days"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.path", "./data/embrionix.db")
	v.SetDefault("logging.level", "info")
	v.SetDefault("polling.interval_seconds", 30)
	v.SetDefault("polling.timeout_seconds", 10)
	v.SetDefault("polling.retry_count", 2)
	v.SetDefault("polling.icmp_enabled", true)
	v.SetDefault("polling.history_retention_days", 30)
	v.SetDefault("cors.allowed_origins", []string{"http://localhost:5173"})

	v.SetEnvPrefix("EMB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	return &cfg, nil
}

func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

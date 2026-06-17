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
	Alerting AlertingConfig `mapstructure:"alerting"`
	Reports  ReportsConfig  `mapstructure:"reports"`
	Updates  UpdatesConfig  `mapstructure:"updates"`
	Auth     AuthConfig     `mapstructure:"auth"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

// ReportsConfig controls scheduled fleet reports. The PDF report is always
// available on demand; the scheduler only delivers a text summary to the
// alerting webhook on the configured cron.
type ReportsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Cron    string `mapstructure:"cron"` // standard 5-field cron (e.g. "0 8 * * 1" = Mon 08:00)
}

// UpdatesConfig controls in-app update checking and self-update. The app checks
// GitHub Releases of Repo for a newer tag; an admin can then trigger a self-update
// that downloads the matching binary, swaps it in place, and restarts.
type UpdatesConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	Repo               string `mapstructure:"repo"`                 // "owner/name" on GitHub
	CheckIntervalHours int    `mapstructure:"check_interval_hours"` // how often to poll GitHub Releases
	// RestartMode controls how the process restarts after a self-update:
	//   "self" — spawn a fresh instance, then exit (for unsupervised runs).
	//   "exit" — just exit and let a service manager (systemd/NSSM) restart it.
	RestartMode string `mapstructure:"restart_mode"`
}

// AuthConfig controls authentication and RBAC. Disabled by default so an
// existing deployment keeps working without any login; enabling it requires a
// jwt_secret and seeds an admin account on first start.
type AuthConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	JWTSecret     string `mapstructure:"jwt_secret"`
	TokenTTLHours int    `mapstructure:"token_ttl_hours"`
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
	APIKey        string `mapstructure:"api_key"` // optional static key for integrations (admin-equivalent)
}

// AlertingConfig holds tunable health thresholds and notification settings.
type AlertingConfig struct {
	TempWarningC     float64  `mapstructure:"temp_warning_c"`
	TempCriticalC    float64  `mapstructure:"temp_critical_c"`
	ResponseWarnMs   int64    `mapstructure:"response_warning_ms"`
	WebhookURL       string   `mapstructure:"webhook_url"`
	WebhookOn        []string `mapstructure:"webhook_on"` // statuses whose entry fires a webhook
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
	FullEvery            int  `mapstructure:"full_every"`            // do a full (all-endpoint) poll every Nth cycle
	MaxConcurrentPolls   int  `mapstructure:"max_concurrent_polls"`  // cap simultaneous device polls to bound bursts
	// BlueProbe selects how the Blue management path's reachability is checked:
	//   "icmp" — OS ping (for devices that answer ICMP but not TCP on Blue)
	//   "tcp"  — TCP connect to port 80 (same as Red)
	// Red is always probed via TCP (it runs the HTTP management API).
	BlueProbe string `mapstructure:"blue_probe"`
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
	v.SetDefault("polling.full_every", 10)
	v.SetDefault("polling.max_concurrent_polls", 8)
	v.SetDefault("polling.blue_probe", "icmp")
	v.SetDefault("alerting.temp_warning_c", 70)
	v.SetDefault("alerting.temp_critical_c", 75)
	v.SetDefault("alerting.response_warning_ms", 6000)
	v.SetDefault("alerting.webhook_url", "")
	v.SetDefault("alerting.webhook_on", []string{"critical", "offline"})
	v.SetDefault("reports.enabled", false)
	v.SetDefault("reports.cron", "0 8 * * 1")
	v.SetDefault("updates.enabled", true)
	v.SetDefault("updates.repo", "jojo024/Embrionix-Dashboard")
	v.SetDefault("updates.check_interval_hours", 6)
	v.SetDefault("updates.restart_mode", "self")
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.token_ttl_hours", 12)
	v.SetDefault("auth.admin_username", "admin")
	v.SetDefault("auth.admin_password", "")
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

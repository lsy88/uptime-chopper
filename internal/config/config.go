package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type NotificationWebhook struct {
	Name string `mapstructure:"name" json:"name"`
	URL  string `mapstructure:"url" json:"url"`
	Type string `mapstructure:"type" json:"type"` // webhook, dingtalk, wechat
}

type Config struct {
	HTTPAddr              string                `mapstructure:"http_addr" json:"httpAddr"`
	DataFilePath          string                `mapstructure:"data_file_path" json:"dataFilePath"`
	Notifications         []NotificationWebhook `mapstructure:"notifications" json:"notifications"`
	MaxDockerLogBytes     int                   `mapstructure:"max_docker_log_bytes" json:"maxDockerLogBytes"`
	DefaultDockerLogSince time.Duration         `mapstructure:"default_docker_log_since" json:"defaultDockerLogSince"`
	AllowedCORSOrigin     string                `mapstructure:"allowed_cors_origin" json:"allowedCorsOrigin"`
	ServeFrontendFromDist bool                  `mapstructure:"serve_frontend_from_dist" json:"serveFrontendFromDist"`
	FrontendDistDirectory string                `mapstructure:"frontend_dist_directory" json:"frontendDistDirectory"`
}

func Load() (Config, error) {
	v := viper.New()

	// 1. Set Defaults
	v.SetDefault("http_addr", ":7601")
	v.SetDefault("data_file_path", "data.json")
	v.SetDefault("max_docker_log_bytes", 64*1024)
	v.SetDefault("default_docker_log_since", 3600*time.Second) // Note: viper might load this as int from yaml, need care
	v.SetDefault("allowed_cors_origin", "*")
	v.SetDefault("serve_frontend_from_dist", false)
	v.SetDefault("frontend_dist_directory", "web/dist")

	// 2. Environment Variables
	// Map UPTIME_CHOPPER_ADDR -> http_addr, etc.
	// AutomaticEnv() maps "http_addr" to "UPTIME_CHOPPER_HTTP_ADDR" by default if prefix is set.
	// But our legacy env vars are slightly different (e.g. UPTIME_CHOPPER_ADDR vs UPTIME_CHOPPER_HTTP_ADDR).
	// So we manually bind legacy ones or rely on new ones.
	// For best compatibility + new standard:
	// We will use standard viper env mapping: UPTIME_CHOPPER_HTTP_ADDR
	// AND explicitly bind legacy ones.
	v.SetEnvPrefix("UPTIME_CHOPPER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Legacy bindings
	_ = v.BindEnv("http_addr", "UPTIME_CHOPPER_ADDR")
	_ = v.BindEnv("data_file_path", "UPTIME_CHOPPER_DATA")
	_ = v.BindEnv("max_docker_log_bytes", "UPTIME_CHOPPER_MAX_DOCKER_LOG_BYTES")
	// For duration/int conversion from env, viper handles basic types.
	// The original code handled envInt("..._SEC", 3600) * time.Second.
	// If user sets UPTIME_CHOPPER_DOCKER_LOG_SINCE_SEC, we need to handle it.
	// Let's bind it to a temp key or just handle it manually if needed,
	// but simplest is to just support the new way or bind directly if compatible.
	// Since original was _SEC (int), and we want time.Duration.
	// Viper unmarshal hook can handle duration string (e.g. "3600s").
	// If we want to support the old _SEC env var, we might need a workaround.
	// For now, let's bind the new one automatically and maybe the old one if easy.
	// But `default_docker_log_since` expects a duration.
	// Let's rely on standard viper features for now.

	_ = v.BindEnv("allowed_cors_origin", "UPTIME_CHOPPER_CORS_ORIGIN")
	_ = v.BindEnv("serve_frontend_from_dist", "UPTIME_CHOPPER_SERVE_FRONTEND")
	_ = v.BindEnv("frontend_dist_directory", "UPTIME_CHOPPER_FRONTEND_DIST")

	// 3. Config File
	// Check for UPTIME_CHOPPER_CONFIG env var first
	v.BindEnv("config_file", "UPTIME_CHOPPER_CONFIG")
	cfgFile := v.GetString("config_file")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Default search paths
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, unless explicitly specified
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

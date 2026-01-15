package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type NotificationWebhook struct {
	Name string `mapstructure:"name" yaml:"name"`
	URL  string `mapstructure:"url" yaml:"url"`
	Type string `mapstructure:"type" yaml:"type"` // webhook, dingtalk, wechat
}

type Config struct {
	HTTPAddr              string                `mapstructure:"http_addr" yaml:"http_addr"`
	DataFilePath          string                `mapstructure:"data_file_path" yaml:"data_file_path"`
	Notifications         []NotificationWebhook `mapstructure:"notifications" yaml:"notifications"`
	MaxDockerLogBytes     int                   `mapstructure:"max_docker_log_bytes" yaml:"max_docker_log_bytes"`
	DefaultDockerLogSince time.Duration         `mapstructure:"default_docker_log_since" yaml:"default_docker_log_since"`
	AllowedCORSOrigin     string                `mapstructure:"allowed_cors_origin" yaml:"allowed_cors_origin"`
	ServeFrontendFromDist bool                  `mapstructure:"serve_frontend_from_dist" yaml:"serve_frontend_from_dist"`
	FrontendDistDirectory string                `mapstructure:"frontend_dist_directory" yaml:"frontend_dist_directory"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("UPTIME_CHOPPER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, unless explicitly specified
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.DataFilePath == "" {
		cfg.DataFilePath = "data/data.json"
	}

	return &cfg, nil
}

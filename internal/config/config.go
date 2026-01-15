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

	var cfg *Config
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

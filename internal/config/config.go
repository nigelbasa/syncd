package config

import (
	"time"

	"github.com/spf13/viper"
)

// Load reads the syncd configuration from the given file path,
// with environment variable overrides (prefix: SYNCD_).
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Env overrides: SYNCD_APP_MODE, SYNCD_DATABASE_DSN, etc.
	v.AutomaticEnv()
	v.SetEnvPrefix("SYNCD")

	// Defaults
	v.SetDefault("app.mode", "client")
	v.SetDefault("app.port", 8080)
	v.SetDefault("sync.poll_interval", "5s")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Handle duration string parsing explicitly since mapstructure
	// doesn't natively decode time.Duration from string.
	if cfg.Sync.PollInterval == 0 {
		d, err := time.ParseDuration(v.GetString("sync.poll_interval"))
		if err != nil {
			cfg.Sync.PollInterval = 5 * time.Second
		} else {
			cfg.Sync.PollInterval = d
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

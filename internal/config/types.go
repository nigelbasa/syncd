package config

import "time"

// Config is the root configuration for syncd.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Sync     SyncConfig     `mapstructure:"sync"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	// Mode is either "client" or "server".
	Mode string `mapstructure:"mode"`
	// Port is the HTTP listen port (server mode only).
	Port int `mapstructure:"port"`
	// RemoteURL is the server URL to sync with (client mode only).
	RemoteURL string `mapstructure:"remote_url"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	// Driver is either "sqlite" or "postgres".
	Driver string `mapstructure:"driver"`
	// DSN is the data source name or file path.
	DSN string `mapstructure:"dsn"`
	// SyncDB is the path to the attached outbox database (SQLite only).
	SyncDB string `mapstructure:"sync_db"`
}

// SyncConfig holds synchronization behaviour settings.
type SyncConfig struct {
	// APIKey is the shared secret for client-server authentication.
	APIKey string `mapstructure:"api_key"`
	// PollInterval controls how often the watcher checks for new outbox events.
	PollInterval time.Duration `mapstructure:"poll_interval"`
	// Tables is the list of application tables to watch for CDC.
	Tables []string `mapstructure:"tables"`
}

// Validate performs basic validation on the loaded configuration.
func (c *Config) Validate() error {
	if c.App.Mode != "client" && c.App.Mode != "server" {
		return ErrInvalidMode
	}
	if c.Database.Driver != "sqlite" && c.Database.Driver != "postgres" {
		return ErrInvalidDriver
	}
	if c.Database.DSN == "" {
		return ErrMissingDSN
	}
	if c.App.Mode == "client" && c.App.RemoteURL == "" {
		return ErrMissingRemoteURL
	}
	if c.App.Mode == "client" && c.Database.Driver == "sqlite" && c.Database.SyncDB == "" {
		return ErrMissingSyncDB
	}
	if len(c.Sync.Tables) == 0 {
		return ErrNoTables
	}
	if c.Sync.APIKey == "" {
		return ErrMissingAPIKey
	}
	return nil
}

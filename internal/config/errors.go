package config

import "errors"

var (
	ErrInvalidMode      = errors.New("syncd: app.mode must be 'client' or 'server'")
	ErrInvalidDriver    = errors.New("syncd: database.driver must be 'sqlite' or 'postgres'")
	ErrMissingDSN       = errors.New("syncd: database.dsn is required")
	ErrMissingRemoteURL = errors.New("syncd: app.remote_url is required in client mode")
	ErrMissingSyncDB    = errors.New("syncd: database.sync_db is required for SQLite client mode")
	ErrNoTables         = errors.New("syncd: sync.tables must contain at least one table")
	ErrMissingAPIKey    = errors.New("syncd: sync.api_key is required")
)

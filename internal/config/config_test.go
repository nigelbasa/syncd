package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "syncd.yaml")

	content := `
app:
  mode: "client"
  port: 9090
  remote_url: "http://localhost:8080"
database:
  driver: "sqlite"
  dsn: "./app.db"
  sync_db: "./sync.db"
sync:
  api_key: "test-key"
  poll_interval: "10s"
  tables:
    - "users"
    - "items"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.App.Mode != "client" {
		t.Errorf("Mode = %q, want %q", cfg.App.Mode, "client")
	}
	if cfg.App.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.App.Port, 9090)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Driver = %q, want %q", cfg.Database.Driver, "sqlite")
	}
	if cfg.Sync.PollInterval != 10*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.Sync.PollInterval, 10*time.Second)
	}
	if len(cfg.Sync.Tables) != 2 {
		t.Errorf("Tables count = %d, want 2", len(cfg.Sync.Tables))
	}
}

func TestLoad_MissingDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "syncd.yaml")

	content := `
app:
  mode: "server"
database:
  driver: "postgres"
  dsn: ""
sync:
  api_key: "key"
  tables: ["users"]
`
	os.WriteFile(path, []byte(content), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Expected error for missing DSN")
	}
}

func TestLoad_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "syncd.yaml")

	content := `
app:
  mode: "invalid"
database:
  driver: "sqlite"
  dsn: "./app.db"
sync:
  api_key: "key"
  tables: ["users"]
`
	os.WriteFile(path, []byte(content), 0644)

	_, err := Load(path)
	if err != ErrInvalidMode {
		t.Fatalf("Expected ErrInvalidMode, got: %v", err)
	}
}

func TestLoad_ClientMissingRemoteURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "syncd.yaml")

	content := `
app:
  mode: "client"
  remote_url: ""
database:
  driver: "sqlite"
  dsn: "./app.db"
  sync_db: "./sync.db"
sync:
  api_key: "key"
  tables: ["users"]
`
	os.WriteFile(path, []byte(content), 0644)

	_, err := Load(path)
	if err != ErrMissingRemoteURL {
		t.Fatalf("Expected ErrMissingRemoteURL, got: %v", err)
	}
}

func TestValidate_NoTables(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Mode: "server"},
		Database: DatabaseConfig{Driver: "postgres", DSN: "postgres://localhost/db"},
		Sync:     SyncConfig{APIKey: "key", Tables: nil},
	}
	if err := cfg.Validate(); err != ErrNoTables {
		t.Errorf("Expected ErrNoTables, got: %v", err)
	}
}

# Module: CLI & Config (`internal/config` & `cmd/syncd`)

## Description
This module is responsible for the user-facing command-line interface and the loading/validation of application settings. Syncd is designed to be a standalone binary, so it uses CLI commands to initialize the system, install literal database triggers, manage administrative credentials, and start the daemon.

## Key CLI Commands (using `cobra`)

- `syncd init`: Generates a default `syncd.yaml` configuration file.
- `syncd install-triggers --table <tabular>`: Reads the DB schema, generates the necessary PostgreSQL or SQLite triggers for the specified table, and executes them.
- `syncd start --mode <server|client>`: Boots the background sync daemon and (if server) the web API.
- `syncd admin create <username>`: Hashes a password and creates a web portal admin user in the underlying DB.

## Configuration File (`syncd.yaml` or `.env`)
Syncd relies heavily on its configuration file to know its identity and where to look.

### Expected Config Structure
```yaml
app:
  mode: "client" # or "server"
  port: 8080
  remote_url: "https://sync.myvps.com" # Only required if mode=client

database:
  driver: "sqlite" # or "postgres"
  dsn: "./local_data/app.db" # Connection string or path
  sync_db: "./local_data/syncd_logs.db" # (SQLite only) The ATTACH target for outbox 

sync:
  api_key: "XXXXXX" # Token to auth with the server
  poll_interval: "5s" # How often to check for local changes
  tables: 
    - "users"
    - "payments"
```

## Structure
```
cmd/syncd/
├── main.go        (Execute cobra root)
├── start.go       (Start daemon command)
├── init.go        (Generate config command)
└── triggers.go    (Trigger installation command)

internal/config/
├── config.go      (Viper/Env parsing logic)
└── types.go       (Config struct definitions)
```

## Snippets
### Viper Config Loading
```go
func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.AutomaticEnv()
    viper.SetEnvPrefix("SYNCD")
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

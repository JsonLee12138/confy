# Confy

A full-scenario Go configuration management library built on [viper](https://github.com/spf13/viper).

## Features

- **Environment-aware multi-file merging** — auto-discovers `config.yaml` → `config.local.yaml` → `config.{env}.yaml` → `config.{env}.local.yaml`
- **Environment variable override** — `database.host` ↔ `MYAPP_DATABASE_HOST`
- **Multi-format support** — YAML, JSON, TOML (mixed in same directory)
- **.env file loading** — environment-aware `.env` discovery: `.env` → `.env.local` → `.env.{env}` → `.env.{env}.local`
- **Hot-reload** — file watching with fsnotify for development
- **Struct binding** — unmarshal to Go structs with `mapstructure` tags
- **Struct defaults** — `default:"value"` tag support via `creasty/defaults`
- **Validation** — custom `Validator` interface + `required:"true"` tag
- **Config encryption** — AES-256-GCM for sensitive values
- **Template inheritance** — `base: parent.yaml` for config inheritance
- **Snapshot/Restore** — capture and rollback config state
- **Functional options** — clean, extensible API

## Install

```bash
go get github.com/JsonLee12138/confy@latest
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/JsonLee12138/confy"
)

type AppConfig struct {
    Server struct {
        Port int    `mapstructure:"port"`
        Host string `mapstructure:"host"`
    } `mapstructure:"server"`
    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Password string `mapstructure:"password"`
    } `mapstructure:"database"`
}

func main() {
    cfg, err := confy.New(
        confy.WithPath("config"),
        confy.WithEnvPrefix("MYAPP"),
    )
    if err != nil {
        panic(err)
    }

    var appCfg AppConfig
    if err := cfg.BindWithDefaults(&appCfg); err != nil {
        panic(err)
    }

    fmt.Printf("Server: %s:%d\n", appCfg.Server.Host, appCfg.Server.Port)
}
```

## Configuration Files

Place config files in the `config/` directory:

```
config/
├── config.yaml              # Base config
├── config.local.yaml        # Local overrides (git-ignored)
├── config.production.yaml   # Production overrides
├── database.yaml            # Module config (loaded with LoadAll)
```

### File Priority (highest last)

1. `config.yaml`
2. `config.local.yaml`
3. `config.{env}.yaml`
4. `config.{env}.local.yaml`

Environment is detected via `GO_ENV_MODE` env var (`development`/`production`/`test`).

## Options

| Option | Description |
|--------|-------------|
| `WithPath(path)` | Config directory path |
| `WithFile(name)` | Base config file name |
| `WithFileType(ft)` | Default format: `confy.YAML`, `confy.JSON`, `confy.TOML` |
| `WithEnvPrefix(prefix)` | Env var prefix |
| `WithWatch(enable)` | Enable hot-reload |
| `WithOnChange(fn)` | Callback on config change |
| `WithLoadAll(enable)` | Load all files in directory |
| `WithDotEnv(path)` | Load a `.env` file |
| `WithDotEnvAuto(dir...)` | Auto-discover `.env` files by env mode |
| `WithEncryption(algo, key)` | Enable value encryption |

## .env Files

Two ways to load `.env` files:

### Manual — `WithDotEnv(path)`

Load specific `.env` files by path:

```go
confy.New(
    confy.WithDotEnv(".env"),
    confy.WithDotEnv(".env.local"),
)
```

### Auto-discovery — `WithDotEnvAuto(dir...)`

Automatically discovers and loads `.env` files based on `GO_ENV_MODE`, following the same priority pattern as config files:

```go
confy.New(confy.WithDotEnvAuto())      // search in current directory
confy.New(confy.WithDotEnvAuto("etc")) // search in "etc/" directory
```

#### .env File Priority (highest last)

1. `.env`
2. `.env.local`
3. `.env.{env}`
4. `.env.{env}.local`

When `GO_ENV_MODE=production`, the discovery order is:

```
.env → .env.local → .env.production → .env.production.local
```

Environment aliases are supported: `dev`/`development`, `prod`/`pro`/`production`, `test`/`testing`.

Files that don't exist are silently skipped. System environment variables always take highest priority.

## Encrypted Values

Mark encrypted values with `enc:AES_GCM:` prefix:

```yaml
database:
  password: "enc:AES_GCM:base64EncodedCiphertext"
```

## Template Inheritance

```yaml
# base.yaml
server:
  port: 8080
  host: 0.0.0.0

# config.yaml
base: base.yaml
server:
  port: 9090  # overrides parent
```

## Example

A complete Gin + confy demo is available in the `example/` directory:

```bash
cd example
go run .
```

The example demonstrates:
- **Multi-file merging** — `config.yaml` + `config.local.yaml`
- **Template inheritance** — `config.yaml` extends `base.yaml`
- **Hot-reload** — config changes trigger automatic rebind
- **Struct defaults** — Go struct `default` tags fill in missing values
- **Validation** — `App.Validate()` enforces port ranges
- **Env var overrides** — set `MYAPP_SERVER_PORT=9090` to override

Endpoints:
- `GET /health` — health check
- `GET /config` — view current config (password masked)

## License

MIT

# Confy

A full-scenario Go configuration management library built on [viper](https://github.com/spf13/viper).

## Features

- **Environment-aware multi-file merging** — auto-discovers `config.yaml` → `config.local.yaml` → `config.{env}.yaml` → `config.{env}.local.yaml`
- **Environment variable override** — `database.host` ↔ `MYAPP_DATABASE_HOST`
- **Multi-format support** — YAML, JSON, TOML (mixed in same directory)
- **.env file loading** — auto-loads `.env` files with proper priority
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
| `WithEncryption(algo, key)` | Enable value encryption |

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

## License

MIT

# Confy

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-%3E%3D%201.23-00ADD8?style=flat-square&logo=go" alt="Go Version" /></a>
  <a href="https://pkg.go.dev/github.com/JsonLee12138/confy"><img src="https://img.shields.io/badge/Go_Reference-pkg.go.dev-blue?style=flat-square&logo=go" alt="Go Reference" /></a>
  <a href="https://goreportcard.com/report/github.com/JsonLee12138/confy"><img src="https://goreportcard.com/badge/github.com/JsonLee12138/confy?style=flat-square" alt="Go Report Card" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg?style=flat-square" alt="License: MIT" /></a>
</p>

<p align="center">
  <b>English</b> | <a href="README.zh.md">中文文档</a>
</p>

---

**Confy** is a Go configuration library built on [Viper](https://github.com/spf13/viper). Load **local files**, **Nacos**, or **environment variables** into structs through one binding API — with multi-environment cascading, tag-based defaults, zero-value retention, hot-reloading, AES-GCM field decryption, and explicit validation.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Choose a Provider](#choose-a-provider)
- [Watching, Encryption & Validation](#watching-encryption--validation)
- [Runnable Examples](#runnable-examples)
- [Development](#development)
- [License](#license)

## Features

- **Multi-Source Providers** — Local files (YAML / JSON / TOML), Nacos 2.x remote config, and standalone ENV.
- **Environment Cascading & Inheritance** — `config.yaml → config.local.yaml → config.{env}.yaml → config.{env}.local.yaml`, plus `base: parent.yaml` template inheritance with cycle detection.
- **Safe Struct Binding** — `default:"…"` tags applied before decoding; explicit `0` / `false` / `""` preserved; atomic target swap on success, rollback on failure.
- **Hot-Reloading** — Shared `RWMutex`, async `OnChange` / `OnError` callbacks outside locks.
- **Field Decryption** — `enc:AES_GCM:` prefix decrypted only in the binding view; source ciphertext stays intact.
- **Validation** — `Validator` interface for business rules; `required:"true"` tag for structural checks.
- **Minimal Core** — Root package does not depend on the Nacos SDK.

## Installation

```sh
go get github.com/JsonLee12138/confy@latest
```

## Quick Start

Create `config/config.yaml`:

```yaml
server:
  host: 127.0.0.1
  port: 8080
```

Create `main.go`:

```go
package main

import (
    "fmt"
    "log"

    "github.com/JsonLee12138/confy"
)

type AppConfig struct {
    Server struct {
        Host string `mapstructure:"host" default:"127.0.0.1"`
        Port int    `mapstructure:"port" default:"8080"`
    } `mapstructure:"server"`
}

func main() {
    cfg, err := confy.New(
        confy.WithPath("config"),
        confy.WithEnvPrefix("MYAPP"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer cfg.Close()

    var app AppConfig
    if err := cfg.Bind(&app); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("server=%s:%d\n", app.Server.Host, app.Server.Port)
}
```

```sh
go run .
MYAPP_SERVER_PORT=9090 go run .   # override via ENV
```

`Bind` applies `default` tags before decoding, preserving configured `0`, `false`, and empty strings. Each bind decodes into a fresh struct; failure leaves the target unchanged. `time.Time` fields accept RFC3339 timestamps. `BindWithDefaults` is deprecated — use `Bind`.

## Choose a Provider

| Source | Setup | Updates |
| --- | --- | --- |
| Local | Default, or `local.New(…)` | Native file watching |
| Nacos | `nacos.New(…)` | Remote subscription & reconciliation |
| ENV | `env.New(schema, prefix)` | Explicit rebinding; no watcher |

**Local files** — the quick start above uses the default Local Provider. You can also select it explicitly with `confy.WithProvider(local.New(…))`. Files merge in order: `config.yaml → config.local.yaml → config.{env}.yaml → config.{env}.local.yaml`. Set `GO_ENV_MODE` to `development` / `production` / `test` (aliases like `dev`, `prod` work too). A file can inherit another via `base: base.yaml`.

**Nacos** — use the same `AppConfig` and `Bind` call:

```go
import (
    "os"
    "github.com/JsonLee12138/confy/provider/nacos"
    "github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

cfg, err := confy.New(
    confy.WithProvider(nacos.New(
        nacos.WithServers([]constant.ServerConfig{{IpAddr: "127.0.0.1", Port: 8848}}),
        nacos.WithClientConfig(constant.ClientConfig{
            NamespaceId: "",
            Username:    os.Getenv("NACOS_USERNAME"),
            Password:    os.Getenv("NACOS_PASSWORD"),
            TimeoutMs:   5000,
        }),
        nacos.WithDataID("app.yaml"),
        nacos.WithGroup("APP_GROUP"),
        nacos.WithFormat(confy.YAML),
    )),
    confy.WithEnvPrefix("MYAPP"),
)
```

YAML is the default format; the Data ID extension does not select a parser. Inject credentials before constructing the Provider. See the [Nacos guide](provider/nacos/README.md) for details.

**Environment variables** — for bootstrap settings, bind ENV directly:

```go
var conn struct {
    Address string `mapstructure:"address"`
    Port    int    `mapstructure:"port"`
}
err := confy.BindEnv(&conn, "NACOS")   // reads NACOS_ADDRESS, NACOS_PORT
```

Or use the full Config API: `confy.New(confy.WithProvider(env.New(AppConfig{}, "MYAPP")))`. See the [ENV guide](provider/env/README.md).

## Watching, Encryption & Validation

```go
confy.WithWatch(true),
confy.WithOnChange(func(event confy.Event) { /* target updated */ }),
confy.WithOnError(func(err error) { /* handle refresh error */ }),
```

Watching starts after the first successful `Bind`. Callbacks run outside internal locks. Parse errors preserve the previous source; binding errors preserve the previous target. `Close` stops delivery and releases Provider resources.

Encrypted fields use the `enc:AES_GCM:` prefix and require `WithEncryption("aes-gcm", key)`. Decryption happens only during binding; source ciphertext remains in shared configuration.

Implement `Validate() error` on your struct and call `cfg.Validate()` after binding. `Bind` does not invoke it automatically.

## Runnable Examples

```sh
go run ./example/local
MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./example/env
NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_GROUP=APP_GROUP NACOS_DATA_ID=app.yaml go run ./example/nacos
```

See [example/README.md](example/README.md) for prerequisites. The original Gin demo is still available with `go run ./example`.

## Development

New adapters go in `provider/<source>` and implement `provider.Provider`. Watching adapters also implement `provider.WatchProvider`. See [AGENTS.md](AGENTS.md#provider-adapter-development).

```sh
go test -race ./...
go vet ./...
```

Cross-package E2E tests live in [`e2e/`](e2e/README.md). Real Nacos tests require the `nacos_integration` build tag — see [provider/TESTING.md](provider/TESTING.md).

## License

[MIT](LICENSE)

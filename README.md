# Confy

[English](README.md) | [Chinese documentation](README.zh.md)

Confy is a Go configuration library built on Viper. Load local files, Nacos configuration, or environment variables into structs through the same binding API. It supports YAML, JSON, TOML, default values, explicit validation, encrypted fields, and configuration updates.

## Usage

### Installation

Requires Go 1.23 or later.

```sh
go get github.com/JsonLee12138/confy@latest
```

### Quick start

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

Run `go run .`, or override the port with `MYAPP_SERVER_PORT=9090 go run .`.

`Bind` applies `default` tags before decoding, preserving configured `0`, `false`, and empty strings. Each bind replaces the target from a fresh struct; failed decoding leaves the target unchanged. Move initial field values into default tags or configuration. Optional nested struct pointers need `default:"{}"` to initialize their defaults. `BindWithDefaults` is deprecated; use `Bind`.

### Choose a Provider

| Source | Setup | Updates |
| --- | --- | --- |
| Local | Default, or `local.New(...)` | Native file watching |
| Nacos | `nacos.New(...)` | Remote subscription and reconciliation |
| ENV | `env.New(schema, prefix)` | Explicit rebinding; no watcher |

**Local files**

The quick start uses the default Local Provider. To select it explicitly:

```go
import "github.com/JsonLee12138/confy/provider/local"

cfg, err := confy.New(confy.WithProvider(local.New(
    local.WithPath("config"),
    local.WithFile("config"),
    local.WithFileType(confy.YAML),
    local.WithLoadAll(false),
)))
```

Local files merge in this order, with later files taking priority:

```text
config.yaml -> config.local.yaml -> config.{env}.yaml -> config.{env}.local.yaml
```

Set `GO_ENV_MODE` to select `development`, `production`, or `test`. Local loading also supports environment aliases and mixed YAML, JSON, and TOML files. `WithLoadAll(true)` discovers other configuration files in the directory. A local file can inherit another file with `base: base.yaml`; child values override inherited values.

**Nacos**

Use the same `AppConfig` and `Bind` call from the quick start:

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
            NamespaceId: "", // Public namespace; otherwise use the namespace ID.
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

Handle `err`, defer `cfg.Close()`, then call `cfg.Bind(&app)`. No local configuration directory is needed. YAML is the default format; use `WithFormat(confy.JSON)` or `WithFormat(confy.TOML)` explicitly for other formats. The Data ID extension does not select the parser.

Inject connection credentials before constructing the Provider. Confy's dotenv loading cannot update options already collected by `nacos.New`. See the [Nacos guide](provider/nacos/README.md) for cache, logging, authentication, and transport details.

**Environment variables**

For bootstrap settings, bind ENV directly without loading files:

```go
var connection struct {
    Address string `mapstructure:"address"`
    Port    int    `mapstructure:"port"`
}
// Reads NACOS_ADDRESS and NACOS_PORT; use "" for no prefix.
err := confy.BindEnv(&connection, "NACOS")
```

Direct `BindEnv` does not apply defaults or business validation. Missing or empty variables produce zero values; errors leave the instance unchanged.

To use ENV through the regular Config API:

```go
import "github.com/JsonLee12138/confy/provider/env"

cfg, err := confy.New(confy.WithProvider(env.New(AppConfig{}, "MYAPP")))
// After checking err: defer cfg.Close(), then cfg.Bind(&app).
```

This path applies the usual `Bind` defaults and decryption rules. ENV keys come from the struct schema, including nested fields. ENV has no watcher; call `Bind` again to read changed variables. See the [ENV guide](provider/env/README.md) for supported schemas.

### Environment overrides and dotenv

For Local and Nacos, `server.port` maps to `MYAPP_SERVER_PORT` when using `WithEnvPrefix("MYAPP")`. Effective priority, from low to high, is source content, nonempty ENV, then explicit `cfg.Set`. ENV-only fields in file/remote configuration need declared placeholder keys; unrelated process variables are not discovered automatically.

Load dotenv files while creating Config:

```go
confy.New(confy.WithDotEnv(".env"))
confy.New(confy.WithDotEnvAuto())      // Current directory.
confy.New(confy.WithDotEnvAuto("etc")) // Specific directory.
```

Automatic discovery uses `.env`, `.env.local`, `.env.{env}`, and `.env.{env}.local`, including supported environment aliases. Missing files are skipped. Existing nonempty process variables are preserved; dotenv fills variables only while their current value is empty.

### Watching and validation

Enable watching on Local or Nacos when creating Config:

```go
confy.WithWatch(true),
confy.WithOnChange(func(event confy.Event) {
    // The bound target has been updated successfully.
}),
confy.WithOnError(func(err error) {
    // Handle refresh or binding errors.
}),
```

Watching starts after the first successful bind. Further binds do not add subscriptions: automatic updates keep targeting the first watched instance, while `Validate` checks the most recently bound instance. Synchronize application reads of any target being updated in the background. Callbacks run outside internal locks.

Business validation is explicit. Implement `Validate() error` on your struct and call `cfg.Validate()` after binding or when handling an update. `Bind` does not invoke it automatically. `ValidateType(&app)` separately checks top-level `required:"true"` fields and type compatibility.

Parse errors preserve the previous source. Binding or decryption errors preserve the previous target, while `Get` can expose the latest parseable source. `Close` stops delivery and releases supported Provider resources. Local watches the first selected file; changes to inherited files alone may not trigger a reload. Native Viper watching can stop after file removal or some atomic replacements on macOS; its native watcher remains process-scoped.

### Other APIs and options

Use `cfg.Get(key)` and `cfg.Set(key, value)` to read and override values. `Snapshot()` returns a settings map; `Restore(snapshot)` applies it as overrides, and `Export(path)` writes configuration to a file. Restore and Set do not automatically rebind the struct; call `Bind` when needed.

Encrypted fields use the `enc:AES_GCM:` prefix and require `WithEncryption("aes-gcm", key)`. Decryption happens during binding; source ciphertext remains in shared configuration. ENV values, snapshots, and exports can contain plaintext.

| Option | Purpose |
| --- | --- |
| `WithPath`, `WithFile`, `WithFileType`, `WithLoadAll` | Configure default Local loading |
| `WithProvider` | Select an explicit Local, Nacos, ENV, or custom source |
| `WithEnvPrefix` | Set the ENV override prefix |
| `WithDotEnv`, `WithDotEnvAuto` | Load dotenv files |
| `WithWatch`, `WithOnChange`, `WithOnError` | Configure updates and error handling |
| `WithEncryption` | Enable field decryption |

Select one source per Config. Explicit `WithProvider` conflicts with root Local options, including `WithLoadAll(false)`; configure that source through its own package options instead. Common errors are available as `confy.ErrXXX`; use `errors.Is` to check them. Direct ENV errors include `ErrEnvSchema`, `ErrEnvTarget`, and `ErrEnvDecode`.

### Runnable examples

From `example/`:

```sh
go run ./local
MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./env
NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_GROUP=APP_GROUP NACOS_DATA_ID=app.yaml go run ./nacos
```

For Nacos, prepare the configuration and export credentials first. See [example instructions](example/README.md). The original Gin demo remains available with `go run .`.

## Development

New adapters belong in `provider/<source>` and implement `provider.Provider`. Watching adapters also implement `provider.WatchProvider`: refresh the same shared Viper, release the shared lock, then notify Bind. Detailed contracts and development rules are in [AGENTS.md](AGENTS.md#provider-adapter-development).

```sh
go test ./...
go test -race ./...
go vet ./...
```

Cross-package E2E tests live in [`e2e/`](e2e/README.md) and call public APIs from the external `e2e_test` package. Run Local/ENV tests with `go test -race ./e2e`. Real Nacos tests use the same directory and require explicit connection settings and the `nacos_integration` build tag; see [test instructions and results](provider/TESTING.md).

## License

[MIT](LICENSE)

# Confy

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-%3E%3D%201.23-00ADD8?style=flat-square&logo=go" alt="Go Version" /></a>
  <a href="https://pkg.go.dev/github.com/JsonLee12138/confy"><img src="https://img.shields.io/badge/Go_Reference-pkg.go.dev-blue?style=flat-square&logo=go" alt="Go Reference" /></a>
  <a href="https://goreportcard.com/report/github.com/JsonLee12138/confy"><img src="https://goreportcard.com/badge/github.com/JsonLee12138/confy?style=flat-square" alt="Go Report Card" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg?style=flat-square" alt="License: MIT" /></a>
</p>

<p align="center">
  <a href="README.md">English</a> | <b>中文文档</b>
</p>

---

**Confy** 是基于 [Viper](https://github.com/spf13/viper) 的 Go 配置库。通过统一的绑定 API，将**本地文件**、**Nacos** 或**环境变量**加载到结构体——支持多环境级联、标签默认值、显式零值保留、热更新、AES-GCM 字段解密和显式校验。

## 目录

- [核心特性](#核心特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [选择 Provider](#选择-provider)
- [监听、加密与校验](#监听加密与校验)
- [可运行示例](#可运行示例)
- [开发](#开发)
- [许可证](#许可证)

## 核心特性

- **多来源统一接入** — 本地文件（YAML / JSON / TOML）、Nacos 2.x 远程配置、原生 ENV。
- **环境级联与模板继承** — `config.yaml → config.local.yaml → config.{env}.yaml → config.{env}.local.yaml`，以及 `base: parent.yaml` 继承（含循环检测）。
- **安全结构体绑定** — 先设 `default:"…"` 默认值再解码；显式 `0` / `false` / `""` 不丢失；成功原子替换目标，失败保留原值。
- **热更新** — 共享 `RWMutex`，`OnChange` / `OnError` 回调在锁外异步执行。
- **字段解密** — `enc:AES_GCM:` 前缀仅在绑定视图解密，源密文保持不变。
- **校验** — `Validator` 接口支持业务规则；`required:"true"` 标签检查必填项。
- **极简核心** — 根包不依赖 Nacos SDK。

## 安装

```sh
go get github.com/JsonLee12138/confy@latest
```

## 快速开始

创建 `config/config.yaml`：

```yaml
server:
  host: 127.0.0.1
  port: 8080
```

创建 `main.go`：

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
MYAPP_SERVER_PORT=9090 go run .   # 通过环境变量覆盖
```

`Bind` 先应用 `default` 标签再解码，保留配置中显式的 `0`、`false` 和空字符串。每次绑定在新结构体上解码，失败时目标不变。`time.Time` 字段支持 RFC3339 时间字符串。`BindWithDefaults` 已弃用，请直接使用 `Bind`。

## 选择 Provider

| 来源 | 接入方式 | 更新方式 |
| --- | --- | --- |
| Local | 默认来源，或 `local.New(…)` | 原生文件监听 |
| Nacos | `nacos.New(…)` | 远程订阅与一致性读取 |
| ENV | `env.New(schema, prefix)` | 主动重新绑定，不提供监听 |

**本地文件** — 快速开始使用默认的 Local Provider，也可以通过 `confy.WithProvider(local.New(…))` 显式选择。文件按以下顺序合并：`config.yaml → config.local.yaml → config.{env}.yaml → config.{env}.local.yaml`。通过 `GO_ENV_MODE` 选择 `development` / `production` / `test`（`dev`、`prod` 等别名同样支持）。使用 `base: base.yaml` 可继承其他文件。

**Nacos** — 沿用同样的 `AppConfig` 和 `Bind` 调用：

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

默认格式为 YAML；Data ID 扩展名不会自动选择解析器。连接凭据应在构造 Provider 前注入。详见 [Nacos 说明](provider/nacos/README.md)。

**环境变量** — 对于启动配置，可以直接绑定 ENV：

```go
var conn struct {
    Address string `mapstructure:"address"`
    Port    int    `mapstructure:"port"`
}
err := confy.BindEnv(&conn, "NACOS")   // 读取 NACOS_ADDRESS、NACOS_PORT
```

也可以通过完整 Config API：`confy.New(confy.WithProvider(env.New(AppConfig{}, "MYAPP")))`。详见 [ENV 说明](provider/env/README.md)。

## 监听、加密与校验

```go
confy.WithWatch(true),
confy.WithOnChange(func(event confy.Event) { /* 绑定目标已更新 */ }),
confy.WithOnError(func(err error) { /* 处理刷新错误 */ }),
```

首次成功 `Bind` 后开始监听。回调在内部锁外执行。解析失败保留旧来源，绑定失败保留旧目标。`Close` 停止交付并释放 Provider 资源。

加密字段使用 `enc:AES_GCM:` 前缀，通过 `WithEncryption("aes-gcm", key)` 启用。解密仅在绑定阶段发生，共享配置中保留源密文。

在结构体上实现 `Validate() error`，绑定后调用 `cfg.Validate()`。`Bind` 不自动调用业务校验。

## 可运行示例

```sh
go run ./example/local
MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./example/env
NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_GROUP=APP_GROUP NACOS_DATA_ID=app.yaml go run ./example/nacos
```

详见 [example/README.md](example/README.md)。原 Gin 综合示例仍可通过 `go run ./example` 启动。

## 开发

新增适配器放在 `provider/<source>`，实现 `provider.Provider`。支持监听的适配器还需实现 `provider.WatchProvider`。详见 [AGENTS.md](AGENTS.md#provider-adapter-development)。

```sh
go test -race ./...
go vet ./...
```

跨包 E2E 测试集中在 [`e2e/`](e2e/README.md)。真实 Nacos 测试需要 `nacos_integration` 构建标签，见 [provider/TESTING.md](provider/TESTING.md)。

## 许可证

[MIT](LICENSE)

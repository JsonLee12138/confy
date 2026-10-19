# Confy

[English](README.md) | [中文文档](README.zh.md)

Confy 是基于 Viper 的 Go 配置库，通过统一的绑定 API，将本地文件、Nacos 配置或环境变量加载到结构体。支持 YAML、JSON、TOML、默认值、显式校验、字段解密和配置更新。

## 使用

### 安装

需要 Go 1.23 或更高版本。

```sh
go get github.com/JsonLee12138/confy@latest
```

### 快速开始

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

执行 `go run .`，或者通过 `MYAPP_SERVER_PORT=9090 go run .` 覆盖端口。

`Bind` 先应用 `default` 标签，再解码配置，保留配置中显式的 `0`、`false` 和空字符串。每次绑定都从新结构体开始，成功后替换目标，解码失败时目标不变。初始字段值请写入默认值标签或配置。可选的嵌套结构体指针需要声明 `default:"{}"` 才会预先初始化其默认值。`BindWithDefaults` 已弃用，请使用 `Bind`。`time.Time` 和 `*time.Time` 字段支持 RFC3339 时间字符串，包括小数秒。

### 选择 Provider

| 来源 | 接入方式 | 更新方式 |
| --- | --- | --- |
| Local | 默认来源，或 `local.New(...)` | 原生文件监听 |
| Nacos | `nacos.New(...)` | 远程订阅和一致性读取 |
| ENV | `env.New(schema, prefix)` | 主动重新绑定，不提供监听 |

**本地文件**

快速开始使用默认的 Local Provider，也可以显式选择：

```go
import "github.com/JsonLee12138/confy/provider/local"

cfg, err := confy.New(confy.WithProvider(local.New(
    local.WithPath("config"),
    local.WithFile("config"),
    local.WithFileType(confy.YAML),
    local.WithLoadAll(false),
)))
```

本地文件按以下顺序合并，后者优先：

```text
config.yaml -> config.local.yaml -> config.{env}.yaml -> config.{env}.local.yaml
```

通过 `GO_ENV_MODE` 选择 `development`、`production` 或 `test` 环境。本地加载还支持环境别名，以及 YAML、JSON、TOML 混合文件。`WithLoadAll(true)` 会发现目录中的其他配置文件。使用 `base: base.yaml` 可继承其他本地文件，子文件覆盖继承值。每个文件（包括继承文件）优先按支持的扩展名选择解析器，扩展名无法识别时才使用配置的文件类型。

**Nacos**

沿用快速开始中的 `AppConfig` 和 `Bind` 调用：

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
            NamespaceId: "", // public 命名空间；其他命名空间填写 ID。
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

处理 `err` 后，通过 `defer cfg.Close()` 释放资源，并调用 `cfg.Bind(&app)`。此模式不需要本地配置目录。默认格式是 YAML，其他格式请显式指定 `WithFormat(confy.JSON)` 或 `WithFormat(confy.TOML)`；Data ID 扩展名不会自动选择解析器。

连接凭据应在构造 Provider 前注入。Confy 加载 dotenv 无法更新 `nacos.New` 已收集的选项。缓存、日志、鉴权和传输细节见 [Nacos 说明](provider/nacos/README.md)。

**环境变量**

对于连接参数等启动配置，可以直接绑定 ENV，无需加载文件：

```go
var connection struct {
    Address string `mapstructure:"address"`
    Port    int    `mapstructure:"port"`
}
// 读取 NACOS_ADDRESS 和 NACOS_PORT；空前缀 "" 表示无前缀。
err := confy.BindEnv(&connection, "NACOS")
```

直接 `BindEnv` 不应用默认值或业务校验。缺失或为空的变量使用零值，发生错误时实例不变。

也可以通过常规 Config API 使用 ENV：

```go
import "github.com/JsonLee12138/confy/provider/env"

cfg, err := confy.New(confy.WithProvider(env.New(AppConfig{}, "MYAPP")))
// 检查 err 后：defer cfg.Close()，再调用 cfg.Bind(&app)。
```

此方式应用统一的 `Bind` 默认值和解密规则。ENV 键来自结构体声明，包括嵌套字段。ENV 不提供监听，变量变化后请再次调用 `Bind`。支持的结构见 [ENV 说明](provider/env/README.md)。

### 环境变量覆盖与 dotenv

Local 和 Nacos 使用 `WithEnvPrefix("MYAPP")` 时，`server.port` 对应 `MYAPP_SERVER_PORT`。优先级从低到高依次为：来源配置、非空 ENV、显式 `cfg.Set`。文件或远程配置中仅由 ENV 提供的字段需要声明占位键，不会自动发现无关的进程变量。

创建 Config 时可以加载 dotenv：

```go
confy.New(confy.WithDotEnv(".env"))
confy.New(confy.WithDotEnvAuto())      // 当前目录。
confy.New(confy.WithDotEnvAuto("etc")) // 指定目录。
```

自动发现 `.env`、`.env.local`、`.env.{env}` 和 `.env.{env}.local`，也支持环境别名。不存在的文件会跳过。已有的非空进程变量保持不变；dotenv 仅在变量当前值为空时填充。

### 监听与校验

创建 Local 或 Nacos Config 时启用监听：

```go
confy.WithWatch(true),
confy.WithOnChange(func(event confy.Event) {
    // 绑定目标已成功更新。
}),
confy.WithOnError(func(err error) {
    // 处理刷新或绑定错误。
}),
```

首次成功绑定后开始监听。后续 Bind 不新增订阅：自动更新仍作用于首次监听的实例，`Validate` 检查最近一次绑定的实例。应用读取后台更新的目标时需要自行同步。回调在内部锁外执行。

业务校验由调用方显式执行：在结构体上实现 `Validate() error`，然后在绑定后或处理更新时调用 `cfg.Validate()`。`Bind` 不自动调用业务校验。`ValidateType(&app)` 另行检查顶层的 `required:"true"` 字段和类型兼容性。

解析失败保留旧来源。绑定或解密失败保留旧目标，但 `Get` 可能已返回最新的可解析来源。`Close` 停止交付并释放 Provider 支持的资源。Local 只监听首个选定文件，继承文件单独变化不一定触发重载。Viper 原生监听可能在文件删除或 macOS 的部分原子替换后停止，原生监听句柄随进程运行。

### 其他 API 与选项

通过 `cfg.Get(key)` 读取值，通过 `cfg.Set(key, value)` 设置覆盖值。`Snapshot()` 返回配置映射，`Restore(snapshot)` 将其作为覆盖值应用，`Export(path)` 将配置写入文件。Restore 和 Set 不自动重新绑定结构体，需要时再次调用 `Bind`。

加密字段使用 `enc:AES_GCM:` 前缀，并通过 `WithEncryption("aes-gcm", key)` 启用解密。解密发生在绑定阶段，共享配置中保留源密文。ENV 值、快照和导出内容可能包含明文。

| 选项 | 用途 |
| --- | --- |
| `WithPath`、`WithFile`、`WithFileType`、`WithLoadAll` | 配置默认 Local 加载 |
| `WithProvider` | 显式选择 Local、Nacos、ENV 或自定义来源 |
| `WithEnvPrefix` | 设置 ENV 覆盖前缀 |
| `WithDotEnv`、`WithDotEnvAuto` | 加载 dotenv 文件 |
| `WithWatch`、`WithOnChange`、`WithOnError` | 配置更新和错误处理 |
| `WithEncryption` | 启用字段解密 |

每个 Config 只选择一个来源。显式 `WithProvider` 与根包的 Local 选项冲突，包括 `WithLoadAll(false)`；请改用对应 Provider 包中的选项。通用错误通过 `confy.ErrXXX` 暴露，可使用 `errors.Is` 判断。直接 ENV 绑定的错误包括 `ErrEnvSchema`、`ErrEnvTarget` 和 `ErrEnvDecode`。

### 可运行示例

在 `example/` 目录执行：

```sh
go run ./local
MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./env
NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_GROUP=APP_GROUP NACOS_DATA_ID=app.yaml go run ./nacos
```

Nacos 需要先准备配置并导出凭据，详见 [示例说明](example/README.md)。原 Gin 综合示例仍可通过 `go run .` 启动。

## 开发

新增适配器放在 `provider/<source>`，实现 `provider.Provider`。支持监听的适配器还需实现 `provider.WatchProvider`：刷新同一个共享 Viper，释放共享锁，再通知 Bind。详细契约和开发规范见 [AGENTS.md](AGENTS.md#provider-adapter-development)。

```sh
go test ./...
go test -race ./...
go vet ./...
```

跨包 E2E 集中放在 [`e2e/`](e2e/README.md)，使用外部测试包 `e2e_test`，通过公开 API 调用不同包。执行 `go test -race ./e2e` 验证 Local/ENV。真实 Nacos 测试也在同一目录，需要显式连接配置和 `nacos_integration` 构建标签，见 [测试命令和结果](provider/TESTING.md)。

## 许可证

[MIT](LICENSE)

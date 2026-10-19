# ENV Provider

```go
import (
    "github.com/JsonLee12138/confy"
    "github.com/JsonLee12138/confy/provider/env"
)

// APP_SERVER_PORT maps to server.port.
var settings struct {
    Server struct {
        Port int `mapstructure:"port"`
    } `mapstructure:"server"`
}
err := confy.BindEnv(&settings, "APP") // also available as env.BindEnv
```

`BindEnv(instance, prefix)` 使用结构体注册 `Viper.BindEnv` 键，再通过 `Viper.Unmarshal` 解码到新实例；成功后替换目标，失败保留原值。不加载文件，不应用默认值标签或业务校验。prefix 不带末尾下划线，空字符串表示无前缀。缺失或空 ENV 使用零值。

作为配置来源可使用 `confy.New(confy.WithProvider(env.New(settings, "APP")))`，再调用 `cfg.Bind(&settings)`，此路径遵循 Config 的默认值、解密和显式 Validate 规则。不要启用 WithWatch：ENV Provider 不实现监听；每次 Load 返回独立的 Viper，ENV 按 Viper 语义在读取时查询。

仅注册结构体声明的键；支持 mapstructure 重命名、嵌套结构体/指针、值结构体 squash，忽略未导出字段和 `mapstructure:"-"`。递归结构体、指针 squash 和 remain 返回 `core.ErrEnvSchema`。目标必须是非 nil 结构体指针，否则返回 `core.ErrEnvTarget`；解码失败返回 `core.ErrEnvDecode`，不泄露 ENV 值。

错误统一定义在 `core/errors.go`，根包暴露相同值，可直接使用 `errors.Is(err, confy.ErrEnvDecode)` 等。原 `core/env_loader.go` 的实现已迁至本包，调用入口改为 `env.New`、`env.BindEnv` 或 `confy.BindEnv`。

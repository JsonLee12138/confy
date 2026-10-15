# 用户六项调整与验证

日期：2026-10-15。用户本轮指令覆盖原 Spec 的目录位置约束。

## 示例与 ENV

`example/main.go` 已恢复到 `5333862`，字节比较一致。独立程序为 `example/nacos/main.go`，不启用监听，只读取配置：输入连接参数与 ENV → NacosProvider.Load → 公共 ENV 覆盖 → BindWithDefaults/Validate → 输出服务器地址。示例不导入 Viper/defaults，也没有配置发布/写回接口。

旧示例的进程内副本发布逻辑已撤回。测试夹具的发布/删除仍仅用于隔离联调，和库/业务读取示例无关。

实际运行新示例，Nacos 源 `server.port=8080`，设置 `MYAPP_SERVER_PORT=9090`，输出 `server=127.0.0.1:9090`，退出码 0；注入的密码未打印。唯一测试 Data ID 已删除。

新示例使用的真实公共接口：

`example/nacos/main.go#L62-L62`

> ```go
> 62: 		confy.WithEnvPrefix("MYAPP"),
> ```

`example/nacos/main.go#L69-L71`

> ```go
> 69: 	if err = cfg.BindWithDefaults(&app); err != nil {
> 70: 		log.Fatal(err)
> 71: 	}
> ```

ENV 没有迁移到 Nacos 适配器内，仍由公共层执行；环境空值和未知键发现沿用原行为。真实集成测试新增 server.port 嵌套 ENV 检查，验证首次读取、更新和重连后都为 9090：

`provider/nacos/integration_test.go#L190-L192`

> ```go
> 190: 	if cfg.Get("server.port") != "9090" {
> 191: 		t.Fatal("initial Nacos read did not apply nested ENV override")
> 192: 	}
> ```

## core 契约与准确分派

之前的判断将任何非 LocalProvider 都视作 remote，无法准确表达 Nacos 与自定义来源。调整前真实代码：

`bind.go@97e91ee#L58-L58`

> ```go
> 58: func (c *Config) remote() bool { _, local := c.provider.(*LocalProvider); return !local }
> ```

现在输入 Provider → Type 明确标识 → switch 选择对应绑定规则 → 公共解码与候选事务继续共用。加载/监听调用始终经同一 core 契约，无 SDK 或来源专用 Bind 副本：

`core/provider.go#L12-L16`

> ```go
> 12: type Provider interface {
> 13: 	// Type identifies the implementation for explicit adapter dispatch.
> 14: 	Type() ProviderType
> 15: 	Load(context.Context) (*viper.Viper, error)
> 16: }
> ```

`core/provider.go#L19-L22`

> ```go
> 19: type WatchProvider interface {
> 20: 	Provider
> 21: 	Watch(context.Context) (Watcher, error)
> 22: }
> ```

`provider_adapter.go#L15-L25`

> ```go
> 15: 	switch provider.Type() {
> 16: 	case core.ProviderLocal:
> 17: 		// Preserve local explicit validation and historical multi-target behavior.
> 18: 		return bindingAdapter{}
> 19: 	case core.ProviderNacos:
> 20: 		return bindingAdapter{singleTarget: true, strictDecoding: true, validateUpdates: true, deduplicate: true}
> 21: 	default:
> 22: 		// Custom implementations use conservative candidate validation without
> 23: 		// being identified as Nacos.
> 24: 		return bindingAdapter{singleTarget: true, strictDecoding: true, validateUpdates: true, deduplicate: true}
> 25: 	}
> ```

Local/Nacos 均有编译期契约断言，core/provider_test.go 另验证根包类型别名兼容。自定义新 Provider 必须实现 Type；旧版 local New/WithPath/Bind 的调用方式不变。目前只有 switch 分派，没有引入注册表。

依赖检查通过：根包和 core 的生产依赖均无 Nacos SDK；Nacos 生产适配器依赖 core，不依赖根包。

## Config 状态清理与兼容性

旧 snapshot 字段在 Snapshot/Restore 中只有赋值，无读取用途；调用者本来就通过 Snapshot 返回值显式传给 Restore。调整前的一处无用写入：

`snapshot.go@97e91ee#L18-L18`

> ```go
> 18: 	c.snapshot = snapshot
> ```

现已删除该字段及两处赋值，保持 Snapshot/Restore 签名和结果。原 watchOnce 已由“成功订阅才标记 active”的生命周期取代，注册失败仍可重试。Config 保留确实需要的 v/opts/mu/bound，以及 Provider 和显式覆盖记录；新增监听/关闭状态分别归组：

`confy.go#L14-L23`

> ```go
> 14: type Config struct {
> 15: 	v         *viper.Viper
> 16: 	opts      *configOptions
> 17: 	mu        sync.RWMutex
> 18: 	provider  Provider
> 19: 	watch     watchState
> 20: 	lifecycle lifecycleState
> 21: 	overrides map[string]any
> 22: 	bound     any
> 23: }
> ```

`watch.go#L18-L25`

> ```go
> 18: type watchState struct {
> 19: 	watcher      Watcher
> 20: 	active       bool
> 21: 	target       any
> 22: 	defaults     bool
> 23: 	lastSettings map[string]any
> 24: }
> 25:
> ```

`watch.go#L27-L34`

> ```go
> 27: type lifecycleState struct {
> 28: 	ctx    context.Context
> 29: 	cancel context.CancelFunc
> 30: 	closed bool
> 31: 	once   sync.Once
> 32: 	err    error
> 33: }
> 34:
> ```

bound 保留给显式 Validate；watch.target 保留首次监听目标，以兼容原本地多次 Bind 的行为，不能直接合并。opts 中本地路径/格式/优先级选项仍由 LocalProvider 使用，未废弃其 API。

## 验证结果

- 原 main 与 `git show 5333862:example/main.go` 的字节比较通过。
- example Go 源码无 Viper/defaults import；新示例本机实跑及 8080 → ENV 9090 覆盖通过。
- Provider 类型分派、声明 Local 类型的自定义实现兼容规则、core 契约断言和根包别名测试通过。
- 原有本地/Provider/加密/默认值/快照/监听测试与 `go test -race ./...` 通过。
- 本机 r-nacos 真实 `-race -tags=nacos_integration` 通过，包括嵌套 ENV、热更新、失败回滚、断连重连和严格启动。
- root build/vet 与 example 独立 build/test 均通过。
- Mutation 检查：临时移除 Local switch 分支后，声明 Local 类型的行为回归测试按预期失败；已恢复代码。

上一轮记录的鉴权/TLS 环境限制及依赖漏洞待处理项仍保留；本轮没有改变这些发布条件。

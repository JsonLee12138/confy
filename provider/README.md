# 配置来源契约

`provider/provider.go` 和 `provider/types.go` 是 Provider、Watcher、Change、Event、FileType 的统一定义；`core` 保留类型别名及 继承等辅助功能和公共错误。实现位于 `provider/local`、`provider/nacos` 和 [`provider/env`](env/README.md)。ENV 通过结构体注册 Viper 的 ENV 键，不提供监听；每次 Load 创建独立 Viper。

- `Provider.Type()` 明确声明来源类型，`Load(ctx)` 加载完整配置。Local/Nacos 重复 Load 和监听更新均刷新同一个 Viper；Config 直接持有该实例。
- `WatchProvider.Watch(ctx, onChange, onError)` 注册监听，可报告的注册失败同步返回；注册失败后允许重试。
- `WatchProvider` 包含 `SynchronizedProvider`：`ConfigMutex() *sync.RWMutex` 必须返回稳定、非 nil 的共享锁。Provider 更新 Viper 和 Config 的 Get/Set/Bind/Snapshot/Export 共用它。直接读取 Load/Change.Config 返回的 Viper 时也必须持有此锁。
- 来源在后台串行调用回调，回调不得在持有来源锁或共享配置锁时执行，允许调用 Config.Get/Set/Bind/Close。回调应及时返回。
- `Watcher.Close()` 必须幂等，停止后续交付；不等待正在执行的回调，因此可以在回调内调用。Context 取消同样停止交付。Local 的 Viper 原生监听句柄仍随进程运行。
- Nacos 将 SDK 通知合并为有界唤醒信号，调用共用 refresh 重新读取并更新共享 Viper；非终止错误通过 onError 报告，后续仍可恢复。重复 Watch 沿用活动订阅的原 Context 和回调。
- `Change.Config` 必须是 Load 返回的同一 Viper，表示更新已经完成；不再传递独立候选。Local 使用私有 Viper 承接原生文件通知，再调用 Load 重建继承/多文件配置并刷新共享 Viper，避免原生无锁写入与 Config 读取竞争。原生监听范围仍为首个选定文件。
- 源配置使用 Viper 的配置层完整替换，清除已删除字段；ENV 使用 AutomaticEnv，显式 Set 保存在 Viper 自身覆盖层，无需额外覆盖表。Bind 通过闭包持有 instance，首次绑定与更新共用解码流程，再调用业务回调；不按 Provider.Type 分派策略，也不保存配置快照做去重；根包没有独立 watch.go 或候选提交层。
- 解析失败保留共享 Viper；解码/解密失败保留上次成功绑定的目标，Viper 已是最新可解析内容。解密只发生在临时绑定视图中，不通过 Set 污染共享 Viper；Get/Snapshot/Export 保留源密文（ENV 注入的秘密仍可能是明文）。
- Config.Close 释放 Watcher 和实现 io.Closer 的 Provider。应用直接读取自动更新结构体时仍需自行同步。

`Bind` 在新的结构体上执行一次 defaults.Set，再解码覆盖，成功后替换目标。`BindWithDefaults` 已弃用，作为直接调用 Bind 的兼容入口保留。传入结构体只作为结果容器，不保留调用前的预设值；解码失败时目标不变。显式的 0、false 和空字符串保留；删除字段后重新使用标签默认值。嵌套可选结构体指针需声明 `default:"{}"` 才会预先初始化；集合合并与 null 沿用 Viper/mapstructure 的解码语义。

自定义监听 Provider 需要实现 ConfigMutex、原位替换源配置后再通知；仅把新 Viper 放进 Change.Config 的旧实现不再适用。旧 Next 式 Watcher 还需迁移为回调式 Watch 并实现 Close；core 类型别名不保留旧方法签名。

所有 Provider 使用相同绑定规则：业务 Validator 只由显式 cfg.Validate 调用，Bind 和更新不会自动执行。允许再次 Bind 其他目标，不新增订阅；自动重绑仍作用于首次监听捕获的目标，Validate 检查最后一次 Bind 的目标，不提供多目标广播。

通知去重由 Provider 自己实现。Nacos 比较刷新后的有效设置（包含 ENV/Set），忽略无变化的轮询和重复 SDK 通知；Bind 对每次收到的通知都执行重绑。解码器与 defaults.Set 返回的目标错误直接返回，只有反射和默认值所需的非 nil 指针检查留在绑定入口。密文字段对所有来源统一要求启用解密并提供正确密钥；不再有 Local 的额外豁免。ErrBoundTarget 保留作 API 兼容，不再由 Bind 返回。

新增适配器请遵循 [开发规则](../AGENTS.md#provider-adapter-development)，可运行示例见 [example/README.md](../example/README.md)。

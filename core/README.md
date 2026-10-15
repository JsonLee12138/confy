# 配置来源统一规范

`core` 定义 Local、Nacos 和后续来源共用的契约，不依赖根包实现或 Nacos SDK。

## Provider

- `Type() ProviderType`：明确声明 Local/Nacos/Custom 类型，不能用“不是 Local”推断为 Nacos。当前公共协调器通过 switch 分派来源差异规则，未实现注册表。
- `Load(context.Context) (*viper.Viper, error)`：返回完整的、独立的配置实例；成功后所有权转交给 Config。不得继续异步修改该实例，不得返回 nil,nil。
- Provider 只负责来源加载与格式解析。ENV、运行时 Set、字段解密、Bind/defaults 及业务校验由公共协调器处理。
- 每个 Provider 由单个 Config 管理。如果实现 io.Closer，Config.Close 负责释放；SDK 外部客户端借用不属于当前 API。

## WatchProvider / Watcher

- 监听是可选能力：WatchProvider 在 Provider 基础上实现 `Watch(ctx) (Watcher,error)`。注册失败同步返回错误，不返回伪成功句柄；允许后续重试。
- Watcher.Next 由单个消费者调用，阻塞等待完整 Change。每个 Change.Config 独立拥有，不返回字段增量，不触碰业务结构体。
- 允许合并尚未消费的全量事件，但必须最终收敛。SDK 回调只能做有界操作。
- 非终止错误拒绝一次更新，后续 Next 可以继续；终止用 context.Canceled/context.DeadlineExceeded/io.EOF。
- Close 幂等，释放自有订阅并解除 Next 阻塞。Watch Context 或 Next Context 取消也必须能解除等待；已运行的无 Context SDK 请求只能由 SDK 超时约束。

## Change / Event 与公共层

- Change.Config 是完整候选；Event.Name/Op 只携带来源标识与事件操作，不含配置内容、凭据或密钥。
- 公共层按 Provider < 非空 ENV < 显式 Set 组合候选，使用共用绑定流程，然后按 Provider 类型决定自动业务校验等差异。
- 候选失败保留当前有效值；成功提交后解锁再调用 OnChange，失败交给 OnError。
- 本地保留原有显式 Validate 语义和监听首个绑定目标的兼容行为。Nacos 限制单绑定目标，候选提交前校验。
- Config 内部锁不等于应用直接读取自动更新结构体的同步保证。

根包的 Provider/WatchProvider/Watcher/Change/Event/FileType 为这里的别名，原有 `confy.New/WithPath/Bind/BindWithDefaults` 使用方式不变。自定义新 Provider 必须实现 Type 和 Load。

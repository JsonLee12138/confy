### 严苛的技术调研与方案引用规范（违者视为调研失败）：
当你被要求“参考某代码/模块/库”编写技术方案或分析报告时：
1. 【禁止空洞引用】：严禁仅罗列类名、文件名或通用概念（如“参考了 UserModule”）。
2. 【行号级代码快照（Code Snippet with Line Numbers）】：
   - 提到的每一个参考结论，必须用引用块展示真实代码片段，并标明文件路径与精确行号（如 `src/auth/service.py#L45-L62`）。
3. 【调用链与数据流实质证据】：
   - 必须明确写出：输入参数是什么 -> 经过了哪个核心判断分支 -> 发生了什么副作用（如写 DB/发 MQ） -> 返回了什么。
4. 【批判性对比（Diff Analysis）】：
   - 必须回答：该参考模块的实现存在哪些缺陷或边界条件限制（如高并发下的死锁风险、缺少超时控制）？新方案是在哪一行代码逻辑上做出的改进？

<a id="provider-adapter-development"></a>

# Provider 适配器开发规则

## 目录与依赖

新适配器放在 `provider/<source>/`，统一实现 `provider` 包的接口。实现代码依赖 `provider`、`core` 和来源 SDK，不导入根包 `confy`，避免循环依赖。来源独有的选项放在自己的包中，通用错误定义在 `core/errors.go`，根包 `errors.go` 通过变量别名暴露相同错误值；来源特有的错误可以留在适配器包中。

## 加载契约

真实接口位于 [`provider/provider.go#L15-L19`](provider/provider.go#L15-L19)：

> ```go
> // L15-L19
> type Provider interface {
>     // Type identifies the implementation for explicit adapter dispatch.
>     Type() ProviderType
>     Load(context.Context) (*viper.Viper, error)
> }
> ```

`New` 收集选项；`Load(ctx)` 检查上下文、初始化连接、读取来源、解析并返回非 nil Viper，失败返回错误。`Type` 仅标识来源，Bind 不按来源类型分派。加载期间创建的客户端和文件若由 Provider 持有，实现 `io.Closer`，保证 Close 幂等；初始化失败也需清理。一个 Provider 实例归一个 Config 使用。

文档型来源初始加载和监听更新必须复用同一个写入方法。真实 Nacos 写入片段位于 [`provider/nacos/loader.go#L51-L60`](provider/nacos/loader.go#L51-L60)：

> ```go
> // L51-L60
> func (p *Provider) writeConfig(ctx context.Context, content string) error {
>     p.configMu.Lock()
>     defer p.configMu.Unlock()
>     if err := ctx.Err(); err != nil {
>         return err
>     }
>     if err := p.v.ReadConfig(strings.NewReader(content)); err != nil {
>         return p.safe(ErrDocument)
>     }
>     return nil
> }
> ```

数据流：远程文本 `content` → 上下文检查 → 在共享锁下调用 `ReadConfig` 替换 Viper 源配置层 → 成功返回 nil；失败返回安全错误，旧源配置保留。使用 `ReadConfig` 替换完整文档可清除删除字段并保留 ENV/显式 Set 优先级。不要用逐字段 `Set` 刷新来源，否则删除字段残留且覆盖 ENV。

ENV 来源不读取文件：根据结构体注册 Viper.BindEnv 键并使用 AutomaticEnv。ENV Provider 当前每次 Load 返回独立 Viper、不提供监听；直接 BindEnv 还会 Unmarshal 到临时目标，成功后替换用户实例。

## 可选监听契约

真实签名位于 [`provider/provider.go#L23-L39`](provider/provider.go#L23-L39)：

> ```go
> // L23-L26
> type SynchronizedProvider interface {
>     Provider
>     ConfigMutex() *sync.RWMutex
> }
> // L32-L39
> type WatchProvider interface {
>     SynchronizedProvider
>     Watch(ctx context.Context, onChange func(Change), onError func(error)) (Watcher, error)
> }
> ```

1. ConfigMutex 返回稳定且非 nil 的锁，Provider 写入与 Config 读写共用它。Viper 自身不保证并发读写安全。
2. 初始 Load 和后续刷新使用同一 Viper。监听通知只说明来源发生变化：读取最新完整内容，调用共用写入方法，成功后释放锁，再异步、串行执行 onChange。Change.Config 必须等于 Load 返回的实例。
3. Watch 注册时 Config 持有共享锁，因此注册过程不得同步调用回调或获取 ConfigMutex；读取/刷新工作应放到后台。注册失败同步返回错误。重复注册的行为应明确定义。
4. 刷新失败调用 onError；可恢复错误继续监听，解析失败保留旧源配置。不要打印配置或凭据。通知去重、通知合并、重连和关闭由 Provider 负责。
5. Watcher.Close 必须幂等，不等待当前回调结束，支持回调内 Close；context 取消停止通知。监听与 Close 的竞争必须测试。
6. 监听实现仅刷新 Viper 并通知。Bind 统一执行 defaults、解密、Unmarshal 和成功后的业务回调；不适配来源，不去重，不自动调用业务 Validate。不支持监听的来源不要实现 WatchProvider；开启 WithWatch 时 Bind 返回 ErrWatchUnsupported。

## 边界与改进点

`ReadConfig` 只证明文本可解析，无法保证结构体字段类型、密钥或业务规则有效。Nacos 对应改进在上述 `loader.go:57`：先更新 Viper，Bind 再解码；Bind 失败保留用户实例，但 Get 已反映最新可解析来源。业务校验由用户显式调用 Validate。不要把这些阶段描述为一个会回滚来源的事务。

来源 SDK 回调可能乱序；Nacos 通过合并唤醒信号、串行重新读取最新内容处理，不直接提交旧回调文本。其他适配器要按来源实际保证说明版本顺序与中间版本是否可能合并。

Local 使用 Viper 原生文件监听，当前只监听首个文件；继承/其他文件单独变化不保证触发刷新。Viper 在 Remove 事件后停止原生监听，macOS 上某些原子替换会触发此行为。Close 停止交付，原生监听句柄仍随进程运行。自定义适配器若要求删除重建或原子替换后继续监听，需要自己实现并测试该能力。

## 验证与示例

- 提供 Load → confy.New(WithProvider(p)) → Bind → Get/Set 的可运行示例。
- 覆盖默认值和显式零值、删除字段、ENV/Set 优先级、失败保留绑定目标、恢复及 Close。
- 监听适配器用真实来源验证更新、注册间隙、重复通知、断线恢复；fake/编译成功不能代替真实服务验证。
- 运行 `env -u GOROOT go test -race ./...` 和 `env -u GOROOT go vet ./...`。
- Local/ENV E2E：`env -u GOROOT go test -race -run '^TestE2E' -v .`。
- Nacos 的真实测试命令和必要环境变量见 [provider/nacos/README.md](provider/nacos/README.md)。只对独立测试 Group/Data ID 写入或删除。
- 三种现有来源的可运行示例见 [example/README.md](example/README.md)。

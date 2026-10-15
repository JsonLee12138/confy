# Local / Nacos 实施与验收记录

日期：2026-10-15（Asia/Shanghai）。规格：`docs/SPEC.md`；实施授权来自用户随后要求“根据 Spec 进行开发”的指令。分支：`codex/nacos-provider`。

## 真实环境与结果

- 服务：现有 Docker `rnacos`，镜像 `qingpan/rnacos:stable`；运行二进制读回 `rnacos 0.8.7`。镜像 ID `sha256:731cefc0ae2ca1f52d6a18b5b7b4c1d432e1d0ef5d2c5910add32bef9bd69fad`。
- 目标：`127.0.0.1:8848` HTTP、`9848` gRPC，public Namespace，专用 `CONFY_INTEGRATION` Group。
- SDK：官方 `v2.3.6-0.20260917093256-70993e7b0590`，固定提交 `70993e7b0590aee5299849315d01a92fd346f8ab`；Go module 最低声明仍为 1.23.0。
- 只创建唯一 `confy-integration-<nanoseconds>.yaml` 夹具，退出时删除；未改业务配置或重启共享服务。断连只关闭测试代理自身连接。
- 已实测：Load/Bind 间更新、连续发布最终收敛、字段删除、ENV/Set/defaults、AES-GCM 字段、损坏密文/解析失败/空内容回滚、整份删除后恢复、HTTP/gRPC 链路与断连重连、Close 清理。
- 最终读回：远端 `name=offline, port=9`；有效配置保留显式 `Set(port,99)`，回调确认 `name=offline`。
- 严格启动：远端不可达，同时预置 snapshot/failover，New 仍返回错误；失败 New 创建的私有目录已清理。
- 测试秘密通过 ENV 和字段解密进入有效配置；扫描自有 SDK 文件未发现其明文。SDK 原始缓存允许持有 Nacos 源文档，不能宣称任意远程明文秘密都不落盘。

## 验收覆盖与边界

| 范围 | 结果与证据 |
|---|---|
| AC-01–04 | 原有回归通过；YAML/JSON/TOML 本地与 Provider 等价测试、nil/重复/冲突 Option 测试通过 |
| AC-05–07 | Bind 无 fsnotify/WatchConfig/SDK import；首次订阅、失败重试、禁用无订阅、本地普通写入/原子替换/多文件优先级测试通过 |
| AC-08–12 | ENV/Set、并发提交、删除残留、Validator/映射/密文拒绝、恢复后重绑通过；首次 Validate 仍显式 |
| AC-13–15 | 回调重入 Get/Set/Bind/Close、内部访问 race、真实重连 race、Watch Context/Close 解阻塞、自有资源清理通过 |
| AC-16–18 | 真实旧缓存故障启动拒绝、全份删除/空文档拒绝、核心依赖无 SDK 通过 |
| AC-19 | 夹具中 ENV/解密明文不出现在 SDK 文件；SDK 日志禁用。SDK 仍缓存源文档；Export/Snapshot 不脱敏。详细限制见适配器 README |
| AC-20 | HTTP/gRPC、读取/更新/重连通过；当前 OpenAPI 不强制鉴权，未验证鉴权允许/拒绝和 TLS 证书链，因此不能声称完整生产接入验收通过 |

## 测试命令

- `env -u GOROOT go build ./...`
- `env -u GOROOT go test ./...`
- `env -u GOROOT go test -race ./...`
- `env -u GOROOT go vet ./...`
- `env -u GOROOT go list -deps .`：检查输出无 `nacos-sdk-go`。
- `example/` 内 `go build ./...` 和 `go test ./...`；独立 module 已 tidy。
- 显式注入 `NACOS_INTEGRATION=1 NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_NAMESPACE_ID= NACOS_TEST_GROUP=CONFY_INTEGRATION` 后运行 `go test -race -tags=nacos_integration -count=1 -timeout=180s -v ./provider/nacos/...`。
- Mutation 检查：暂将重复 Provider 判断从 `> 1` 改为 `> 2`，`TestProviderOptions` 按预期失败；已恢复原文件。

## 源码快照、数据流与差异

### 本地加载接缝

原基线输入 Options → loadAll 分支决定文件发现 → 无文件返回 ErrNoConfigFiles；它会迫使所有来源经过本地文件发现：

`confy.go@5333862#L44-L49`

> ```go
> 44: 	configPaths := getConfigFilePaths(o)
> 45: 	if o.loadAll {
> 46: 		configPaths = getAllConfigFilePaths(o)
> 47: 	}
> 48: 	if len(configPaths) == 0 {
> 49: 		return nil, fmt.Errorf("%w: in path %s", ErrNoConfigFiles, o.basePath)
> ```

现在输入 Options → 默认选择 LocalProvider 或显式 Provider → Load 转移独立 Viper 所有权 → 公共 ENV 处理。原文件发现/继承/优先级循环保留在 LocalProvider，没有让远程依赖本地目录：

`confy.go#L62-L66`

> ```go
> 62: 	if provider == nil {
> 63: 		provider = &LocalProvider{opts: o}
> 64: 	}
> 65: 	ctx, cancel := context.WithCancel(context.Background())
> 66: 	v, err := provider.Load(ctx)
> ```
`local_provider.go#L41-L41`

> ```go
> 41: 		resolved, err := resolveBaseInheritance(configPath, o.fileType, make(map[string]bool))
> ```

本地监听只固定首次成功 Load 选中的第一个文件；监听其父目录是为了接收该文件的原子替换，事件过滤不会扩大为监听其他配置文件。重新加载仍走完整合并算法。

### 候选事务与回调

输入完整候选 → compose 重应用 ENV/Set → 新建零值临时目标 → decode/defaults → 远程 Validator 分支 → 全部成功才写入目标与有效 Viper → 解锁后 OnChange。旧实现直接写业务对象，缺少拒绝后保留当前有效配置的事务边界；新实现的具体改进如下：

`watch.go#L102-L117`

> ```go
> 102: 	temporary := reflect.New(target.Elem().Type())
> 103: 	err := c.decode(candidate, temporary.Interface(), c.watchDefaults)
> 104: 	if err == nil && c.remote() {
> 105: 		if validator, ok := temporary.Interface().(Validator); ok {
> 106: 			if validator.Validate() != nil {
> 107: 				err = ErrCandidate
> 108: 			}
> 109: 		}
> 110: 	}
> 111: 	if err != nil {
> 112: 		c.mu.Unlock()
> 113: 		return ErrCandidate
> 114: 	}
> 115: 	target.Elem().Set(temporary.Elem())
> 116: 	c.v = candidate
> 117: 	c.lastSettings = settings
> ```
`watch.go#L121-L124`

> ```go
> 121: 	c.mu.Unlock()
> 122: 	if fn != nil {
> 123: 		fn(change.Event)
> 124: 	}
> ```

失败不会修改当前有效实例/绑定目标，OnError 只输出安全阶段。边界：应用直接读取绑定目标仍需同步；Validator 必须无副作用且不重入同一 Config。Gin 示例改为 Snapshot 独立解码，再原子发布视图。

### SDK race 与版本选择

v2.3.5 的真实 race 检查失败：输入重连事件 → 成功建连分支写 currentConnection → 并发通知线程读取该连接 → race 检测返回失败。两处真实代码：

`nacos-sdk-go/v2@v2.3.5/common/remote/rpc/rpc_client.go#L449-L449`

> ```go
> 449: 			r.currentConnection = connectionNew
> ```
`nacos-sdk-go/v2@v2.3.5/common/remote/rpc/rpc_client.go#L487-L487`

> ```go
> 487: 	logger.Infof("%s notify %s event to listeners , connectionId=%s", r.name, event.toString(), r.currentConnection.getConnectionId())
> ```

选定上游固定提交后，连接字段改为原子存取，真实 reconnect race 检查通过。没有改 vendor/module cache，也没有降低 race 标准：

`nacos-sdk-go/v2@70993e7/common/remote/rpc/rpc_client.go#L103-L103`

> ```go
> 103: 	currentConnection           atomic.Value // stores IConnection
> ```
`nacos-sdk-go/v2@70993e7/common/remote/rpc/rpc_client.go#L119-L119`

> ```go
> 119: 	v := r.currentConnection.Load()
> ```
`nacos-sdk-go/v2@70993e7/common/remote/rpc/rpc_client.go#L129-L129`

> ```go
> 129: 	r.currentConnection.Store(conn)
> ```

[上游固定源码](https://github.com/nacos-group/nacos-sdk-go/blob/70993e7b0590aee5299849315d01a92fd346f8ab/common/remote/rpc/rpc_client.go#L103)，[原子连接修复提交](https://github.com/nacos-group/nacos-sdk-go/commit/c9be203caf4147691703c853de5338e4e3c306c6)。版本选择不代表整个 SDK/间接依赖不存在其他问题。

### 通知收敛与缓存限制

实际压力测试曾出现删除后重建未及时通知，不能只用 ListenConfig 返回 nil 作为最终收敛证据。输入 SDK 通知仅有界入队 → Next 读取远端当前文档 → 空闲时 5 秒一致性核对、读失败 1 秒重试 → 解析完整候选返回公共协调器；同配置/同覆盖状态去重，不伪造重连成功事件。改进点为实际计时分支：

`provider/nacos/watcher.go#L75-L80`

> ```go
> 75: 	delay := 5 * time.Second
> 76: 	if w.retry {
> 77: 		delay = time.Second
> 78: 	}
> 79: 	timer := time.NewTimer(delay)
> 80: 	defer timer.Stop()
> ```

这额外产生空闲时每 5 秒一次 SDK 读取；它不保证每个中间版本都交付，也不替代 SDK 的长连接、鉴权刷新与协议重连。

严格启动输入 ClientConfig.CacheDir → 创建新的 0700 子目录 → 禁止 snapshot fallback → SDK 读取；旧缓存不进入本实例，失败不回退业务本地文件。改进点：

`provider/nacos/nacos.go#L97-L97`

> ```go
> 97: 	dir, err := os.MkdirTemp(parent, "confy-nacos-")
> ```
`provider/nacos/nacos.go#L110-L110`

> ```go
> 110: 	config.DisableUseSnapShot = true
> ```

限制：SDK 仍写原始源内容；Close 删除自有目录，进程崩溃可能留下私有目录。SDK 日志是进程全局的，适配器使用 non-recording logger，也会影响进程内其他官方 SDK 客户端。应用不能后续换回会记录内容的 SDK logger 并继续声称本保护成立。上述行为已在 README 明示。

## 依赖安全扫描（未通过）

额外执行 `GOTOOLCHAIN=go1.25.2 go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...`，未修改项目 go/toolchain 声明。原 Go 1.23.0 环境运行扫描器遇到 macOS LC_UUID/类型加载工具错误，临时使用另一工具链完成扫描。

扫描结果：32 条符号层漏洞报告涉及 `golang.org/x/net@v0.33.0` 和 Go 标准库；另有 20 条导入包层、37 条 module 层报告。静态符号可达性不等于逐项可利用性证明，但不能记为安全检查通过。例：`GO-2026-6617` 报告 x/net 修复版本 v0.60.0；该版本/标准库修复涉及后续 Go 基线与依赖升级，应独立确认兼容性后处理。本次未静默提高项目最低 Go 版本，也未做发布。

功能开发和当前本机联调已有证据；生产鉴权/TLS 与依赖安全放行仍未完成。

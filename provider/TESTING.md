# Provider 端到端验证

验证日期：2026-10-19。目标是当前工作区源码及本地 `rnacos` 容器，HTTP `127.0.0.1:8848`、gRPC `127.0.0.1:9848`。

## 可重复运行

```sh
# e2e 外部包：真实文件和进程 ENV，跨包验证完整用户 API 链路。
env -u GOROOT go test -race -count=1 -v ./e2e -run '^TestE2E'
env -u GOROOT go test -race -count=1 ./...
env -u GOROOT go vet ./...

# 由运行环境提供 NACOS_USERNAME、NACOS_PASSWORD；不将密码写入仓库。
export NACOS_INTEGRATION=1
export NACOS_SERVER_ADDR=127.0.0.1:8848
export NACOS_NAMESPACE_ID=
export NACOS_TEST_GROUP=CONFY_INTEGRATION
env -u GOROOT go test -race -tags=nacos_integration -count=1 -timeout=180s -v ./e2e -run '^TestIntegration'

# example 是独立 Go 模块。
cd example
env -u GOROOT go test ./...
env -u GOROOT go run ./local
env -u GOROOT MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./env
env -u GOROOT NACOS_GROUP=CONFY_EXAMPLES NACOS_DATA_ID=confy-example.yaml MYAPP_SERVER_PORT=9090 go run ./nacos
```

全部跨包测试集中在 `e2e/`，使用 `e2e_test` 包，仅调用 `confy` 与 Provider 的公开接口；普通单元测试保留在实现包中。新增回归还覆盖混合 YAML/TOML 文件及继承的监听重载、ENV 的 time.Time/时间指针、非法时间保留目标、duration/字符串和数字切片兼容，以及解密绑定视图中的时间字段。

## 已通过的行为

| 来源 | 实际验证 |
| --- | --- |
| Local | 真实文件及 base 继承 → New/Bind；嵌套 ENV 覆盖；显式 Set；零值和默认值；文件原位修改 → 自动重绑；删除字段清除；无效 YAML 保留旧源及目标；恢复；Close 后拒绝 Bind |
| ENV | 真实进程变量 → confy.BindEnv 及 env.New/Config.Bind；嵌套键；零值、空值和默认值；修改 ENV 后显式重绑；Set 优先级；错误不改变目标；恢复；开启监听返回 ErrWatchUnsupported |
| Nacos | 用户提供的账号登录；独立 Data ID 真实发布/读取；初始加载与订阅间隙；连续更新；ENV/Set 优先级；解密；删除字段；解析/字段类型/解密错误及恢复；显式 Validate；删除配置与恢复；断开测试自有代理后重连；拒绝旧缓存启动；私有缓存清理 |
| Nacos 格式 | YAML、JSON、TOML 各自真实发布 → Load/Bind → 发布更新 → 自动重绑，确认字段删除和默认值 |

根模块全量 race、vet、example 模块编译及三个示例实际执行均通过。`go doc Config.BindWithDefaults` 和 `go doc ConfigInterface` 已显示独立 `Deprecated:` 段落；尚未发布新版本，线上包文档需要抓取对应版本后才更新。

Nacos 自动测试总耗时约 27 秒。临时 `confy-integration-*`、`confy-format-*` Data ID 在测试结束后删除；只断开测试自有的 TCP 代理，不停止共享 Nacos。

## 控制台中保留的示例

public Namespace（空 namespace ID）、`CONFY_EXAMPLES` Group 中保留三份非敏感配置：

| Data ID | 格式 | 源端口 |
| --- | --- | --- |
| confy-example.yaml | YAML | 8080 |
| confy-example.json | JSON | 8081 |
| confy-example.toml | TOML | 8082 |

三份均已通过 HTTP 发布并逐字读回核对。Nacos 示例使用 YAML；JSON/TOML 自动测试显式指定 WithFormat。测试前已有的 `GENAIW_GROUP/genaiw-agent.yaml` 未被写入或删除，测试后清单确认仍在。

## 已知边界

- 本地服务提供明文 HTTP/gRPC。已验证登录成功，但不带 token 读取本次新建的示例配置仍返回 HTTP 200，说明该服务未强制此配置读取接口鉴权；不能据此宣称生产鉴权或 TLS 已通过。
- Local 使用 Viper 原生监听：本次 macOS 原子替换文件的尝试未触发预期更新；原位写入全部通过。Viper 收到 Remove 后会结束监听，当前不承诺删除重建或所有原子保存方式都能持续监听，详见 [开发规则](../AGENTS.md#provider-adapter-development)。
- ENV 没有变更通知；修改环境变量后需主动重新 Bind。
- 不保证每个中间 Nacos 版本都交付，测试等待指定最终版本；错误通知没有版本号，不能把任意下一条错误视作最新发布的处理完成证明。

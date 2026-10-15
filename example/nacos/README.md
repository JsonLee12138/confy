# Nacos 读取示例

原 `example/main.go` 保持原样。本目录提供独立程序，只读取配置，不发布或写回 Nacos；仅使用 Confy 的 BindWithDefaults/Validate/WithEnvPrefix API。

在 Nacos 中准备 `app.yaml`（默认 DEFAULT_GROUP、public Namespace）：

```yaml
server:
  host: 127.0.0.1
  port: 8080
database:
  password: ""
```

从 `example/` 运行：

```sh
NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_DATA_ID=app.yaml MYAPP_SERVER_PORT=9090 go run ./nacos
```

输出 `server=127.0.0.1:9090`，说明 ENV 覆盖了 Nacos 的 8080。`MYAPP_DATABASE_PASSWORD` 可覆盖 password 空占位键，示例不打印密码。

可通过 `NACOS_NAMESPACE_ID`、`NACOS_GROUP`、`NACOS_USERNAME`、`NACOS_PASSWORD` 传入连接配置。凭据从部署环境注入。此程序只做一次加载，未开启监听，因此可以直接读取已绑定结构体；需要热更新时仍应同步应用对绑定目标的并发访问。

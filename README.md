# mbot2

`mbot2` 是一个小型 MQTT 设备管理示例，由 MQTT Broker 和设备客户端两个独立的 Go 模块组成：

- **mkrab**：支持账号认证、角色 ACL、消息持久化和 Web 管理界面的 MQTT Broker。
- **mbot**：采集主机信息、上报模拟温度，并通过 MQTT 接收控制命令的设备客户端。

```text
控制端 ──发布命令──> mkrab ──转发命令──> mbot
控制端 <─查看消息── mkrab <─温度/结果── mbot
                         │
                         └── SQLite 审计与 Web 管理界面
```

> [!WARNING]
> `mbot` 当前允许 MQTT 消息触发任意 Shell 命令，`mkrab` 的 Web 管理接口当前也没有身份认证。项目适合本地开发和可信网络测试；用于生产环境前，请先完成命令白名单、Web 身份认证、TLS 和最小权限 ACL。

## 功能概览

### mkrab Broker

- MQTT over TCP、TLS、WebSocket 和 WSS，可分别启停
- SQLite 持久化，GORM 自动建表和软删除
- bcrypt 密码认证，以及基于角色的发布/订阅 ACL
- 客户端连接、接收消息和主题数据记录
- 内置 Web 管理界面和实时 WebSocket 订阅
- Cobra 命令行、Wire 依赖装配和 GORM Gen 查询代码

### mbot 客户端

- MQTT 自动连接、断线重连和主题订阅
- 采集操作系统、CPU、内存、磁盘、网络、虚拟化及硬件信息
- 根据硬件特征生成 SHA-256 主机 ID
- 每 30 秒上报一次模拟温度
- 接收命令、执行限时 Shell 任务并发布结果

## 项目结构

```text
mbot2/
├── README.md
├── mkrab/                    # MQTT Broker（独立 Go 模块）
│   ├── cmd/mkrab/            # 命令行入口
│   ├── internal/admin/       # Web 管理界面及 API
│   ├── internal/broker/      # MQTT 服务、认证和消息处理
│   ├── internal/data/        # SQLite、Repository 和生成的 Query
│   ├── config.example.yaml
│   └── go.mod
└── mbot/                     # 设备客户端（独立 Go 模块）
    ├── main.go               # MQTT 收发和命令执行
    ├── util/                 # 主机信息采集及 JSON 工具
    └── go.mod
```

## 环境要求

- Go 1.26.1 或兼容版本
- 可选：Mosquitto CLI，用于手动发布和订阅测试消息

仓库没有根级 `go.mod`，请在 `mkrab` 和 `mbot` 目录中分别执行 Go 命令。

## 快速开始

### 1. 启动 mkrab

```sh
cd mkrab
cp config.example.yaml config.yaml
go run ./cmd/mkrab serve --config config.yaml
```

默认开启以下服务：

| 服务 | 地址 | 说明 |
| --- | --- | --- |
| MQTT TCP | `127.0.0.1:1883` | 客户端连接入口 |
| Web 管理界面 | `http://127.0.0.1:8080` | 查看主题、消息、连接、用户及 ACL |
| SQLite | `mkrab/mqtt.db` | 账号、ACL、消息和连接记录 |

首次启动且用户表为空时，Broker 会创建 `root` 用户、`admin` 角色和允许发布/订阅 `#` 的规则。随机初始密码只会在首次启动日志中显示一次，请立即保存。

### 2. 配置 mbot

打开 `mbot/main.go`，修改 `cfg`。其中 `Password` 应替换为上一步日志中的初始密码：

```go
var cfg = &MqttConfig{
	Broker:   "127.0.0.1:1883",
	ClientID: "go-temperature-client",
	Username: "root",
	Password: "<mkrab 首次启动生成的密码>",
	SubTopic: "sugar/home/cmd",
	PubTopic: "sugar/home/sensor/temperature",
}
```

多个客户端同时运行时，每个实例必须使用不同的 `ClientID`。生产环境不要把密码硬编码在源码中。

### 3. 启动 mbot

另开一个终端：

```sh
cd mbot
go run .
```

客户端将连接 Broker、订阅命令主题、打印一次主机信息 JSON，并每 30 秒发布一条模拟温度消息。此时可在 `http://127.0.0.1:8080` 查看连接和消息。

## mkrab 配置

完整示例位于 [`mkrab/config.example.yaml`](mkrab/config.example.yaml)：

```yaml
database: "mqtt.db"

listeners:
  tcp:
    enabled: true
    address: ":1883"
  tls:
    enabled: false
    address: ":8883"
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
  websocket:
    enabled: false
    address: ":8083"
  wss:
    enabled: false
    address: ":8084"
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

message_store:
  enabled: true
  topic: "#"

web:
  enabled: true
  address: "127.0.0.1:8080"
```

至少需要启用一个 MQTT Listener。TLS 和 WSS 必须提供证书与私钥，最低支持 TLS 1.2。`message_store.topic` 决定 Broker 内置订阅要持久化的主题范围。

### 数据和权限模型

Broker 自动维护以下 SQLite 表：

| 表 | 用途 |
| --- | --- |
| `mqtt_user` | 用户、bcrypt 密码哈希和启用状态 |
| `mqtt_role` | 角色 |
| `mqtt_user_role` | 用户与角色关系 |
| `mqtt_acl_rule` | 角色的发布/订阅 allow、deny 规则 |
| `received_message` | Broker 内置订阅收到的消息 |
| `mqtt_client_connection` | 客户端连接、断开和协议元数据 |

ACL 的 `operation` 为 `publish` 或 `subscribe`，`effect` 为 `allow` 或 `deny`。同一用户的多个角色合并时 deny 优先；没有匹配到 allow 时默认拒绝。认证和鉴权会实时读取数据库，修改数据后无需重启 Broker。

## MQTT 消息协议

### 温度上报

默认主题：`sugar/home/sensor/temperature`

```json
{
  "device_id": "sensor_001",
  "temperature": 22.5,
  "ts": 1750000000
}
```

`temperature` 单位为摄氏度，`ts` 是秒级 Unix 时间戳。当前设备 ID 和温度均为示例数据。

### 命令下发与结果

默认命令主题：`sugar/home/cmd`

```json
{
  "msg": "bash",
  "cmd": "uname -a"
}
```

当 `msg` 为 `bash` 时，`mbot` 使用 `/bin/bash -c` 执行 `cmd`；超过 10 秒的任务会被终止。结果发布到 `sugar/home/sensor/temperature`：

```json
{
  "result": "Linux host 6.x ..."
}
```

当前温度和命令结果共用一个发布主题，消费者需要根据 JSON 字段区分消息类型。如果命令同时产生标准输出和标准错误，当前实现返回标准错误。

### 使用 Mosquitto 测试

将 `<password>` 替换为 Broker 首次启动时生成的密码。

监听上报和命令结果：

```sh
mosquitto_sub \
  -h 127.0.0.1 -p 1883 \
  -u root -P '<password>' \
  -t sugar/home/sensor/temperature
```

发送测试命令：

```sh
mosquitto_pub \
  -h 127.0.0.1 -p 1883 \
  -u root -P '<password>' \
  -t sugar/home/cmd \
  -m '{"msg":"bash","cmd":"uname -a"}'
```

## 主机信息采集

`mbot` 启动时输出操作系统、CPU、内存、磁盘、IPv4 网络、虚拟化和硬件信息。采集公网 IP 时会访问 `https://ipinfo.io`；请求失败时 `public_ip` 为空，不影响 MQTT 客户端继续运行。

## 开发与测试

分别测试两个模块：

```sh
(cd mkrab && go test ./...)
(cd mbot && go test ./...)
```

修改 mkrab 数据模型后重新生成 GORM Query：

```sh
cd mkrab
go run ./cmd/mkrab gen
```

修改 Wire provider 后重新生成依赖装配代码：

```sh
cd mkrab
go generate ./internal/app
```

## 安全建议

- 使用独立的低权限系统用户运行 `mbot`，并将任意 Shell 执行改为命令白名单。
- 为每台设备分配独立账号、客户端 ID 和主题权限。
- 只允许受信控制端向设备命令主题发布消息。
- 为 mkrab Web 管理界面增加身份认证，不要直接暴露到公网。
- 生产环境启用 TLS/WSS，安全保存私钥和 MQTT 凭据。
- 定期清理或归档 SQLite 中的消息及连接审计数据。

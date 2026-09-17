# mkrab MQTT broker

```sh
cp config.example.yaml config.yaml
go run ./cmd/mkrab serve --config config.yaml
```

四种监听方式都可在 YAML 中独立启用或关闭：

- MQTT over TCP，默认端口 `1883`
- MQTT over TLS，默认端口 `8883`
- MQTT over WebSocket，示例端口 `8083`
- MQTT over WSS，示例端口 `8084`

TLS 和 WSS 必须配置 `cert_file` 与 `key_file`，最低支持 TLS 1.2。至少需要启用一个
Listener；完整配置见 `config.example.yaml`。

服务启动时由 GORM 自动创建以下 SQLite 表：

- `mqtt_user`：用户、bcrypt 密码和启用状态
- `mqtt_role`：角色
- `mqtt_user_role`：用户与角色的多对多关系
- `mqtt_acl_rule`：角色的发布/订阅 allow、deny 规则
- `received_message`：内置订阅收到的 MQTT 消息
- `mqtt_client_connection`：客户端每次连接、断开及协议元数据

所有表都包含 `created_at`、`updated_at`、`deleted_at`。删除使用 GORM 软删除，常规 Query
会自动过滤 `deleted_at` 非空的数据；需要查询已删除记录时可显式使用 `Unscoped()`。

首次启动且 `mqtt_user` 表为空时，服务会自动创建 `root` 用户、`admin` 角色以及发布和
订阅 `#` 的 allow 规则。随机初始密码只在首次启动日志中显示一次，数据库只保存 bcrypt
哈希。请保存该密码，并在首次登录后更新 `mqtt_user.password_hash`。

ACL 的 `operation` 只能使用 `publish` 或 `subscribe`，`effect` 使用 `allow` 或 `deny`。
同一用户的多个角色合并时 deny 优先，未匹配 allow 时默认拒绝。数据库配置在每次认证和
鉴权时读取，因此修改后无需重启 broker。业务数据库访问使用 `internal/data/query` 中由
GORM Gen 生成的类型安全 Query；模型变化后执行 `go run ./cmd/mkrab gen` 重新生成。
所有业务数据库操作统一封装在 `internal/data/repo`，Broker 和应用层不直接调用 GORM 或
生成的 Query，便于后续 CLI、HTTP API 等入口复用相同的数据访问逻辑。

命令行使用 Cobra，依赖装配使用 Wire。修改 provider 后执行：

```sh
go generate ./internal/app
```

也可以使用 SQL 创建其他用户（bcrypt 示例密码为 `password`）：

```sql
INSERT INTO mqtt_role (name) VALUES ('admin');
INSERT INTO mqtt_user (username, password_hash, enabled)
VALUES ('root', '$2y$12$QOhy2x6iqnr68/BF8EXkpOoSzanLHsUQpntS1eu77vPgNIZJxbyGm', 1);
INSERT INTO mqtt_user_role (user_id, role_id)
SELECT u.id, r.id FROM mqtt_user u, mqtt_role r
WHERE u.username = 'root' AND r.name = 'admin';
INSERT INTO mqtt_acl_rule (role_id, operation, effect, topic_filter)
SELECT id, 'publish', 'allow', '#' FROM mqtt_role WHERE name = 'admin';
INSERT INTO mqtt_acl_rule (role_id, operation, effect, topic_filter)
SELECT id, 'subscribe', 'allow', '#' FROM mqtt_role WHERE name = 'admin';
```

密码只接受 bcrypt 哈希，可使用 `htpasswd -nBC 12 username` 生成。

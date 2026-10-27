# NovaMail Docker 部署

## 快速开始

```bash
# 克隆并启动（首次构建约 3-5 分钟）
git clone https://github.com/<your-username>/NovaMail.git
cd NovaMail
docker compose up -d
```

访问 `http://localhost:8080`。

## 使用预构建镜像（GitHub Actions 自动构建）

推送到 GitHub 后，Actions 会自动构建并发布镜像到 GHCR：

```yaml
services:
  novamail:
    image: ghcr.io/<your-username>/novamail:latest
    ports:
      - "8080:8080"
    volumes:
      - novamail-data:/app/data
    environment:
      TZ: Asia/Shanghai
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `NOVAMAIL_PORT` | 监听端口 | `8080` |
| `NOVAMAIL_DSN` | SQLite 路径 | `/app/data/novamail.db` |
| `NOVAMAIL_JWT_SECRET` | JWT 签名密钥（生产必须设置） | 随机生成（重启后登录失效） |
| `NOVAMAIL_ENCRYPT_KEY` | 敏感字段加密密钥（生产必须设置） | 随机生成（重启后已存密码/Token 无法解密） |
| `NOVAMAIL_OAUTH_MICROSOFT_CLIENT_ID` | Outlook OAuth2 默认 Client ID | 内置开发者 ID |

完整列表见 [环境变量文档](../docs/config/environment.md)。

> 💡 **密钥说明**：未设置 `NOVAMAIL_JWT_SECRET` / `NOVAMAIL_ENCRYPT_KEY` 时，
> 首次启动会自动生成随机密钥并持久化到数据库（`/app/data/novamail.db`），
> 容器重建但数据卷保留时密钥不丢失。仅当**删除数据卷**时才会重新生成，
> 届时已保存的邮箱密码和 OAuth2 Refresh Token 需要重新录入/授权。
> 生产环境如需多实例共享密钥，建议显式设置这两个环境变量。

## Outlook 发件（OAuth2）

微软已停用 SMTP 传统密码认证，Outlook 账号发件需走 OAuth2：

1. 按 [OAuth2 配置指南](../docs/guide/oauth2-microsoft.md) 注册 Azure 应用
2. 设置环境变量：
   ```yaml
   environment:
     NOVAMAIL_OAUTH_MICROSOFT_CLIENT_ID: <your-client-id>
   ```
3. Web UI → 添加账号 → Outlook → 完成 OAuth2 设备码授权
4. 授权后即可正常发件（XOAUTH2 认证，Access Token 过期自动刷新并持久化）

## 数据持久化

所有数据（SQLite 数据库、附件）存储在 `novamail-data` 卷中：

```bash
docker volume inspect novamail-data   # 查看卷位置
docker compose down                     # 停止（数据保留）
docker compose down -v                  # 停止并删除数据（谨慎！）
```

## 备份

```bash
docker compose exec novamail cp /app/data/novamail.db /dev/stdout > backup.db
```

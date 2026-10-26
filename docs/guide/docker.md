# Cosmo Mail Docker 部署

## 快速开始

```bash
# 克隆并启动（首次构建约 3-5 分钟）
git clone https://github.com/<your-username>/Cosmomail.git
cd Cosmomail
docker compose up -d
```

访问 `http://localhost:8080`。

## 使用预构建镜像（GitHub Actions 自动构建）

推送到 GitHub 后，Actions 会自动构建并发布镜像到 GHCR：

```yaml
services:
  cosmomail:
    image: ghcr.io/<your-username>/cosmomail:latest
    ports:
      - "8080:8080"
    volumes:
      - cosmomail-data:/app/data
    environment:
      TZ: Asia/Shanghai
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `COSMOMAIL_PORT` | 监听端口 | `8080` |
| `COSMOMAIL_DSN` | SQLite 路径 | `/app/data/cosmomail.db` |
| `COSMOMAIL_JWT_SECRET` | JWT 签名密钥（生产必须设置） | 随机生成（重启后登录失效） |
| `COSMOMAIL_ENCRYPT_KEY` | 敏感字段加密密钥（生产必须设置） | 随机生成（重启后已存密码/Token 无法解密） |
| `COSMOMAIL_OAUTH_MICROSOFT_CLIENT_ID` | Outlook OAuth2 默认 Client ID | 内置开发者 ID |

完整列表见 [环境变量文档](../docs/config/environment.md)。

> ⚠️ **重要**：未显式设置 `COSMOMAIL_ENCRYPT_KEY` 时密钥随容器重建而变化，
> 已保存的邮箱密码和 OAuth2 Refresh Token 将无法解密。生产部署务必固定密钥。

## Outlook 发件（OAuth2）

微软已停用 SMTP 传统密码认证，Outlook 账号发件需走 OAuth2：

1. 按 [OAuth2 配置指南](../docs/guide/oauth2-microsoft.md) 注册 Azure 应用
2. 设置环境变量：
   ```yaml
   environment:
     COSMOMAIL_OAUTH_MICROSOFT_CLIENT_ID: <your-client-id>
   ```
3. Web UI → 添加账号 → Outlook → 完成 OAuth2 设备码授权
4. 授权后即可正常发件（XOAUTH2 认证，Access Token 过期自动刷新并持久化）

## 数据持久化

所有数据（SQLite 数据库、附件）存储在 `cosmomail-data` 卷中：

```bash
docker volume inspect cosmomail-data   # 查看卷位置
docker compose down                     # 停止（数据保留）
docker compose down -v                  # 停止并删除数据（谨慎！）
```

## 备份

```bash
docker compose exec cosmomail cp /app/data/cosmomail.db /dev/stdout > backup.db
```

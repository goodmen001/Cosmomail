# 安装部署

NovaMail 仅提供原生二进制部署，不包含容器或 NAS 专用安装包。

## 一键安装（Linux / macOS）

```bash
curl -fsSL https://raw.githubusercontent.com/goodmen001/Cosmomail/main/deploy.sh | sudo bash
```

安装完成后：

```bash
novamail status
novamail logs
novamail restart
novamail update
```

默认监听 `0.0.0.0:8080`，数据保存在 `/var/lib/novamail`。

## 手动安装

从 Release 下载对应平台的文件（Linux x86_64 为 `novamail`，ARM 为 `novamail-arm` 或 `novamail-arm64`），赋予执行权限：

```bash
chmod +x novamail
NOVAMAIL_ENV=production \
NOVAMAIL_DSN=/var/lib/novamail/data/novamail.db \
./novamail
```

systemd 示例位于 `server/novamail.service`。

## Windows

下载 `novamail.exe`，在 PowerShell 中运行：

```powershell
$env:NOVAMAIL_ENV = "production"
$env:NOVAMAIL_DSN = "data/novamail.db"
.\novamail.exe
```

## 配置

常用变量：

| 变量 | 默认值 | 说明 |
|---|---:|---|
| `NOVAMAIL_PORT` | `8080` | Web 服务端口 |
| `NOVAMAIL_DSN` | `data/novamail.db` | SQLite 数据库路径 |
| `NOVAMAIL_POLL_INTERVAL` | `60` | IDLE 校验及降级轮询间隔（秒） |
| `NOVAMAIL_IDLE_ENABLED` | `true` | 启用 IMAP IDLE |
| `NOVAMAIL_MAX_CONCURRENT` | `10` | 最大同步并发数 |
| `NOVAMAIL_SYNC_BATCH_SIZE` | `50` | 单次同步处理上限 |

完整示例见项目根目录 `.env.example`。

## 反向代理

SSE 路径 `/api/v1/mails/stream` 必须关闭响应缓冲，并配置长连接读取超时。

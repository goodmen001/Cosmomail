<div align="center">

<img src="./public/images/icon_512.png" alt="NovaMail" width="128" height="128">

# NovaMail

快速、私有的多邮箱统一管理平台。

</div>

NovaMail 使用 Go、SQLite 和 Vue 3 构建，支持 IMAP IDLE、UID 增量同步、POP3 UIDL、SMTP 发信、SSE/Web Push 与 Webhook。邮件和凭据由用户自行托管。

## 快速部署

### Docker Compose（推荐）

```bash
git clone https://github.com/goodmen001/Cosmomail.git
cd NovaMail
docker compose up -d
```

或直接使用 GHCR 预构建镜像（push/tag 自动构建，支持 amd64/arm64）：

```yaml
services:
  novamail:
    image: ghcr.io/goodmen001/cosmomail:latest
    ports:
      - "8080:8080"
    volumes:
      - novamail-data:/app/data
```

详见 [Docker 部署文档](./docs/guide/docker.md)。

### 一键脚本安装

```bash
curl -fsSL https://raw.githubusercontent.com/goodmen001/Cosmomail/main/deploy.sh | sudo bash
```

安装后使用统一的 `novamail` 命令：

```bash
novamail status
novamail start
novamail stop
novamail restart
novamail update
novamail logs
novamail doctor
novamail uninstall
```

## 本地开发

需要 Go 1.25+、Node.js 20+ 和 pnpm/npm。

```bash
./dev.sh start
```

生产构建：

```bash
./build.sh
```

构建产物位于 `bin/`，可通过 `NOVAMAIL_*` 环境变量配置运行参数，示例见 `.env.example`。

## 项目来源与致谢（Project Origin & Acknowledgements）

NovaMail 是从开源项目 [Cosmomail](https://github.com/goodmen001/Cosmomail) 迭代升级而来的分支续作。

- **原项目名称**：Cosmomail
- **原作者**：magiccode（魔法代码）
- **原许可证**：AGPL-3.0-or-later

感谢原作者 magiccode 的出色工作，NovaMail 的所有底层能力（IMAP/POP3/SMTP、多账号管理、推送通知等）均建立在原项目的基础之上。原项目的 AGPL 版权声明完整保留在源码文件头与 [LICENSE](./LICENSE) 中。自 NovaMail 起进行品牌重命名与全新的三主题 UI（Aurora / Obsidian / Moonlight）等迭代。

## License

NovaMail 基于 AGPL-3.0-or-later 发布。该项目由原 AGPL 代码继续开发；原作者版权声明保留在源码与许可证中。通过网络向用户提供服务时，需要向用户提供对应版本的完整源代码。

[完整许可证](./LICENSE)

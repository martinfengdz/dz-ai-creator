# DZAI内容创作平台

DZAI内容创作平台是一个基于 Go (Gin/GORM)、Vue 3 和 uni-app 的全栈 AI 内容创作解决方案，包含 Web 管理端、移动 H5/微信小程序以及可选的模型、对象存储、短信和支付适配。

> 本项目基于 [image-agent (Content-Creation)](https://github.com/insistlv/Content-Creation) 开源项目改造，原项目采用 AGPL-3.0 许可证。

---

## 🚀 快速部署（Docker 推荐）

### 前置要求

- Docker 及 Docker Compose（建议 Docker 24+）
- 一个空域名或 IP（可选，用于公网访问）

### 1. 下载源码

```bash
git clone https://github.com/martinfengdz/dz-ai-creator.git
cd dz-ai-creator
```

或从 [Releases 页面](https://github.com/martinfengdz/dz-ai-creator/releases) 下载最新版源码压缩包。

### 2. 创建密钥文件

```bash
mkdir -p secrets
```

创建三个密钥文件：

| 文件 | 内容 | 生成命令 |
|------|------|---------|
| `secrets/postgres_password` | PostgreSQL 数据库密码 | `openssl rand -base64 16` |
| `secrets/database_url` | 数据库连接串 | 见下方 |
| `secrets/app_secrets_master_key` | 主密钥（32字节Base64） | `openssl rand -base64 32` |

**database_url 内容格式：**

```
postgres://dz_ai_creator:<密码>@postgres:5432/dz_ai_creator?sslmode=disable
```

将 `<密码>` 替换为 `postgres_password` 文件中的值。

一键创建示例：

```bash
# 自动生成所有密钥
echo -n "$(openssl rand -base64 16 | tr -d '\n')" > secrets/postgres_password
echo -n "postgres://dz_ai_creator:$(cat secrets/postgres_password)@postgres:5432/dz_ai_creator?sslmode=disable" > secrets/database_url
echo -n "$(openssl rand -base64 32 | tr -d '\n')" > secrets/app_secrets_master_key
```

### 3. 启动服务

```bash
docker compose up -d --build
```

首次启动会自动创建数据库表结构（`STARTUP_DATABASE_MIGRATIONS=bootstrap`）。

### 4. 查看运行状态

```bash
docker compose ps
docker compose logs -f app  # 查看启动日志
```

服务默认运行在 **http://localhost:8888**。

---

## 🛠️ 初始化配置

### 创建管理员账号

```bash
docker exec -it dz-ai-creator dz-ai-creator-admin create --username admin
```

按提示输入密码（不少于 12 位），此账号用于登录 Web 管理后台。

### 配置 API Key

在 Web 管理后台 → 系统设置 → 密钥管理中配置以下服务（按需）：

| 服务 | 必需？ | 用途 |
|------|--------|------|
| OpenAI API Key | ✅ 必需 | AI 内容生成（文本/图片） |
| DeepSeek API Key | 可选 | 文本生成 |
| 阿里云短信 | 可选 | 短信验证码 |
| 支付宝 | 可选 | 在线支付 |
| 微信支付 | 可选 | 微信支付 |
| OSS 对象存储 | 可选 | 文件存储 |

也可以通过命令行批量导入环境变量：

```bash
docker exec -e OPENAI_API_KEY=sk-xxxxx dz-ai-creator \
  dz-ai-creator secrets import-env --actor admin
```

---

## 🐳 Docker Compose 详解

### compose.yaml 结构

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| `app` | 本地构建 | `8888` | DZAI 主程序（Go + Vue 3） |
| `postgres` | postgres:15-alpine | `5432`（内部） | 数据库 |

### 数据持久化

| 卷 | 挂载路径 | 说明 |
|----|---------|------|
| `postgres_data` | `/var/lib/postgresql/data` | 数据库文件 |
| `app_data` | `/app/data` | 上传文件、spool 等 |

### 生产环境调优

```bash
# 关闭自动迁移（数据表已创建后）
export STARTUP_DATABASE_MIGRATIONS=existing
docker compose up -d
```

如需自定义域名，设置：

```bash
export APP_BASE_URL=https://your-domain.com
docker compose up -d
```

---

## 💻 手动部署（开发环境）

需要 Go 1.26、Node.js 24 和 PostgreSQL 15+。

```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env 填入 DATABASE_URL 和 APP_SECRETS_MASTER_KEY

# 2. 构建前端
cd web && npm ci && npm run build && cd ..

# 3. 启动服务
STARTUP_DATABASE_MIGRATIONS=bootstrap go run ./cmd/server

# 4. 创建管理员
go run ./cmd/admin create --username admin

# 5. 导入业务密钥
OPENAI_API_KEY=sk-xxxxx go run ./cmd/secrets import-env --actor admin
```

--- 

## 🔧 运维命令

| 命令 | 用途 |
|------|------|
| `docker compose logs -f app` | 查看实时日志 |
| `docker compose restart app` | 重启应用 |
| `docker compose down` | 停止并删除容器 |
| `docker exec dz-ai-creator dz-ai-creator-admin create --username <name>` | 创建管理员 |
| `docker exec dz-ai-creator dz-ai-creator secrets import-env --actor <name>` | 导入业务密钥 |

---

## 🔒 安全默认值

- 应用只从环境或挂载文件读取 `DATABASE_URL(_FILE)` 与 `APP_SECRETS_MASTER_KEY(_FILE)` 两个根信任。
- 业务凭据使用 AES-256-GCM 加密后保存在 `secret_records`，随机 nonce，并使用 `namespace/owner_id/name` 作为 AAD。
- 没有模型 API Key 时服务仍可启动；相关功能显示为未配置。
- JWT 密钥首次启动时自动生成并加密保存。首个管理员通过交互式命令创建，不使用启动密码环境变量。
- 管理接口只返回密钥是否已配置、更新时间和密钥版本，从不返回原值。

---

## 📚 更多文档

- [自托管详细指南](docs/self-hosting.md) — 密钥管理、主密钥轮换、迁移模式
- [支付宝沙箱配置](docs/alipay-setup.md)
- [阿里云短信配置](docs/aliyun-sms-setup.md)
- [微信虚拟支付配置](docs/wechat-virtual-pay-setup.md)
- [公开发布流程](docs/public-release.md)

---

## 📄 许可证与源码提供

Copyright © DZAI

本项目基于 [image-agent (Content-Creation)](https://github.com/insistlv/Content-Creation) 改造，按 [GNU Affero General Public License v3.0](LICENSE) 发布。通过网络运行修改版本时，运营者必须按 AGPL-3.0 第 13 条向交互用户提供对应源码。构建部署时请把 `VITE_SOURCE_CODE_URL` 设置为该运行版本的公开源码地址。

素材与第三方声明见 [ASSET_LICENSES.md](ASSET_LICENSES.md) 和 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 与 [DCO](DCO)。

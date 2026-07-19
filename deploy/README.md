# 生产部署指南

## 前置要求

- Docker Engine 24+（含 docker compose 插件）
- 域名 + DNS（可选，用于 HTTPS）

## 首次部署

```bash
# 1. 克隆仓库
git clone https://github.com/martinfengdz/dz-ai-creator.git
cd dz-ai-creator

# 2. 生成 secrets
python3 scripts/setup-secrets.py

# 3. 启动
docker compose -f deploy/compose.yaml up -d

# 4. 创建管理员账号
docker exec -it dz-ai-creator-app-1 dz-ai-creator-admin create --username admin

# 5. 导入业务凭据（API Key 等）
# 先将凭据写入临时环境变量文件，然后：
docker exec -it dz-ai-creator-app-1 dz-ai-creator-admin secrets import-env
```

## 使用预编译镜像（推荐）

该仓库默认通过 GitHub Actions 自动构建 Docker 镜像并推送到 GHCR：

- `ghcr.io/martinfengdz/dz-ai-creator:latest` — 最新 main 分支
- `ghcr.io/martinfengdz/dz-ai-creator:v1.0.0` — 语义版本标签
- `ghcr.io/martinfengdz/dz-ai-creator:<sha>` — 短 commit SHA

更新版本：

```bash
docker compose -f deploy/compose.yaml pull
docker compose -f deploy/compose.yaml up -d
```

## 升级

```bash
docker compose -f deploy/compose.yaml pull app
docker compose -f deploy/compose.yaml up -d
```

数据库迁移在启动时自动执行（`STARTUP_DATABASE_MIGRATIONS=bootstrap`）。

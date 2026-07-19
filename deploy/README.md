# dz-ai-creator 生产部署指南

> 帝赞AI内容创作平台 — 一键部署

## 前置要求

- Docker Engine 24+（含 docker compose 插件）
- 网络畅通（可拉取 ghcr.io 镜像）

## 快速部署（5分钟）

```bash
# 1. 克隆仓库
git clone https://github.com/martinfengdz/dz-ai-creator.git
cd dz-ai-creator

# 2. 生成 secrets
python3 scripts/setup-secrets.py

# 3. 启动
docker compose -f deploy/compose.yaml up -d

# 4. 检查运行状态
docker compose -f deploy/compose.yaml ps
```

启动后访问 http://localhost:8888

## 升级

```bash
cd dz-ai-creator
git pull
docker compose -f deploy/compose.yaml pull app
docker compose -f deploy/compose.yaml up -d
```

数据库迁移在启动时自动执行。

## 首次使用

1. 创建管理员账号（需要进入容器操作）
2. 导入业务 API Key
3. 配置域名（可选）

## 架构

| 组件 | 说明 |
|------|------|
| **PostgreSQL 15** | 数据库，独立容器 |
| **App（Go + Vue.js + UniApp）** | 主应用，从 ghcr.io 拉取预构建镜像 |

## 版本更新日志

| 版本 | 说明 |
|------|------|
| v1.0.0 | 初始版本 |
| v1.0.1 | 修复 GitHub Actions 构建问题 |
| v1.0.2 | 合并*App方法到core包，修复provider/sms局部Config类型 |
| v1.0.3 | P0-P2安全审计修复：SQL注入防护、密钥加密、CSRF加固等7项安全修复 |

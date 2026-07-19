#!/bin/bash
set -e

# 生成发布版本所需的 secrets 文件
# 运行：bash scripts/setup-secrets.sh

SECRETS_DIR="$(cd "$(dirname "$0")/.." && pwd)/secrets"
mkdir -p "$SECRETS_DIR"

# 1. PostgreSQL 密码
POSTGRES_PASS="dz_creator_$(openssl rand -base64 12 | tr '+/' '-_')"
echo -n "$POSTGRES_PASS" > "$SECRETS_DIR/postgres_password"
echo "[✓] postgres_password"

# 2. Database URL
echo -n "postgres://dz_ai_creator:${POSTGRES_PASS}@postgres:5432/dz_ai_creator?sslmode=disable" > "$SECRETS_DIR/database_url"
echo "[✓] database_url"

# 3. App Secrets Master Key (32 bytes base64)
MASTER_KEY=$(openssl rand -base64 32)
echo -n "$MASTER_KEY" > "$SECRETS_DIR/app_secrets_master_key"
echo "[✓] app_secrets_master_key"

echo ""
echo "===== Secrets 已生成到 $SECRETS_DIR/ ====="
echo "首次启动后执行以下命令导入业务凭据："
echo "  dz-ai-creator-admin secrets import-env"
SEOEF
chmod +x /tmp/dz-ai-creator/scripts/setup-secrets.sh
echo "script: $?"

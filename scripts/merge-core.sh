#!/bin/bash
# merge-core.sh — 构建时将外部 *App 方法文件合并到 core 包
# 解决 Go 不允许在不同包中为类型添加方法的限制
# 
# v1.0.6 — 完整修复：扩展依赖复制+busybox兼容+go:embed资源
# 根本原因：v1.0.2重构时SecretStore/alipayConfig等被移出core包但未加入复制列表
set -eu

SRC="${1:-$(pwd)}"
OUT="${SRC}/internal/pkg/core/build"

rm -rf "$OUT"
mkdir -p "$OUT"

# 1. 复制已有的 core 包文件
cp "$SRC"/internal/pkg/core/*.go "$OUT/" 2>/dev/null || true

# 2. 复制所有含 *App 方法的文件（排除测试文件）
# busybox兼容：用find代替grep --include
find "$SRC/internal" -name '*.go' ! -path '*/internal/pkg/core/*' ! -name '*_test.go' \
  -exec grep -l 'func (a \*App)' {} + 2>/dev/null \
  | while read -r f; do
    cp "$f" "$OUT/$(basename "$f")"
  done

# 3. 复制所有被 *App 方法文件引用的外部类型定义文件
# （v1.0.2重构时这些类型被移出core包但未同步到复制列表）
# 注意：不复制 provider.go（它通过 prov 导入保持外部包引用，避免Config冲突）
for dir in "internal/pkg/secrets" "internal/payment" "internal/pkg/storage"; do
  cp "$SRC/$dir"/*.go "$OUT/" 2>/dev/null || true
done
cp "$SRC/internal/middleware/limiter.go" "$OUT/" 2>/dev/null || true

# 4. 复制 go:embed 资源（如 ecommerce layout 使用的字体）
# embed路径相对于文件最终位置（internal/pkg/core/），
# 由Dockerfile的 mv internal/pkg/core/build/* internal/pkg/core/ 处理
cp -r "$SRC/internal/app/ecommerce/fonts" "$OUT/" 2>/dev/null || true

# 5. 统一改为 package core（保留 build tag 等注释行）
for f in "$OUT"/*.go; do
  if [ "$(basename "$f")" != "_test.go" ]; then
    sed -i '/^package /s/package .*/package core/' "$f"
  fi
done

# 6. 排除 Windows-only 文件（Linux 构建不需要，且缺 build tag 会导致重复函数）
rm -f "$OUT/system_resources_disk_windows.go"

echo "✅ Merged $(ls "$OUT"/*.go 2>/dev/null | wc -l) files into $OUT"

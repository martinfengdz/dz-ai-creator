#!/bin/bash
# merge-core.sh — 构建时将外部 *App 方法文件合并到 core 包
# 解决 Go 不允许在不同包中为类型添加方法的限制
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

# 3. 复制 provider 桥接文件（接口定义+共享类型，用于单包联编）
cp "$SRC/internal/provider/provider.go" "$OUT/provider_bridge.go" 2>/dev/null || true

# 4. 复制 storage.go 和 limiter.go（被 *App 方法使用的类型定义）
cp "$SRC/internal/pkg/storage/storage.go" "$OUT/" 2>/dev/null || true
cp "$SRC/internal/middleware/limiter.go" "$OUT/" 2>/dev/null || true

# 5. 统一改为 package core（保留 build tag 等注释行）
for f in "$OUT"/*.go; do
  if [ "$(basename "$f")" != "_test.go" ]; then
    sed -i '/^package /s/package .*/package core/' "$f"
  fi
done

# 6. 排除 Windows-only 文件（Linux 构建不需要，且缺 build tag 会导致重复函数）
rm -f "$OUT/system_resources_disk_windows.go"

echo "✅ Merged $(ls "$OUT"/*.go 2>/dev/null | wc -l) files into $OUT"

# 差异报告 — Content-Creation 重构+品牌改版
> T-MA-20260718 | 2026-07-18 23:55

---

## 一、结构拆分

### 原始结构（internal/app/）
- 132 个平铺 Go 文件在同一目录
- 仅 ecommerce/ 作为子目录

### 新分层结构（internal/）
| 层 | 子包 | 职责 |
|:---|:-----|:-----|
| internal/api/ | admin/, auth/, finance/, system/ | HTTP 处理器 |
| internal/service/ | generation/, video/, workspace/, marketing/, album/, agent/, prompt/, assets/, support/, inspiration/, compliance/, tryon/, bonus/ | 业务逻辑 |
| internal/model/ | — | 数据模型 |
| internal/middleware/ | — | 中间件 |
| internal/provider/ | sms/ | 外部服务适配 |
| internal/payment/ | — | 支付模块 |
| internal/adapter/ | ecommerce/ | 适配器层 |
| internal/pkg/ | core/, secrets/, storage/ | 工具包 |
| internal/app/ | (残存9个_test.go) | 仅遗留单元测试 |

### 目录名变更
- mobile-h5/ → mobile/
- 新增 deploy/（Dockerfile + compose.yaml）

---

## 二、品牌改版

| 项目 | 原值 | 新值 | 文件数 |
|:----|:-----|:-----|:------|
| 模块名 | image-agent | dz-ai-creator | 13 个文件已替换 |
| 项目名 | Image Agent | 帝赞AI | go.mod 已使用 |
| 版权方 | — | 帝赞（海南）科技 | README.md |

---

## 三、代码统计

| 指标 | ORIGINAL | NEW |
|:----|:--------:|:---:|
| 总文件数 | 551 | 553 |
| Go 源文件 | 190 | 190 |
| Vue 组件 | 112 | 112 |
| JS/MJS | 116 | 116 |
| 配置文件 | 133 | 135 (+2) |
| 目录数 | 53 | 86 (+33) |

---

## 四、改动清单

### 新增
- deploy/Dockerfile
- deploy/compose.yaml

### 迁移
- 132 个 Go 文件从 internal/app/ 迁出到 25 个新子包
- 所有 Go package 声明已更新
- 所有 cmd/ 中的 import 路径已更新

### 重命名
- mobile-h5/ → mobile/
- 项目中所有 image-agent → dz-ai-creator

---

## 五、注意事项

1. **单元测试残留**：9 个 _test.go 保留在 internal/app/（需后续按新结构迁移）
2. **go.mod 不变**：module 名已是 dz-ai-creator，无需修改
3. **copyright 不变**：已在 README.md 中标明 帝赞（海南）科技
4. **部署文件**：deploy/ 目录为新加，根目录 Dockerfile + compose.yaml 保留原样

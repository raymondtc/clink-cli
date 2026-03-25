# Clink CLI

天润融通 AI 友好型 CLI 工具（Go 版本）

## 特性

- 🚀 **高性能**: Go 编写，单二进制文件，无依赖
- 🎯 **多平台**: 支持 Linux/macOS/Windows，x64/ARM64
- 📦 **SDK 驱动**: 基于官方 Java SDK 源码自动生成 OpenAPI 规范
- 🎨 **统一渲染**: 支持表格、JSON 等多种输出格式
- 🧪 **完整测试**: 单元测试覆盖

## 安装

### 一键安装（推荐）

支持 macOS 和 Linux，自动识别系统和架构：

```bash
curl -fsSL https://raw.githubusercontent.com/raymondtc/clink-cli/main/install.sh | bash
```

或 wget：

```bash
wget -qO- https://raw.githubusercontent.com/raymondtc/clink-cli/main/install.sh | bash
```

安装完成后，配置环境变量即可使用。

### 手动安装

从 [GitHub Releases](https://github.com/raymondtc/clink-cli/releases) 下载对应平台的二进制文件。

## 配置

### 认证方式

天润融通 API 需要以下认证信息：

- **Access ID / Access Key ID** - API 访问 ID
- **Access Secret / Secret** - API 访问密钥

### 环境变量

支持多种环境变量名称：

```bash
# 方式 1：使用 CLINK_ACCESS_ID 和 CLINK_ACCESS_SECRET
export CLINK_ACCESS_ID="your_access_id"
export CLINK_ACCESS_SECRET="your_access_secret"

# 方式 2：使用 CLINK_ACCESS_KEY_ID 和 CLINK_SECRET
export CLINK_ACCESS_KEY_ID="your_access_id"
export CLINK_SECRET="your_access_secret"
```

### 命令行参数

```bash
# 使用 --access-id 和 --access-secret
clink --access-id xxx --access-secret yyy records inbound
```

### 全局参数

```bash
clink --help

Flags:
      --access-id string       Access ID (env: CLINK_ACCESS_ID or CLINK_ACCESS_KEY_ID)
      --access-secret string   Access Secret (env: CLINK_ACCESS_SECRET or CLINK_SECRET)
      --base-url string        API base URL (default: https://api-sh.clink.cn)
  -h, --help                   help for clink
  -o, --output string          Output format: table, json (default: "table")
```

## CLI 使用

### 查询通话记录

```bash
# 查询呼入记录
clink records inbound --start 2024-01-01 --end 2024-01-31

# 查询外呼记录
clink records outbound --start 2024-01-01 --end 2024-01-31

# 筛选特定座席
clink records inbound --agent 1001

# 筛选特定电话号码
clink records inbound --phone 13800138000

# JSON 格式输出
clink records inbound --start 2024-01-01 --end 2024-01-31 -o json
```

**参数说明**（自动生成自 OpenAPI 规范）：

| 参数 | OpenAPI 字段 | 说明 | 默认值 |
|------|-------------|------|--------|
| `-s, --start` | startTime | 开始时间 | 7天前 |
| `-e, --end` | endTime | 结束时间 | 今天 |
| `-p, --phone` | customerNumber | 客户号码筛选 | - |
| `-a, --agent` | cno | 座席号筛选 | - |
| `--offset` | offset | 偏移量 | 0 |
| `--limit` | limit | 查询条数 | 50 |

### 查询座席状态

```bash
# 所有座席
clink agents

# 特定座席
clink agents --agent 1001
```

**参数说明**：

| 参数 | OpenAPI 字段 | 说明 |
|------|-------------|------|
| `-a, --agent` | cno | 座席号（可选，不传则查询所有） |

### 发起外呼

```bash
# WebCall（无需座席，默认）
clink call 13800138000

# 指定 IVR 流程
clink call 13800138000 --ivr "工作时间"

# 指定座席外呼
clink call 13800138000 --agent 1001

# 指定外显号码
clink call 13800138000 --clid "01012345678"
```

**参数说明**：

| 参数 | OpenAPI 字段 | 说明 | 默认值 |
|------|-------------|------|--------|
| `phone` | customerNumber | 客户号码（位置参数） | 必填 |
| `-a, --agent` | cno | 座席号（指定后使用座席外呼） | - |
| `--clid` | clid | 外显号码 | - |
| `--ivr` | ivrName | IVR名称 | 工作时间 |
| `--request-id` | requestUniqueId | 请求唯一ID（防重放） | - |

### 查询队列状态

```bash
# 查询默认队列
clink queue

# 指定队列
clink queue --qnos "queue1,queue2"
```

## 项目结构

```
clink-cli/
├── sdk/
│   └── clink-sdk/             # Git 子模块：官方 Java SDK
├── scripts/
│   ├── extract-openapi.go     # 从 SDK 提取 OpenAPI 规范
│   └── generate.go            # 从 OpenAPI 生成 Go 代码
├── openapi/                   # 生成的 OpenAPI 规范（不提交到 Git）
├── cmd/
│   ├── clink/                 # CLI 工具入口
│   └── clink-mcp/             # MCP Server 入口
├── pkg/
│   ├── client/                # HTTP 客户端（含认证）
│   ├── api/                   # 业务 API 层
│   ├── models/                # 数据模型
│   └── renderer/              # 统一结果渲染器
├── Makefile                   # 构建自动化
├── .gitmodules                # 子模块配置
└── docs/
    └── DEVELOPMENT.md         # 开发文档
```

## 开发

### 从 SDK 生成 OpenAPI

本项目使用官方 Java SDK 源码作为 API 契约的源头，通过静态分析生成 OpenAPI 3.0 规范：

```bash
# 首次使用：初始化子模块
make sync-sdk

# 生成 OpenAPI 规范（从 SDK 源码提取）
make extract-openapi

# 完整流程：同步 SDK + 提取 OpenAPI
make openapi
```

生成的 OpenAPI 规范位于 `openapi/openapi.json`，包含：
- 146 个 API 端点映射
- 完整的请求/响应参数
- 中文 Javadoc 注释

### 生成 Go 代码

```bash
# 从 OpenAPI 生成类型和客户端代码
make generate

# 完整流程：SDK → OpenAPI → Go 代码
make sync-sdk extract-openapi generate
```

### 运行测试

```bash
go test -v ./...

# 带覆盖率
go test -v -race -cover ./...
```

### 构建

```bash
# 构建本地版本
make dev-build

# 清理生成文件
make clean
```

## 更新 API 支持

当官方 SDK 更新时：

```bash
# 1. 同步最新 SDK
make sync-sdk

# 2. 重新生成 OpenAPI
make extract-openapi

# 3. 重新生成代码
make generate

# 4. 构建测试
make dev-build
```

## 技术说明

### 为什么选择 SDK 源码解析？

| 方案 | 稳定性 | 维护成本 | 信息完整度 |
|------|--------|----------|-----------|
| HTML 文档抓取 | ⭐⭐ | 高（网页变动） | 低 |
| SDK 源码解析 | ⭐⭐⭐⭐⭐ | 低（跟随 SDK 版本） | 高（含完整注释） |
| Java 反射 | ⭐⭐⭐⭐ | 中（需编译） | 高 |

本项目采用 **Go 静态分析** 解析 Java SDK 源码：
- 无需 Java 运行时
- 解析 146 个 API 仅需 1-2 秒
- 精确提取字段类型、注释、枚举值
- 生成标准 OpenAPI 3.0 规范

### 解析流程

```
SDK 子模块 (Java 源码)
       ↓
PathEnum.java → API 路径映射表
       ↓
*Request.java → 参数、类型、描述
       ↓
*Response.java → 返回结构引用
       ↓
OpenAPI 3.0 JSON
       ↓
Go 代码生成器
       ↓
类型定义 + HTTP 客户端 + CLI 命令
```

## License

MIT

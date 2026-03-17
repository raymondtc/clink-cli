# Clink CLI

天润融通 AI 友好型 CLI 工具（Go 版本）

## 特性

- 🚀 **高性能**: Go 编写，单二进制文件，无依赖
- 🎯 **多平台**: 支持 Linux/macOS/Windows，x64/ARM64
- 📦 **OpenAPI 驱动**: 基于 OpenAPI 规范自动生成 CLI 参数
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
├── api/
│   └── openapi.yaml          # OpenAPI 规范（单一事实源）
├── cmd/
│   └── clink/                # CLI 工具入口
│       └── main.go           # 主程序（使用生成的参数映射）
├── pkg/
│   ├── client/               # HTTP 客户端（含认证）
│   ├── api/                  # 业务 API 层
│   ├── models/               # 数据模型
│   ├── generated/            # 生成的代码（oapi-codegen）
│   └── renderer/             # 统一结果渲染器
├── scripts/
│   ├── generate.go           # 代码生成器（类型和客户端）
│   └── cli_generator.go      # CLI 代码生成器（参数映射）
├── .github/workflows/
│   └── ci.yml                # CI/CD 配置
└── docs/
    └── DEVELOPMENT.md        # 开发文档
```

## OpenAPI 参数映射

CLI flags 自动生成自 OpenAPI 规范：

| OpenAPI 参数 | CLI Flag | 转换规则 |
|-------------|----------|----------|
| `startTime` | `--start` | 驼峰 -> 短横线连接 |
| `customerNumber` | `--phone` | 语义化映射 |
| `cno` | `--agent` | 语义化映射 |
| `ivrName` | `--ivr` | 语义化映射 |

转换规则：
1. 驼峰命名转换为短横线连接（如 `startTime` -> `start-time`，再简化为 `--start`）
2. 常用参数使用语义化简称（如 `--phone` 代替 `--customer-number`）

## 开发

### 生成代码

```bash
# 从 OpenAPI 生成类型和客户端代码
make generate

# 生成 CLI 参数映射代码（可选）
go run scripts/cli_generator.go api/openapi.yaml
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
make build

# 完整重建
clean + generate + build
make rebuild
```

## 添加新 API 支持

本项目使用代码生成系统自动生成 CLI 命令。查看完整文档：

📖 **[代码生成系统文档](docs/GENERATOR.md)**

### 快速添加新接口

```bash
# 1. 编辑配置
vim config/generator.yaml

# 2. 生成代码
make generate

# 3. 构建测试
make build
./bin/clink your-new-command --help
```

### 传统方式（手动添加）

如需手动添加（不推荐）：

1. 更新 OpenAPI 规范 (`api/openapi.yaml`)
2. 运行 `make generate` 生成类型
3. 在 `cmd/clink/main.go` 添加命令

## License

MIT

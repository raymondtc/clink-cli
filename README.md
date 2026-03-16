# Clink CLI

天润融通 AI 友好型 CLI 与 MCP 工具（Go 版本）

## 特性

- 🚀 **高性能**: Go 编写，单二进制文件，无依赖
- 🎯 **多平台**: 支持 Linux/macOS/Windows，x64/ARM64
- 🔧 **双模式**: CLI 工具 + MCP Server
- 🧪 **完整测试**: 单元测试覆盖
- 📦 **OpenAPI**: 基于 OpenAPI 规范，自定义代码生成器

## 安装

### 二进制下载

从 [GitHub Releases](https://github.com/raymondtc/clink-cli/releases) 下载对应平台的二进制文件。

```bash
# Linux/macOS
curl -L https://github.com/raymondtc/clink-cli/releases/latest/download/clink-linux-amd64 -o clink
chmod +x clink
sudo mv clink /usr/local/bin/

# 同时安装 MCP Server
curl -L https://github.com/raymondtc/clink-cli/releases/latest/download/clink-mcp-linux-amd64 -o clink-mcp
chmod +x clink-mcp
sudo mv clink-mcp /usr/local/bin/
```

### 源码编译

```bash
git clone https://github.com/raymondtc/clink-cli.git
cd clink-cli
go build -o clink ./cmd/clink
go build -o clink-mcp ./cmd/clink-mcp
```

## 配置

### 认证方式

天润融通 API 需要以下认证信息：

- **Access ID / Access Key ID** - API 访问 ID
- **Access Secret / Secret** - API 访问密钥
- **Enterprise ID**（可选）- 企业 ID

### 环境变量

支持多种环境变量名称：

```bash
# 方式 1：使用 CLINK_ACCESS_ID 和 CLINK_ACCESS_SECRET
export CLINK_ACCESS_ID="your_access_id"
export CLINK_ACCESS_SECRET="your_access_secret"

# 方式 2：使用 CLINK_ACCESS_KEY_ID 和 CLINK_SECRET
export CLINK_ACCESS_KEY_ID="your_access_id"
export CLINK_SECRET="your_access_secret"

# Enterprise ID（可选）
export CLINK_ENTERPRISE_ID="your_enterprise_id"
```

### 命令行参数

```bash
# 使用 --access-id 和 --access-secret
clink --access-id xxx --access-secret yyy records inbound

# 或使用 --access-key-id 和 --secret
clink --access-key-id xxx --secret yyy records inbound
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
```

### 查询座席状态

```bash
# 所有座席
clink agents

# 特定座席
clink agents --agent 1001
```

### 发起外呼

```bash
clink call 13800138000 --agent 1001
```

### 查询队列状态

```bash
clink queue
```

## MCP Server 使用

### OpenClaw 配置

```json
{
  "mcpServers": {
    "clink": {
      "command": "/usr/local/bin/clink-mcp",
      "env": {
        "CLINK_ACCESS_ID": "your_access_id",
        "CLINK_ACCESS_SECRET": "your_access_secret"
      }
    }
  }
}
```

### 可用工具

- `get_inbound_records` - 获取呼入记录
- `get_outbound_records` - 获取外呼记录
- `get_agent_status` - 查询座席状态
- `make_call` - 发起外呼
- `get_queue_status` - 查询队列状态

详见 [Agent 使用手册](docs/AGENT_MANUAL.md)

## 项目结构

```
clink-cli/
├── api/
│   └── openapi.yaml          # OpenAPI 规范
├── cmd/
│   ├── clink/                # CLI 入口
│   └── clink-mcp/            # MCP Server 入口
├── pkg/
│   ├── client/               # HTTP 客户端
│   ├── api/                  # 业务 API 层
│   └── models/               # 数据模型
├── scripts/
│   └── generate.go           # 代码生成器
├── .github/workflows/
│   └── ci.yml                # CI/CD 配置
└── docs/
    └── AGENT_MANUAL.md       # Agent 使用手册
```

## 开发

```bash
# 生成代码（基于 OpenAPI）
make generate

# 运行测试
go test -v ./...

# 运行测试（带覆盖率）
go test -v -race -cover ./...

# 构建本地版本
make build

# 完整重建
clean + generate + build
make rebuild
```

## License

MIT

# Clink CLI 开发文档

本文档供 AI Agent 参考，用于后续开发迭代。

## 项目架构

```
clink-cli/
├── api/
│   └── openapi.yaml          # OpenAPI 规范（单一事实源）
├── cmd/
│   ├── clink/                # CLI 工具入口
│   └── clink-mcp/            # MCP Server 入口
├── pkg/
│   ├── client/               # HTTP 客户端（含认证、Mock）
│   ├── api/                  # 业务 API 层
│   ├── models/               # 数据模型
│   └── generated/            # 生成的代码（可选）
├── scripts/
│   ├── generate.go           # 代码生成器
│   ├── go.mod
│   └── go.sum
├── docs/
│   ├── AGENT_MANUAL.md       # Agent 使用手册
│   └── DEVELOPMENT.md        # 本文件
└── .github/workflows/
    └── ci.yml                # CI/CD 配置
```

## 核心组件

### 1. HTTP 客户端 (pkg/client/client.go)

**职责**: 处理 HTTP 请求、API 认证、Mock 模式

**关键功能**:
- HMAC-SHA256 签名认证
- 自动添加认证参数 (accessId, timestamp, signature)
- Enterprise ID 可选（为空时不添加）
- Mock 模式支持（无需真实 API 凭证）

**配置项**:
```go
type Config struct {
    AccessID      string        // API Access ID
    AccessSecret  string        // API Secret
    EnterpriseID  string        // 企业 ID（可选）
    BaseURL       string        // API 基础 URL
    Timeout       time.Duration // 超时时间
    EnableMock    bool          // 是否启用 Mock
}
```

### 2. 业务 API 层 (pkg/api/api.go)

**职责**: 封装业务逻辑，处理数据转换

**接口方法**:
- `GetInboundRecords()` - 获取呼入记录
- `GetOutboundRecords()` - 获取外呼记录
- `GetAgentStatus()` - 查询座席状态
- `MakeCall()` - 发起外呼
- `GetQueueStatus()` - 查询队列状态

### 3. 数据模型 (pkg/models/models.go)

**核心类型**:
```go
CallRecord   // 通话记录
Agent        // 座席信息
Queue        // 队列状态
CallResult   // 呼叫结果
APIResponse  // API 响应包装
```

## 代码生成

### 生成器 (scripts/generate.go)

**用途**: 基于 OpenAPI YAML 生成 Go 代码

**使用方法**:
```bash
go run scripts/generate.go api/openapi.yaml
```

**生成内容**:
- `pkg/generated/types.go` - 类型定义
- `pkg/generated/client.go` - HTTP 客户端

**扩展方式**:
1. 修改 `scripts/generate.go` 中的模板
2. 重新运行生成命令

## 添加新功能

### 场景 1: 添加新 API 接口

1. **更新 OpenAPI 规范** (`api/openapi.yaml`):
   ```yaml
   /api/new/endpoint:
     get:
       operationId: getNewData
       parameters:
         - name: param1
           schema:
             type: string
   ```

2. **重新生成代码**（如果使用生成器）:
   ```bash
   make generate
   ```

3. **在业务层添加方法** (`pkg/api/api.go`):
   ```go
   func (a *API) GetNewData(ctx context.Context, param1 string) (*NewData, error) {
       // 实现逻辑
   }
   ```

4. **添加 CLI 命令** (`cmd/clink/main.go`):
   ```go
   var newCmd = &cobra.Command{
       Use:   "new-cmd",
       RunE: func(cmd *cobra.Command, args []string) error {
           // 调用 API
       },
   }
   ```

5. **添加 MCP 工具** (`cmd/clink-mcp/main.go`):
   ```go
   // 在 tools/list 中添加
   {
       "name": "get_new_data",
       "description": "获取新数据",
   }
   ```

### 场景 2: 修改认证方式

**当前**: HMAC-SHA256 签名

**修改位置**: `pkg/client/client.go`
- `generateSignature()` - 签名算法
- `Request()` - 参数添加逻辑

### 场景 3: 添加 Mock 数据

**修改位置**: `pkg/client/client.go`

在 `mockResponse()` 函数中添加新 case:
```go
case strings.Contains(path, "new/endpoint"):
    return &models.APIResponse{
        Code: 200,
        Data: models.NewData{...},
    }
```

## 认证机制

### HMAC-SHA256 签名流程

1. 收集所有请求参数（除 signature 外）
2. 按参数名排序
3. 拼接成字符串: `param1=value1param2=value2`
4. 使用 AccessSecret 作为 key，计算 HMAC-SHA256
5. 将签名转为 hex 字符串

### 代码位置

```go
// pkg/client/client.go
func (c *Client) generateSignature(params map[string]string) string {
    // 排序
    sort.Strings(keys)
    
    // 拼接
    paramStr := strings.Join(parts, "")
    
    // HMAC
    h := hmac.New(sha256.New, []byte(c.config.AccessSecret))
    h.Write([]byte(paramStr))
    return hex.EncodeToString(h.Sum(nil))
}
```

## 环境变量支持

### 认证相关

| 变量名 | 说明 | 优先级 |
|--------|------|--------|
| CLINK_ACCESS_ID | Access ID | 1 |
| CLINK_ACCESS_KEY_ID | Access ID (别名) | 1 |
| CLINK_ACCESS_SECRET | Access Secret | 1 |
| CLINK_SECRET | Access Secret (别名) | 1 |
| CLINK_ENTERPRISE_ID | Enterprise ID（可选） | 2 |

### 获取逻辑

```go
func getEnvWithFallback(names ...string) string {
    for _, name := range names {
        if value := os.Getenv(name); value != "" {
            return value
        }
    }
    return ""
}
```

## 测试

### 运行测试

```bash
go test -v ./...
```

### Mock 模式测试

无需真实 API 凭证：
```go
config := client.DefaultConfig()
config.EnableMock = true
c := client.NewClient(config)
```

## CI/CD

### 触发条件

- **Push to main**: 运行测试
- **Push tag v***: 运行测试 + 构建多平台二进制 + 创建 Release

### 构建流程

1. 下载依赖
2. 生成代码（可选）
3. 运行测试
4. 构建二进制
5. 上传 artifacts
6. 创建 Release

## 常见问题

### Q: 如何添加对新 API 的支持？

A: 参见"场景 1: 添加新 API 接口"

### Q: Enterprise ID 是否必需？

A: 不是必需的。如果为空，不会添加到请求参数中。

### Q: 如何修改代码生成逻辑？

A: 编辑 `scripts/generate.go`，修改模板部分。

### Q: Mock 数据在哪里定义？

A: `pkg/client/client.go` 中的 `mockResponse()` 函数。

## 参考链接

- [天润融通 API 文档](https://develop.clink.cn/)
- [OpenAPI 规范](api/openapi.yaml)
- [Agent 使用手册](docs/AGENT_MANUAL.md)

# Clink CLI 开发文档

本文档供 AI Agent 参考，用于后续开发迭代。

## 项目架构

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
    └── DEVELOPMENT.md        # 本文件
```

## 核心组件

### 1. HTTP 客户端 (pkg/client/client.go)

**职责**: 处理 HTTP 请求、API 认证

**关键功能**:
- HMAC-SHA1 签名认证（天润标准）
- 自动添加认证参数 (AccessKeyId, Timestamp, Expires, Signature)
- 支持北京/上海双平台

**配置项**:
```go
type AuthConfig struct {
    AccessID     string  // API Access ID
    AccessSecret string  // API Secret
}
```

### 2. 统一渲染器 (pkg/renderer/renderer.go)

**职责**: 统一处理 CLI 输出格式

**支持的格式**:
- `table` - 格式化的文本表格（默认）
- `json` - JSON 格式

**使用方法**:
```go
import "github.com/raymondtc/clink-cli/pkg/renderer"

// 创建渲染器
r := renderer.New(renderer.FormatTable)

// 渲染数据
r.Render(data)

// 带总计的列表渲染
renderer.RenderResult(records, total, renderer.FormatTable)
```

### 3. 业务 API 层 (pkg/api/generated_api.go)

**职责**: 封装业务逻辑，处理数据转换

**接口方法**:
- `GetInboundRecords()` - 获取呼入记录
- `GetOutboundRecords()` - 获取外呼记录
- `GetAgentStatus()` - 查询座席状态
- `MakeCall()` - 发起座席外呼
- `Webcall()` - 发起 WebCall
- `GetQueueStatus()` - 查询队列状态

### 4. CLI 主程序 (cmd/clink/main.go)

**职责**: 命令行接口

**特点**:
- Flags 映射自 OpenAPI 参数
- 使用统一渲染器输出
- 支持环境变量和命令行参数

## OpenAPI 参数映射

CLI flags 自动生成自 OpenAPI 规范：

```yaml
# OpenAPI 参数
parameters:
  - name: startTime
    in: query
    schema:
      type: string
```

```go
// CLI flag（自动生成）
cmd.Flags().String("start", "", "开始时间 (OpenAPI: startTime)")
```

### 命名转换规则

| OpenAPI 参数 | CLI Flag | 规则 |
|-------------|----------|------|
| `startTime` | `--start` | 驼峰 -> 短横线，语义化简写 |
| `customerNumber` | `--phone` | 语义化映射 |
| `cno` | `--agent` | 语义化映射 |

## 代码生成

### 生成类型和客户端

```bash
make generate
```

使用 [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) 从 OpenAPI 生成：
- `pkg/generated/clink.gen.go` - 类型和 HTTP 客户端

### 生成 CLI 代码（可选）

```bash
go run scripts/cli_generator.go api/openapi.yaml
```

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

2. **重新生成代码**:
   ```bash
   make generate
   ```

3. **在业务层添加方法** (`pkg/api/generated_api.go`):
   ```go
   func (a *GeneratedAPI) GetNewData(ctx context.Context, param1 string) (*NewData, error) {
       // 实现逻辑
   }
   ```

4. **添加 CLI 命令** (`cmd/clink/main.go`):
   ```go
   var newCmd = &cobra.Command{
       Use:   "new",
       Short: "新功能",
       RunE:  runNew,
   }

   func init() {
       rootCmd.AddCommand(newCmd)
       newCmd.Flags().String("param1", "", "参数说明 (OpenAPI: param1)")
   }

   func runNew(cmd *cobra.Command, args []string) error {
       a, err := createAPI()
       if err != nil {
           return err
       }
       data, err := a.GetNewData(context.Background(), param1)
       if err != nil {
           return err
       }
       return renderOutput(data)
   }
   ```

### 场景 2: 修改输出格式

**修改位置**: `pkg/renderer/renderer.go`

添加新的输出格式：
```go
const (
    FormatTable Format = "table"
    FormatJSON  Format = "json"
    FormatYAML  Format = "yaml"  // 新增
)
```

## 认证机制

### HMAC-SHA1 签名流程

1. 构建签名内容: `BaseURL + Method + Path + SortedParams`
2. 使用 AccessSecret 作为 key，计算 HMAC-SHA1
3. 将签名转为 base64 字符串

### 认证参数

- `AccessKeyId` - Access Key ID
- `Expires` - 60（秒）
- `Timestamp` - UTC 时间 ISO 8601 格式
- `Signature` - HMAC-SHA1 签名

**代码位置**: `pkg/client/auth.go`

## 环境变量支持

| 变量名 | 说明 | 优先级 |
|--------|------|--------|
| CLINK_ACCESS_ID | Access ID | 1 |
| CLINK_ACCESS_KEY_ID | Access ID (别名) | 1 |
| CLINK_ACCESS_SECRET | Access Secret | 1 |
| CLINK_SECRET | Access Secret (别名) | 1 |
| CLINK_BASE_URL | API 基础 URL | 2 |

### 获取逻辑

```go
func resolveAccessID() string {
    if accessID != "" {  // 命令行参数
        return accessID
    }
    if v := os.Getenv("CLINK_ACCESS_ID"); v != "" {
        return v
    }
    return os.Getenv("CLINK_ACCESS_KEY_ID")
}
```

## 测试

### 运行测试

```bash
go test -v ./...
```

### 测试认证

测试需要环境变量：
```bash
export CLINK_ACCESS_ID="your_access_id"
export CLINK_ACCESS_SECRET="your_secret"
go test -v ./pkg/client/...
```

## CI/CD

### 触发条件

- **Push to main**: 运行测试
- **Push tag v***: 运行测试 + 构建多平台二进制 + 创建 Release

### 构建流程

1. 下载依赖
2. 生成代码（`make generate`）
3. 运行测试
4. 构建二进制（仅 CLI，无 MCP）
5. 上传 artifacts
6. 创建 Release

## 常见问题

### Q: 如何添加对新 API 的支持？

A: 参见"场景 1: 添加新 API 接口"

### Q: CLI flags 如何映射 OpenAPI 参数？

A: 参见"OpenAPI 参数映射"章节

### Q: 如何修改输出格式？

A: 修改 `pkg/renderer/renderer.go`，添加新的 Format 和渲染逻辑。

## 参考链接

- [天润融通 API 文档](https://develop.clink.cn/)
- [OpenAPI 规范](api/openapi.yaml)
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)

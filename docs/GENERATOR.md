# Clink CLI 代码生成系统

基于配置和 OpenAPI 规范的自动化代码生成系统。

## 架构概览

```
api/openapi.yaml        ← 完整的 OpenAPI 规范
     ↓
config/generator.yaml   ← 只生成配置中定义的接口 + 自定义映射
     ↓
scripts/clink-generator  → 生成 CLI 命令代码
scripts/api-generator    → 生成 API 方法代码
     ↓
cmd/clink/*_gen.go      ← 生成的 CLI 命令
pkg/api/auto_generated.go ← 生成的 API 方法
```

## 快速开始

### 生成所有代码

```bash
make generate
# 或
./scripts/clink-generate.sh all
```

### 添加新接口

#### 方法 1: 手动编辑配置

1. 编辑 `config/generator.yaml`，添加新接口配置：

```yaml
endpoints:
  myNewEndpoint:
    command: ["mycommand", "sub"]
    description: "我的新接口"
    flags:
      - param: customerNumber
        flag: phone
        shorthand: p
        description: "客户号码"
        required: true
      - param: cno
        flag: agent
        shorthand: a
        description: "座席号"
```

2. 重新生成代码：

```bash
make generate
```

#### 方法 2: 使用交互式脚本

```bash
./scripts/clink-add-endpoint.sh operationId -c "command sub" -d "Description"
```

示例：

```bash
# 添加接口，使用默认命令名
./scripts/clink-add-endpoint.sh listCdrIbs

# 添加接口，指定命令路径
./scripts/clink-add-endpoint.sh webcall -c "call web" -d "发起WebCall外呼"

# 添加接口，自定义 flag 映射
./scripts/clink-add-endpoint.sh webcall \
  -f "customerNumber:phone:p:客户号码" \
  -f "clid:display::外显号码"
```

## 配置文件说明

### `config/generator.yaml`

```yaml
version: "1.0"

global:
  outputFormat: table
  defaultPageSize: 10
  timeFormat: "2006-01-02"

endpoints:
  # operationId (必须与 OpenAPI 中的 operationId 匹配)
  listCdrIbs:
    # 命令路径: clink records inbound
    command: ["records", "inbound"]
    description: "查询呼入通话记录"
    # 自定义使用说明（可选）
    use: "inbound [flags]"
    # 位置参数定义（可选）
    args:
      - name: phone
        description: "电话号码"
        required: true
    # flag 映射
    flags:
      - param: startTime          # OpenAPI 中的参数名
        flag: start                # CLI flag 名
        shorthand: s               # 短选项（可选）
        description: "开始时间"     # 描述（可选，默认使用 OpenAPI）
        default: "-7d"             # 默认值（可选）
        required: false            # 是否必需（可选）
        source: arg                # "arg" 表示从位置参数获取（可选）
```

### 字段说明

| 字段 | 说明 | 必需 |
|------|------|------|
| `command` | 命令路径数组，如 `["records", "inbound"]` 生成 `clink records inbound` | 是 |
| `description` | 命令描述 | 是 |
| `use` | 使用说明，覆盖默认格式 | 否 |
| `args` | 位置参数定义 | 否 |
| `flags` | flag 映射列表 | 否 |
| `flags[].param` | OpenAPI 中的参数名 | 是 |
| `flags[].flag` | CLI flag 名（空表示使用位置参数） | 是 |
| `flags[].shorthand` | 短选项，如 `-s` | 否 |
| `flags[].default` | 默认值 | 否 |
| `flags[].required` | 是否必需参数 | 否 |
| `flags[].source` | `"arg"` 表示从位置参数获取值 | 否 |

## 可用命令

### Make 命令

```bash
make help              # 显示帮助
make generate          # 生成所有代码
make generate-types    # 只生成类型
make generate-cli      # 只生成 CLI
make generate-api      # 只生成 API
make build             # 构建 CLI
make test              # 运行测试
make clean             # 清理生成的文件
make rebuild           # 清理 + 生成 + 构建
make add-endpoint      # 交互式添加接口
```

### 脚本命令

```bash
./scripts/clink-generate.sh all     # 生成所有代码
./scripts/clink-generate.sh types   # 只生成类型
./scripts/clink-generate.sh cli     # 只生成 CLI
./scripts/clink-generate.sh api     # 只生成 API
./scripts/clink-generate.sh check   # 检查配置一致性
./scripts/clink-generate.sh stats   # 显示统计信息
```

## 工作流程

### 1. 添加新接口到 OpenAPI

在 `api/openapi.yaml` 中添加新端点定义。

### 2. 配置接口映射

在 `config/generator.yaml` 中添加配置：

```yaml
endpoints:
  myNewEndpoint:
    command: ["mycommand"]
    description: "我的新命令"
    flags:
      - param: param1
        flag: flag1
        shorthand: f
```

### 3. 生成代码

```bash
make generate
```

### 4. 构建测试

```bash
make build
./bin/clink mycommand --help
```

## 配置检查

检查哪些接口已配置但未在 OpenAPI 中定义：

```bash
./scripts/clink-generate.sh check
```

输出示例：

```
Configuration Check
══════════════════════════════════════════════════════════════
Checking config vs OpenAPI spec...

Configured endpoints:
  ✓ listCdrIbs
  ✓ listCdrObs
  ✗ myOldEndpoint (not in OpenAPI spec)

Available in OpenAPI but not configured:
  • listQueues
  • describeClient
```

## 生成的代码结构

### CLI 命令文件

生成在 `cmd/clink/` 目录：

- `root_gen.go` - 根命令定义
- `records_gen.go` - records 子命令
- `agents_gen.go` - agents 子命令
- `call_gen.go` - call 子命令
- `utils_gen.go` - 工具函数

### API 方法文件

生成在 `pkg/api/` 目录：

- `auto_generated.go` - 自动生成的 API 方法

## 优化特性

1. **白名单机制**：只生成配置中定义的接口
2. **参数映射**：灵活映射 OpenAPI 参数到 CLI flag
3. **统一错误处理**：使用 `pkg/response` 包
4. **类型推断**：自动从 OpenAPI 推断参数类型
5. **代码格式化**：自动生成后自动格式化
6. **配置验证**：检查配置与 OpenAPI 的一致性

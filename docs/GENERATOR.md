# Clink CLI 代码生成系统

基于配置和 OpenAPI 规范的自动化代码生成系统。

## 快速开始

### 1. 添加新接口到 OpenAPI

在 `api/openapi.yaml` 中添加新端点：

```yaml
paths:
  /cc/my_new_endpoint:
    get:
      summary: 我的新接口
      operationId: myNewEndpoint  # 这个 ID 很重要！
      tags: [MyTag]
      parameters:
        - name: startTime
          in: query
          required: true
          schema:
            type: integer
          description: 开始时间
      responses:
        '200':
          description: 成功
```

### 2. 配置接口映射

在 `config/generator.yaml` 中添加：

```yaml
endpoints:
  myNewEndpoint:  # 必须与 operationId 匹配
    command: ["mycommand", "sub"]  # 生成: clink mycommand sub
    description: "我的新接口"
    resultType: "list"  # 可选: list/simple/kv/custom
    flags:
      - param: startTime     # OpenAPI 参数名
        flag: start          # CLI flag 名 (--start)
        shorthand: s         # 短选项 (-s)
        type: int            # 覆盖类型推断
        description: "开始时间"
        default: "0"
```

### 3. 生成并构建

```bash
make generate  # 生成代码
make build     # 构建
./bin/clink mycommand sub --help
```

---

## 完整配置指南

### 配置结构

```yaml
endpoints:
  operationId:           # 必须与 OpenAPI 中的 operationId 一致
    command: []          # 命令路径数组
    description: ""      # 命令描述
    use: ""              # 自定义使用说明（可选）
    resultType: ""       # 结果处理方式（可选）
    customTemplate: ""   # 自定义渲染模板（可选）
    custom: {}           # 特殊配置（可选）
    args: []             # 位置参数定义（可选）
    flags: []            # flag 映射列表
```

### 字段详解

#### `command` (必需)

定义 CLI 命令路径：

```yaml
command: ["records", "inbound"]  # 生成: clink records inbound
command: ["agents"]               # 生成: clink agents (单级命令)
```

#### `resultType` (可选)

控制结果输出方式：

| 值 | 说明 | 示例 |
|----|------|------|
| `list` | 列表输出，带总计 | records inbound |
| `simple` | 简单成功提示 | agent online |
| `kv` | 键值对输出 | call webcall |
| `custom` | 自定义渲染 | agents, queue |

#### `customTemplate` (可选)

使用预定义自定义渲染：

```yaml
customTemplate: "agentsRender"  # 座席状态图标渲染
customTemplate: "queueRender"   # 队列 KV 渲染
```

#### `flags` (必需)

定义参数映射：

```yaml
flags:
  - param: customerNumber    # OpenAPI 参数名
    flag: phone              # CLI flag 名 (--phone)
    shorthand: p             # 短选项 (-p)
    type: string             # 类型（覆盖自动推断）
    description: "客户号码"   # 描述
    default: ""              # 默认值
    defaultFunc: "time.Now().Format(...)"  # 动态默认值
    required: true           # 是否必需
    source: arg              # 从位置参数获取
```

**类型支持**：`string`, `int`, `bool`

**动态默认值**：
```yaml
defaultFunc: "time.Now().AddDate(0,0,-7).Format(\"2006-01-02\")"
```

#### `args` (可选)

定义位置参数：

```yaml
args:
  - name: phone
    description: "客户电话号码"
    required: true
```

#### `custom` (可选)

特殊配置：

**条件 API 调用**（call 命令）：
```yaml
custom:
  conditional: true
  conditionField: "agent"
  conditionAPI: "MakeCall"
# 逻辑：如果 --agent 指定了，调用 MakeCall，否则调用 Webcall
```

**时间范围**（records 命令）：
```yaml
custom:
  timeRange: true
  pagination: true
```

---

## 新增接口完整示例

### 示例 1：简单查询接口

假设要添加一个查询客户信息的接口。

**1. OpenAPI 定义** (`api/openapi.yaml`):

```yaml
  /cc/customer_info:
    get:
      summary: 查询客户信息
      operationId: getCustomerInfo
      tags: [Customer]
      parameters:
        - name: phone
          in: query
          required: true
          schema:
            type: string
          description: 客户电话号码
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  name:
                    type: string
                  phone:
                    type: string
```

**2. 配置映射** (`config/generator.yaml`):

```yaml
endpoints:
  getCustomerInfo:
    command: ["customer", "info"]
    description: "查询客户信息"
    resultType: "kv"
    flags:
      - param: phone
        flag: phone
        shorthand: p
        type: string
        description: "客户电话号码"
        required: true
```

**3. 生成并测试**:

```bash
make generate
make build
./bin/clink customer info --phone 13800138000
```

### 示例 2：带动态默认值的列表查询

**1. OpenAPI 定义**:

```yaml
  /cc/list_records:
    get:
      summary: 查询记录列表
      operationId: listRecords
      tags: [Records]
      parameters:
        - name: startDate
          in: query
          required: true
          schema:
            type: string
          description: 开始日期
        - name: endDate
          in: query
          required: true
          schema:
            type: string
          description: 结束日期
```

**2. 配置映射**:

```yaml
endpoints:
  listRecords:
    command: ["records", "list"]
    description: "查询记录列表"
    resultType: "list"
    custom:
      timeRange: true
    flags:
      - param: startDate
        flag: start
        shorthand: s
        type: string
        description: "开始日期 (YYYY-MM-DD)"
        defaultFunc: "time.Now().AddDate(0,0,-7).Format(\"2006-01-02\")"
      - param: endDate
        flag: end
        shorthand: e
        type: string
        description: "结束日期 (YYYY-MM-DD)"
        defaultFunc: "time.Now().Format(\"2006-01-02\")"
```

### 示例 3：带位置参数的创建接口

**1. OpenAPI 定义**:

```yaml
  /cc/create_task:
    post:
      summary: 创建外呼任务
      operationId: createTask
      tags: [Task]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                phone:
                  type: string
                description:
                  type: string
```

**2. 配置映射**:

```yaml
endpoints:
  createTask:
    command: ["task", "create"]
    description: "创建外呼任务"
    use: "task create <phone>"
    resultType: "simple"
    args:
      - name: phone
        description: "目标电话号码"
        required: true
    flags:
      - param: phone
        flag: ""
        source: arg
      - param: description
        flag: desc
        shorthand: d
        type: string
        description: "任务描述"
```

---

## 可用 Make 命令

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
```

## 脚本命令

```bash
./scripts/clink-generate.sh all     # 生成所有代码
./scripts/clink-generate.sh cli     # 只生成 CLI
./scripts/clink-generate.sh api     # 只生成 API
./scripts/clink-generate.sh check   # 检查配置一致性
./scripts/clink-generate.sh stats   # 显示统计信息
```

---

## 注意事项

1. **operationId 必须匹配**：配置中的 endpoint key 必须与 OpenAPI 中的 `operationId` 完全一致

2. **命令冲突避免**：
   - 单级命令：`["agents"]` 直接注册到 root
   - 多级命令：`["records", "inbound"]` 先注册 records，再添加 inbound 子命令
   - 避免同一命令既是单级又是多级父命令

3. **类型覆盖**：如果 OpenAPI 类型推断不正确，使用 `type` 字段覆盖

4. **关键字避免**：变量名会自动处理 Go 关键字（如 `type` → `typeVal`）

5. **重新生成**：每次修改配置后必须运行 `make generate` 重新生成代码

---

## 配置检查

验证配置是否正确：

```bash
./scripts/clink-generate.sh check
```

输出示例：
```
Configured endpoints:
  ✓ listCdrIbs
  ✓ listCdrObs
  ✗ myOldEndpoint (not in OpenAPI spec)

Available in OpenAPI but not configured:
  • describeClient
  • statClient
```

## 架构图

```
api/openapi.yaml         完整的 OpenAPI 规范
      ↓
config/generator.yaml    只生成配置中定义的接口
      ↓
scripts/clink-generator  生成 CLI 命令代码
      ↓
cmd/clink/*_gen.go       生成的 CLI 命令文件
```

## 模板系统

代码生成基于 Go template，支持以下模板：

- `rootTemplate` - 根命令注册
- `commandTemplate` - 多级命令
- `singleCommandTemplate` - 单级命令

自定义模板需要修改 `scripts/clink-generator/main.go` 中的 template 定义。

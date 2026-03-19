# Clink CLI 代码生成器改进计划 (v2.1)

> 制定日期：2026-03-19  
> 版本：2.1（基于审核结果修订）  
> 状态：已审核，待实施

---

## 执行摘要

基于对 `api-generator-v2` 的详细审核，本计划从"重新设计"调整为"**在现有基础上改进**"。

**关键结论：**
- api-generator-v2 是合理的半成品框架，核心问题是"**运行时库未集成**"
- 重新设计成本 2-3 周，改进现有成本 1.5-2 周
- 采用分阶段改进策略，每阶段都有可用产出

---

## 1. 现状复盘

### 1.1 已完成的资产

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| **API 生成器框架** | `scripts/api-generator-v2/main.go` | ⚠️ 半成品 | 基础框架合理，功能待完善 |
| **运行时库** | `pkg/codegen/*.go` | ⚠️ 未使用 | transformer、request_builder、response_parser 等已编写 |
| **类型定义** | `pkg/codegen/types.go` | ✅ 完整 | GeneratorConfig、EndpointConfig 等定义齐全 |
| **配置 v2** | `config/generator.v2.yaml` | ✅ 完整 | 详细配置了 14 个端点 |
| **生成产物** | `pkg/api/auto_generated.go` | ⚠️ 可用但需改进 | 已生成但命名不规范、功能缺失 |

### 1.2 核心问题清单

**生成器问题（scripts/api-generator-v2/）：**
1. ❌ **运行时库未调用** - 已编写的 `pkg/codegen/` 完全没使用
2. ❌ **useTypes 未实现** - TimeRange、Pagination 类型继承未展开
3. ❌ **请求转换未实现** - 日期→时间戳转换仅标记，无实际代码
4. ❌ **响应处理简陋** - 统一返回 `interface{}`，无字段映射
5. ⚠️ **命名不规范** - `request_id` 含连字符、`bind_type` 下划线命名
6. ⚠️ **模板空行过多** - 生成代码格式混乱

**生成产物问题（pkg/api/auto_generated.go）：**
```go
// 问题 1：参数名含连字符
func (a *GeneratedAPI) Webcall(ctx context.Context, ..., request_id string, ...)

// 问题 2：下划线命名
func (a *GeneratedAPI) Online(ctx context.Context, ..., bind_type int)

// 问题 3：空行混乱
params := &generated.OnlineJSONRequestBody{}


params.Cno = agent
```

### 1.3 运行时库现状

`pkg/codegen/` 下已实现的运行时支持（但生成器未调用）：

| 文件 | 功能 | 状态 |
|------|------|------|
| `transformer.go` | 时间/日期转换、类型转换 | ✅ 已实现 |
| `request_builder.go` | 请求参数构建 | ✅ 已实现 |
| `response_parser.go` | 响应解析、字段映射 | ✅ 已实现 |
| `error_handler.go` | 错误处理、重试策略 | ✅ 已实现 |
| `renderer_adapter.go` | 渲染适配 | ✅ 已实现 |
| `*_test.go` | 单元测试 | ✅ 已编写 |

---

## 2. 改进策略

### 2.1 策略选择：改进而非重写

**对比：**

| 维度 | 改进现有 | 重新设计 |
|------|----------|----------|
| 时间成本 | **1.5-2 周** | 2-3 周 |
| 风险 | **低** | 中高 |
| 每阶段产出 | **可用** | 后期才可用 |
| 现有资产利用 | **充分** | 浪费 |

**理由：**
1. 框架结构合理，核心问题只是"未集成运行时库"
2. 运行时库已编写且质量合格，直接利用
3. 渐进式改进风险可控，可及时发现问题

### 2.2 改进原则

1. **先修复，后增强** - 先解决命名、格式问题，再添加功能
2. **编译验证** - 每阶段生成代码必须编译通过
3. **逐步集成运行时库** - 按优先级逐个集成，而非一次性全部
4. **保持向后兼容** - 已有配置继续支持

---

## 3. 实施路线图

### Phase 1：基础修复（2 天）

**目标：** 修复生成代码的基础质量问题

**任务清单：**

- [ ] **修复命名规范**
  - [ ] 完善 `toVarName()` 函数，完整处理 Go 关键字
    ```go
    var goKeywords = map[string]string{
        "type": "typeVal",
        "range": "rangeVal", 
        "map": "mapVal",
        "chan": "chanVal",
        "interface": "interfaceVal",
        // ... 完整列表
    }
    ```
  - [ ] 连字符转驼峰：`request-id` → `requestId`
  - [ ] 下划线转驼峰：`bind_type` → `bindType`
  - [ ] 添加命名规范检查单元测试

- [ ] **修复模板空行**
  - [ ] 重构模板，使用 `{{- -}}` 控制空白
  - [ ] 添加 `gofmt` 后处理步骤

- [ ] **添加编译验证**
  - [ ] Makefile 中增加 `make generate-verify`
  - [ ] CI 中验证生成代码可编译

**验收标准：**
```bash
# L1 单元测试通过
$ go test ./pkg/codegen/... -v
ok      github.com/raymondtc/clink-cli/pkg/codegen

# 编译验证
$ make generate
$ go build ./pkg/api/...  # 无错误
$ go build ./cmd/...      # 无错误

# 格式检查
$ gofmt -l pkg/api/auto_generated.go  # 无输出（表示已格式化）
$ golint ./pkg/api/...  # 无命名规范警告
```

---

### Phase 2：类型继承（3 天）

**目标：** 实现 `useTypes` 配置展开

**任务清单：**

- [ ] **实现类型展开逻辑**
  ```go
  // 在 buildMethod 中展开 useTypes
  for _, typeName := range endpoint.Parameters.UseTypes {
      typeDef := g.Config.Types[typeName]
      for fieldName, fieldDef := range typeDef {
          // 展开为 APIParam
      }
  }
  ```

- [ ] **支持复用类型**
  - [ ] `TimeRange` → `startTime`, `endTime`
  - [ ] `Pagination` → `offset`, `limit`

- [ ] **添加配置验证**
  - [ ] 检查 useTypes 引用的类型是否存在
  - [ ] 检查字段名冲突
  - [ ] 提供清晰的错误信息

- [ ] **测试覆盖**
  - [ ] L1: 类型展开逻辑单元测试
  - [ ] L2: 为每个端点生成代码并验证编译
  - [ ] 测试类型展开后的参数顺序
  - [ ] 配置验证错误提示测试

**验收标准：**
```bash
# L1 单元测试
$ go test ./pkg/codegen/... -run TestTypeExpansion -v

# 类型展开验证
$ make generate
# 检查 listCdrIbs 方法参数包含 startTime, endTime, offset, limit
$ grep -A 5 "func (a \*GeneratedAPI) ListCdrIbs" pkg/api/auto_generated.go

# 编译验证
$ go build ./pkg/api/...

# 配置验证测试
$ go test ./scripts/api-generator-v2/... -run TestConfigValidation -v
```

---
```yaml
endpoints:
  listCdrIbs:
    parameters:
      useTypes: ["TimeRange", "Pagination"]  # 应展开为 4 个参数
      fields:
        - name: customerNumber  # 额外参数
```

---

### Phase 3：运行时库集成（3 天）

**目标：** 集成运行时库，实现请求转换

**任务清单：**

- [ ] **重构模板架构**
  - [ ] 从"内嵌逻辑"改为"调用运行时库"
  - [ ] 生成代码示例：
    ```go
    func (a *GeneratedAPI) ListCdrIbs(ctx context.Context, params ListCdrIbsParams) (*CdrIbListResult, error) {
        // 使用运行时库构建请求
        req := runtime.BuildRequest(params, endpointCfg)
        resp, err := a.client.Do(ctx, req)
        // ...
    }
    ```

- [ ] **集成 transformer（时间转换）**
  - [ ] 日期 → 时间戳转换
  - [ ] 时区处理（Asia/Shanghai）
  - [ ] EndOfDay 支持（23:59:59）

- [ ] **集成 request_builder**
  - [ ] 参数验证
  - [ ] 必填字段检查
  - [ ] 默认值处理

- [ ] **更新运行时库接口**
  - [ ] 确保运行时库接口与生成器匹配
  - [ ] 添加必要的适配层

**验收标准：**
```bash
# L1: transformer 单元测试
$ go test ./pkg/codegen/... -run TestTimeTransform -v

# L2: 生成代码调用运行时库
$ grep -r "runtime.BuildRequest\|runtime.ApplyTransforms" pkg/api/auto_generated.go

# 编译验证
$ go build ./pkg/api/...

# L3: E2E 测试（使用 mock）
$ CLINK_MOCK=true go test ./pkg/api/... -run TestListCdrIbs -v
```

---

### Phase 4：响应处理（2 天）

**目标：** 实现响应映射和类型安全

**任务清单：**

- [ ] **生成具体返回类型**
  - [ ] 从 `interface{}` 改为具体类型
  - [ ] 根据 `response.mapping` 生成结果结构体

- [ ] **集成 response_parser**
  - [ ] 字段映射（API 字段 → 输出字段）
  - [ ] 枚举转换（`0` → `"离线"`）
  - [ ] 时间格式化

- [ ] **错误处理增强**
  - [ ] 集成 error_handler
  - [ ] 按错误码不同策略（retry/return）

**验收标准：**
```bash
# L1: response_parser 单元测试
$ go test ./pkg/codegen/... -run TestResponseMapping -v

# L2: 生成代码返回具体类型（非 interface{}）
$ grep "func (a \*GeneratedAPI)" pkg/api/auto_generated.go | grep -v "interface{}"
# 应看到具体返回类型如 *CdrIbListResult

# 字段映射验证
$ CLINK_MOCK=true go test ./pkg/api/... -run TestResponseFieldMapping -v
```

---

### Phase 5：完善与优化（2 天）

**目标：** 生产就绪

**任务清单：**

- [ ] **分页自动处理**
  - [ ] 自动翻页（当返回 hasMore 时继续请求）
  - [ ] 支持 offset/limit 和 pageNum/pageSize 两种模式

- [ ] **条件逻辑**
  - [ ] 实现 `webcall` / `makecall` 条件切换

- [ ] **测试覆盖**
  - [ ] 生成器本身的单元测试
  - [ ] 集成测试（端到端）

- [ ] **文档同步**
  - [ ] 更新 `docs/API.md`
  - [ ] 添加配置编写指南

**验收标准：**
```bash
# L1: 所有单元测试通过
$ go test ./pkg/codegen/... -v
ok      github.com/raymondtc/clink-cli/pkg/codegen    0.5s

# L2: 所有生成代码测试通过  
$ go test ./pkg/api/... -v
ok      github.com/raymondtc/clink-cli/pkg/api        1.2s

# L3: 完整 E2E 测试
$ CLINK_MOCK=true go test ./... -v
ok      github.com/raymondtc/clink-cli/...            2.5s

# 代码覆盖率检查（目标：>70%）
$ go test ./pkg/codegen/... ./pkg/api/... -cover
ok      github.com/raymondtc/clink-cli/pkg/codegen    0.5s    coverage: 75.3%
ok      github.com/raymondtc/clink-cli/pkg/api        1.2s    coverage: 68.9%

# 最终验证
$ make generate-verify
✓ Generated code compiles
✓ All tests pass
✓ Code coverage >= 70%
```

---

## 4. 技术实现细节

### 4.1 命名规范修复

```go
// pkg/codegen/naming.go（新增）
package codegen

// GoKeywords 需要转义的 Go 关键字
var GoKeywords = map[string]string{
    "type":      "typeVal",
    "range":     "rangeVal",
    "map":       "mapVal",
    "chan":      "chanVal",
    "interface": "interfaceVal",
    "func":      "funcVal",
    "var":       "varVal",
    "const":     "constVal",
    // ... 完整列表
}

// ToValidIdentifier 将任意字符串转为合法 Go 标识符
func ToValidIdentifier(s string) string {
    // 1. 替换连字符和空格为下划线
    s = strings.ReplaceAll(s, "-", "_")
    s = strings.ReplaceAll(s, " ", "_")
    
    // 2. 转为驼峰
    s = toCamelCase(s)
    
    // 3. 检查关键字
    if replacement, ok := GoKeywords[s]; ok {
        return replacement
    }
    
    return s
}

// toCamelCase 下划线命名转驼峰
func toCamelCase(s string) string {
    parts := strings.Split(s, "_")
    for i, p := range parts {
        if i == 0 {
            parts[i] = strings.ToLower(p)
        } else {
            parts[i] = strings.Title(strings.ToLower(p))
        }
    }
    return strings.Join(parts, "")
}
```

### 4.2 模板重构

```go
// 重构后的模板，减少空行，使用运行时库
const apiMethodTemplate = `// {{.Name}} - {{.Description}}
func (a *GeneratedAPI) {{.Name}}(ctx context.Context{{range .Params}}, {{.VarName}} {{.Type}}{{end}}) ({{.ReturnType}}, error) {
    // 构建请求
    req := &{{.GeneratedParamsType}}{}
    {{range .Params -}}
    {{if .Required}}req.{{.FieldName}} = {{.VarName}}{{else}}if {{.VarName}} != {{.ZeroValue}} {
        req.{{.FieldName}} = &{{.VarName}}
    }{{end}}
    {{end -}}
    
    // 应用请求转换
    {{if .HasTransforms -}}
    if err := runtime.ApplyTransforms(req, {{.TransformConfig}}); err != nil {
        return nil, err
    }
    {{end -}}
    
    // 调用 API
    resp, err := a.client.{{.ClientMethodName}}(ctx, {{if .IsPOST}}*{{end}}req)
    if err != nil {
        return nil, fmt.Errorf("{{.OperationID}}: %w", err)
    }
    
    // 解析响应
    result, err := runtime.ParseResponse[{{.ReturnType}}](resp, {{.ResponseConfig}})
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
`
```

### 4.3 Makefile 更新

```makefile
# 生成并验证
generate-verify: generate
	@echo "Verifying generated code..."
	@go build ./pkg/api/...
	@go build ./cmd/...
	@echo "Running gofmt check..."
	@test -z "$$(gofmt -l pkg/api/auto_generated.go cmd/clink/*_gen.go)" || (echo "Format errors found"; exit 1)
	@echo "✓ Verification passed"

# CI 使用
ci: generate-verify test
```

---

## 5. 测试验证策略

每个 Phase 必须包含以下测试环节：

### 5.1 三层测试模型

```
┌─────────────────────────────────────────────────────────┐
│  L3: 端到端集成测试 (E2E)                                 │
│  - 生成器 → 生成代码 → 编译 → 运行                        │
│  - 使用真实/模拟 API 响应验证                              │
├─────────────────────────────────────────────────────────┤
│  L2: 生成代码单元测试                                     │
│  - 对 auto_generated.go 生成单元测试                       │
│  - 验证参数绑定、转换逻辑                                  │
├─────────────────────────────────────────────────────────┤
│  L1: 生成器单元测试                                       │
│  - 测试 naming、transform 等工具函数                       │
│  - 测试模板渲染                                          │
└─────────────────────────────────────────────────────────┘
```

### 5.2 每阶段测试要求

| 测试类型 | Phase 1 | Phase 2 | Phase 3 | Phase 4 | Phase 5 |
|----------|---------|---------|---------|---------|---------|
| L1 单元测试 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 |
| 编译验证 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 |
| 格式检查 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 | ✅ 必须 |
| L2 生成代码测试 | - | ⚠️ 推荐 | ✅ 必须 | ✅ 必须 | ✅ 必须 |
| L3 E2E 测试 | - | - | ⚠️ 推荐 | ⚠️ 推荐 | ✅ 必须 |

### 5.3 测试用例设计

**L1 测试示例：**
```go
// pkg/codegen/naming_test.go
func TestToValidIdentifier(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"request-id", "requestId"},
        {"bind_type", "bindType"},
        {"type", "typeVal"},  // 关键字
        {"customer-number", "customerNumber"},
    }
    // ...
}
```

**L2 测试示例：**
```go
// pkg/api/auto_generated_test.go (生成)
func TestListCdrIbs_TimeTransform(t *testing.T) {
    api := NewGeneratedAPI(...)
    result, err := api.ListCdrIbs(ctx, "2024-01-01", "2024-01-02", "", "")
    // 验证时间已转换为时间戳
}
```

**L3 E2E 测试：**
```bash
# 生成代码并使用 mock 响应验证
CLINK_MOCK=true go test ./pkg/api/... -v
```

### 5.4 CI/CD 集成

```yaml
# .github/workflows/generate.yml
- name: Generate Code
  run: make generate

- name: Verify Generated Code
  run: make generate-verify

- name: Run L1 Tests
  run: go test ./pkg/codegen/... -v

- name: Run L2 Tests  
  run: go test ./pkg/api/... -v

- name: E2E Test with Mock
  run: CLINK_MOCK=true go test ./... -v
```

---

## 6. 风险与应对

| 风险 | 可能性 | 影响 | 应对 |
|------|--------|------|------|
| 运行时库接口不匹配 | 中 | 高 | Phase 3 前审计运行时库接口，必要时添加适配层 |
| 类型展开后参数顺序混乱 | 中 | 中 | 添加配置验证，确保参数顺序可预期 |
| 生成代码编译失败 | 低 | 高 | 每阶段强制编译验证 |
| 与 oapi-codegen 生成代码冲突 | 低 | 中 | 保持命名空间隔离 |
| 测试覆盖不足导致回归 | 中 | 中 | 强制 L1/L2 测试，每 Phase 验收 |

---

## 6. 下一步行动

### 立即执行（今天）

1. [ ] 创建 feature/codegen-improvement 分支
2. [ ] 编写 `pkg/codegen/naming.go` 命名规范模块
3. [ ] 修复 `toVarName()` 函数的关键字处理

### 本周目标

1. [ ] 完成 Phase 1（基础修复）
2. [ ] 开始 Phase 2（类型继承）

### 需要决策

- [ ] 是否需要在生成器中调用 `gofmt` 自动格式化？
- [ ] 是否保留 v1 配置支持，还是完全迁移到 v2？
- [ ] 错误处理策略：运行时库返回错误 vs 生成器内嵌错误处理代码？
- [ ] **测试策略**：是否要求每个生成的方法都有对应的 L2 测试？
- [ ] **覆盖率目标**：L1/L2 测试覆盖率目标设定（建议 >= 70%）？
---

## 附录：参考文件

| 文件 | 说明 |
|------|------|
| `scripts/api-generator-v2/main.go` | API 生成器实现 |
| `pkg/codegen/*.go` | 运行时库 |
| `config/generator.v2.yaml` | 详细配置 |
| `pkg/api/auto_generated.go` | 当前生成产物 |

---

*最后更新：2026-03-19*  
*审核人：Claude Code*  
*版本：2.1*

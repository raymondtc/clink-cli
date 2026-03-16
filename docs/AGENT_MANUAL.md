# Clink Agent 使用手册

本文档供 AI Agent 使用，介绍如何通过 Clink 工具操作天润融通呼叫中心。

## 简介

Clink 是一个为天润融通呼叫中心提供的 AI 友好型工具集，包含：
- **CLI 工具**: `clink` 命令行
- **MCP Server**: `clink-mcp` 供 Agent 调用

## 快速开始

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/raymondtc/clink-cli/main/install.sh | bash
```

安装完成后，配置环境变量：
```bash
export CLINK_ACCESS_ID="your_access_key_id"
export CLINK_ACCESS_SECRET="your_secret"
```

## 可用工具

### 1. get_inbound_records - 获取呼入通话记录

查询指定时间范围内的呼入通话记录。

**参数**:
- `start_time` (必填): 开始时间，格式 `YYYY-MM-DD` 或 `YYYY-MM-DD HH:MM:SS`
- `end_time` (必填): 结束时间，格式同上
- `phone` (可选): 按客户号码筛选
- `agent_id` (可选): 按座席号筛选

**使用示例**:
```
用户: "查询昨天所有呼入电话"
→ 调用 get_inbound_records
   start_time: "2024-03-15"
   end_time: "2024-03-15"

用户: "查一下座席 0281 最近一周的呼入记录"
→ 调用 get_inbound_records
   start_time: "2024-03-09"
   end_time: "2024-03-15"
   agent_id: "0281"
```

### 2. get_agent_status - 查询座席状态

获取座席的实时状态（在线/忙碌/离线等）。

**参数**:
- `agent_id` (可选): 指定座席号，不提供则返回所有座席

**使用示例**:
```
用户: "现在有多少座席在线？"
→ 调用 get_agent_status
→ 统计返回结果中状态为 "online" 或 "空闲" 的数量

用户: "座席 0281 在忙吗？"
→ 调用 get_agent_status
   agent_id: "0281"
```

### 3. make_call / webcall - 发起电话呼叫 ⚠️

**当用户说"打电话"、"呼叫"、"联系"时，使用此工具。**

**默认使用 WebCall 方式，无需座席 ID！**

发起外呼电话或 WebCall。

**参数**:
- `phone` (必填): 要拨打的电话号码
- `agent_id` (可选): 指定座席号（如需特定座席外呼，否则留空使用 WebCall）
- `display_number` (可选): 外显号码

**使用示例**:
```
用户: "给 18512854639 打个电话"
→ 调用 make_call
   phone: "18512854639"
   （不填 agent_id，自动使用 WebCall）

用户: "呼叫客户 13800138000"
→ 调用 make_call
   phone: "13800138000"
   （不填 agent_id）

用户: "用座席 0281 给客户打电话"
→ 调用 make_call
   phone: "客户号码"
   agent_id: "0281"  （指定座席）
```

**工作原理**:
- 如果不填 `agent_id` → 使用 `/cc/webcall`（WebCall，无需座席）
- 如果填写 `agent_id` → 使用 `/cc/callout`（指定座席外呼）

**注意事项**:
- 默认使用 WebCall，不需要座席在线
- 如需指定座席，请先调用 `get_agent_status` 确认座席状态
- 呼叫需要真实的 API 凭证和企业权限

## 组合使用示例

### 场景 1: 智能外呼任务

```
用户: "给这批客户打电话，通知他们活动信息"

执行步骤:
1. 用户上传客户列表（CSV 格式，包含 phone, name）
2. 调用 get_agent_status 获取在线座席列表
3. 为每个客户:
   a. 分配一个空闲座席
   b. 调用 make_call 发起呼叫
   c. 记录 call_id 和状态
4. 生成外呼任务报告
```

### 场景 2: 来电后自动回拨

```
用户: "刚才漏接的电话，帮我回拨过去"

执行步骤:
1. 调用 get_inbound_records 获取最近的呼入记录
2. 筛选状态为 "missed" 或 "未接" 的记录
3. 获取客户号码
4. 调用 make_call 发起回拨
5. 通知座席回拨已发起
```

### 场景 3: 呼叫中心监控

```
用户: "监控今天的接通率，如果低于 80% 提醒我"

执行步骤:
1. 循环执行（每 30 分钟）:
   a. 调用 get_inbound_records 获取今日呼入记录
   b. 计算接通率 = 接通数 / 总数
   c. 如果接通率 < 80%:
      - 调用 get_agent_status 获取座席状态
      - 分析原因（座席不足/队列过长等）
      - 发送告警通知
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 必需 |
|--------|------|------|
| CLINK_ACCESS_ID / CLINK_ACCESS_KEY_ID | Access Key ID | ✅ |
| CLINK_ACCESS_SECRET / CLINK_SECRET | Access Secret | ✅ |
| CLINK_ENTERPRISE_ID | 企业 ID（部分 API 需要）| ❌ |

### 平台地址

- 北京平台: `https://api-bj.clink.cn`
- 上海平台: `https://api-sh.clink.cn`

## 最佳实践

1. **时间范围**: 查询通话记录时，时间范围不要超过一个月
2. **错误处理**: 遇到 401 错误检查凭证；遇到 400 检查参数格式
3. **频率限制**: 避免短时间内大量调用，遵守 API 限流规则
4. **数据安全**: 通话记录包含敏感信息，注意保密

## 故障排查

**问题**: 返回 "Missing API key found in request"
**解决**: 检查 AccessKeyId 和 AccessSecret 是否正确

**问题**: 返回 "时间范围不能超过一个月"
**解决**: 缩小查询的时间范围

**问题**: 座席查询返回空列表
**解决**: 检查 enterpriseId 是否正确（如需要）

**问题**: 无法发起呼叫
**解决**: 确认账号有外呼权限，且座席处于在线状态

## 参考链接

- [天润融通 API 开发指南](docs/API开发指南.html)
- [项目地址](https://github.com/raymondtc/clink-cli)

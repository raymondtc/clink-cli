# Clink Agent 使用手册

本文档供 AI Agent 使用，介绍如何通过 Clink Go 工具操作天润融通呼叫中心。

## 简介

Clink 是一个为天润融通呼叫中心提供的 AI 友好型工具集（Go 版本），包含：
- **CLI 工具**: `clink` 命令行
- **MCP Server**: `clink-mcp` 供 Agent 调用

## 配置

在使用前，需要配置以下环境变量：

```bash
export CLINK_ACCESS_ID="你的access_id"
export CLINK_ACCESS_SECRET="你的access_secret"
export CLINK_ENTERPRISE_ID="你的enterprise_id"
```

或在 OpenClaw 中配置 MCP Server：

```json
{
  "mcpServers": {
    "clink": {
      "command": "/usr/local/bin/clink-mcp",
      "env": {
        "CLINK_ACCESS_ID": "your_access_id",
        "CLINK_ACCESS_SECRET": "your_access_secret",
        "CLINK_ENTERPRISE_ID": "your_enterprise_id"
      }
    }
  }
}
```

## 可用工具

### 1. get_inbound_records - 获取呼入记录

查询指定时间范围内的呼入通话记录。

**参数**:
- `start_time` (必填): 开始时间，格式 `YYYY-MM-DD HH:MM:SS`
- `end_time` (必填): 结束时间，格式 `YYYY-MM-DD HH:MM:SS`
- `phone` (可选): 按电话号码筛选
- `agent_id` (可选): 按座席ID筛选
- `page` (可选): 页码，默认1
- `page_size` (可选): 每页数量，默认50

**使用示例**:
```
用户: "查询昨天所有呼入电话"
→ 调用 get_inbound_records
   start_time: "2024-01-15 00:00:00"
   end_time: "2024-01-15 23:59:59"

用户: "查一下张三接的最近10个电话"
→ 调用 get_inbound_records
   start_time: "2024-01-01 00:00:00"
   end_time: "2024-01-15 23:59:59"
   agent_id: "1001"
   page_size: 10
```

### 2. get_outbound_records - 获取外呼记录

查询指定时间范围内的外呼通话记录。

**参数**: 同 get_inbound_records

**使用示例**:
```
用户: "看看这周一共打了多少电话"
→ 调用 get_outbound_records
   start_time: "2024-01-08 00:00:00"
   end_time: "2024-01-14 23:59:59"
```

### 3. get_agent_status - 查询座席状态

获取座席的实时状态（在线/忙碌/离线）。

**参数**:
- `agent_id` (可选): 指定座席ID，不提供则返回所有座席

**使用示例**:
```
用户: "现在有多少座席在线？"
→ 调用 get_agent_status
→ 统计返回结果中 status 为 "online" 的数量

用户: "张三在忙吗？"
→ 调用 get_agent_status
   agent_id: "1001"
```

### 4. make_call - 发起外呼

让指定座席拨打一个电话。

**参数**:
- `phone` (必填): 要拨打的电话号码
- `agent_id` (必填): 执行外呼的座席ID
- `display_number` (可选): 外显号码

**使用示例**:
```
用户: "让张三给客户 13800138000 回电话"
→ 调用 make_call
   phone: "13800138000"
   agent_id: "1001"
```

### 5. get_queue_status - 查询队列状态

获取呼叫队列的实时状态（等待人数、平均等待时间等）。

**参数**:
- `queue_id` (可选): 队列ID，不提供则返回所有队列

**使用示例**:
```
用户: "现在有多少人在排队？"
→ 调用 get_queue_status
→ 返回 waitingCount 字段

用户: "客服队列的平均等待时间是多久？"
→ 调用 get_queue_status
→ 返回 avgWaitTime 字段
```

## 组合使用示例

### 场景1: 生成通话报表

```
用户: "给我一份上周的通话统计报表"

执行步骤:
1. 调用 get_inbound_records 获取上周呼入记录
2. 调用 get_outbound_records 获取上周外呼记录
3. 统计以下指标:
   - 总呼入数 / 接通数 / 接通率
   - 总外呼数 / 接通数 / 接通率
   - 平均通话时长
   - 按座席分布
4. 生成报表
```

### 场景2: 监控告警

```
用户: "监控接通率，如果低于80%就告诉我"

执行步骤:
1. 循环执行（每5分钟）:
   a. 调用 get_inbound_records 获取最近1小时记录
   b. 计算接通率 = 接通数 / 总数
   c. 如果接通率 < 80%:
      - 发送告警通知
      - 调用 get_queue_status 获取队列状态
      - 调用 get_agent_status 获取座席状态
      - 分析原因并报告
```

### 场景3: 批量外呼任务

```
用户: "给这批客户打电话"

执行步骤:
1. 用户上传 CSV 文件（包含 phone, name 等字段）
2. 调用 get_agent_status 查询可用座席
3. 为每个客户:
   a. 调用 make_call 发起外呼
   b. 记录 call_id
   c. 等待一段时间后调用 get_outbound_records 查询结果
4. 生成外呼结果报告
```

## 最佳实践

1. **时间范围**: 查询时建议单次不超过7天，大数据量分页获取
2. **错误处理**: API 返回 code != 200 时表示错误，需要处理
3. **性能**: 批量操作时适当添加延迟，避免触发限流
4. **数据安全**: 通话记录包含敏感信息，注意保密

## 常见问题

**Q: 如何获取真实的 API 凭证？**
A: 登录天润融通后台，在「系统设置-安全设置-接口密钥」中生成。

**Q: 录音文件格式是什么？**
A: 通常为 MP3 格式，可以通过 recordingUrl 下载。

**Q: 座席状态有哪些？**
A: online（在线）、busy（通话中）、offline（离线）、pause（暂停）

**Q: 如何安装 clink-mcp？**
A: 从 GitHub Releases 下载对应平台的二进制文件，放到 PATH 中即可。

# Clink CLI - Agent 使用手册

> 本手册专为 AI Agent 设计，提供自然语言操作 Clink CLI 的完整指南
> 
> 版本: v0.2.0 | 更新: 2026-03-25

---

## 🚀 快速开始

### 1. 环境准备

Agent 在执行命令前，需要确认以下环境变量已设置：

```bash
# 必需
export CLINK_ACCESS_ID="your_access_id"
export CLINK_ACCESS_SECRET="your_access_secret"

# 可选（默认上海节点）
export CLINK_BASE_URL="https://api-sh.clink.cn"  # 或 https://api-bj.clink.cn
```

### 2. 基本使用模式

```
clink [全局选项] <命令> [子命令] [选项]
```

**全局选项：**
- `--access-id` / `--access-secret`: API 认证
- `--base-url`: API 节点地址
- `--output, -o`: 输出格式 (`table`, `json`)

---

## 📞 通话记录管理

### 查询呼入记录

**Agent 意图**: "查一下最近 7 天的呼入通话记录"

```bash
clink records inbound
# 默认查询最近 7 天

clink records inbound --start 2024-03-01 --end 2024-03-25
# 指定日期范围

clink records inbound --phone 13800138000
# 按客户号码筛选

clink records inbound --agent 1001 --limit 100
# 按座席筛选，返回 100 条
```

**常用场景对话：**
- "查一下 13800138000 这个号码的通话记录" → `clink records inbound --phone 13800138000`
- "昨天座席 1001 接了多少电话" → `clink records inbound --agent 1001 --start $(date -d yesterday +%Y-%m-%d)`
- "导出本月所有呼入记录" → `clink records inbound --start 2024-03-01 --output json > inbound.json`

### 查询外呼记录

**Agent 意图**: "看看最近的外呼情况"

```bash
clink records outbound
# 最近 7 天外呼记录

clink records outbound --start 2024-03-20 --phone 13900139000
# 特定号码外呼记录
```

### 查询当日通话记录

**Agent 意图**: "今天座席 1001 接了多少电话"

```bash
clink records today --agent 1001
# 座席今日所有通话

clink records today --agent 1001 --limit 100
# 查询最近 100 条
```

**输出示例：**
```
通话ID                客户号码     类型  时间       时长
20240325120001-abc123 138****8000  呼入  09:32:15   03:45
20240325120002-def456 139****9000  外呼  10:15:30   01:20
```

### 查询历史通话记录

**Agent 意图**: "查一下过去一个月的通话记录"

```bash
clink records history --start 30d
# 最近 30 天所有通话记录

clink records history --agents 1001,1002,1003 --type ib
# 多个座席的呼入记录

clink records history --phone 13800138000 --start 2024-01-01 --end 2024-03-25
# 特定号码的历史记录
```

### 查询满意度记录

**Agent 意图**: "看一下客户的满意度评价"

```bash
clink records satisfaction --start 2024-03-01
# 指定时间范围的满意度记录

clink records satisfaction --agent 1001
# 特定座席的满意度

clink records satisfaction --call 20240325120001-abc123
# 特定通话的满意度
```

**输出示例：**
```
满意度ID       通话ID                座席   评价方式   评分  评价时间
SAT-12345678   20240325120001-abc123 1001   按键评价   5     03-25 09:35
SAT-12345679   20240325120002-def456 1002   短信评价   4     03-25 10:18
```

### 获取录音下载链接

**Agent 意图**: "获取通话录音的下载链接"

```bash
clink records url 20240325120001-abc123
# 获取录音 URL

clink records url 20240325120001-abc123 --side 1
# 客户侧录音

clink records url 20240325120001-abc123 --side 2 --timeout 7200
# 座席侧录音，链接有效期 2 小时

clink records url 20240325120001-abc123 --download 0
# 试听链接（非下载）
```

**输出示例：**
```json
{
  "url": "https://record.clink.cn/xxx/record.mp3?token=abc123",
  "expireAt": "2024-03-25T12:00:00Z"
}
```

### 下载通话录音

**Agent 意图**: "下载今天的通话录音"

```bash
# 下载到当前目录
clink records download 20240325120001-abc123

# 指定保存路径
clink records download 20240325120001-abc123 --output ./recordings/

# 下载座席侧录音
clink records download 20240325120001-abc123 --side 2 --output ./agent-records/

# 批量下载（结合 records today 或 inbound）
clink records today --agent 1001 -o json | jq -r '.[].callId' | xargs -I {} clink records download {} --output ./today/
```

**常用场景对话：**
- "下载昨天的所有录音" → `clink records history --start 1d --end now -o json | jq -r '.[].callId' | xargs -I {} clink records download {}`
- "获取客户侧录音链接" → `clink records url <call-id> --side 1`
- "查一下这通电话的满意度" → `clink records satisfaction --call <call-id>`

---

## 👤 座席管理

### 查询座席状态

**Agent 意图**: "现在有哪些座席在线"

```bash
clink agents
# 所有座席状态

clink agents --agent 1001
# 特定座席状态
```

**输出示例：**
```
工号  姓名    状态    当前通话  今日接听  今日外呼
1001  张三    🟢 在线  通话中    25        3
1002  李四    🟡 置忙  -         18        5
1003  王五    ⚫ 离线  -         0         0
```

### 座席登录/登出

**Agent 意图**: "让座席 1001 上线"

```bash
# 登录上线
clink agent online --agent 1001 --queue 8001 --tel 13800138001

# 简单上线（使用默认配置）
clink agent online --agent 1001

# 登出下线
clink agent offline --agent 1001
```

### 座席状态控制

**Agent 意图**: "座席 1001 去休息，置忙一下"

```bash
# 置忙（Pause）
clink agent pause --agent 1001 --type 1 --reason "休息中"

# 置闲（Ready）
clink agent ready --agent 1001
```

**常用场景对话：**
- "座席 1001 下线了" → `clink agent offline --agent 1001`
- "让张三开始接电话" → 先查询座席号 → `clink agent online --agent <cno>`
- "暂停服务 10 分钟" → `clink agent pause --agent 1001 --reason "系统维护"`

---

## 📊 队列管理

### 查询队列状态

**Agent 意图**: "看一下队列情况，有多少人在等"

```bash
clink queue status
# 所有队列状态

clink queue status --queue 8001,8002
# 特定队列
```

**输出示例：**
```
队列  名称        在线座席  等待客户  平均等待
8001  客服一部    5         12        45s
8002  客服二部    3         8         32s
```

### 查询队列列表

```bash
clink queue list
# 队列配置列表

clink queue list --limit 50
# 分页查询
```

---

## 📱 呼叫控制

### 发起外呼

**Agent 意图**: "给 13800138000 打个电话"

```bash
# WebCall（无需座席，系统直接呼叫客户）
clink call 13800138000

# WebCall 高级选项
clink call 13800138000 --ivr "工作时间" --caller 01012345678

# 指定座席外呼
clink call agent 13800138000 --agent 1001
```

**常用场景对话：**
- "回拨给客户 13800138000" → `clink call 13800138000`
- "让座席 1001 打给客户" → `clink call agent <phone> --agent 1001`
- "用销售 IVR 流程呼叫" → `clink call <phone> --ivr "销售流程"`

### 通话中控制

**前提**: 座席当前有通话中

```bash
# 挂断
clink call hangup --agent 1001

# 保持
clink call hold --agent 1001

# 恢复
clink call unhold --agent 1001

# 转接（转给座席 1002）
clink call transfer --agent 1001 --type 1 --target 1002

# 转接（转给队列 8002）
clink call transfer --agent 1001 --type 2 --target 8002

# 转接（转给外部号码）
clink call transfer --agent 1001 --type 3 --target 13800138002
```

### 高级控制（监听、强插）

```bash
# 监听座席 1001 的通话
clink call spy --agent 1001

# 密语（客户听不到）
clink call whisper --agent 1001

# 强插（三方通话）
clink call barge --agent 1001

# 强拆（强制挂断）
clink call disconnect --agent 1001
```

---

## 🛠️ 配置管理

### 座席配置

```bash
# 查看座席列表
clink config agent list

# 查看座席详情
clink config agent get 1001

# 创建座席
clink config agent create --cno 1009 --name "赵六" --role agent

# 更新座席
clink config agent update 1001 --name "张三（新）"

# 删除座席
clink config agent delete 1001

# 绑定电话
clink config agent bind 1001 --tel 13800138001

# 解绑电话
clink config agent unbind 1001
```

### 队列配置

```bash
# 查看队列详情
clink config queue get 8001

# 创建队列
clink config queue create --qno 8009 --name "新队列"

# 更新队列
clink config queue update 8001 --name "客服一部（新）"

# 删除队列
clink config queue delete 8009

# 查看队列座席
clink config queue agents 8001
```

### IVR 配置

```bash
# 查看 IVR 列表
clink config ivr list

# 查看 IVR 节点
clink config ivr nodes "欢迎流程"
```

---

## 📈 统计报表

### 座席工作量统计

```bash
clink stats agent workload --start 2024-03-01 --end 2024-03-25
```

### 队列统计

```bash
clink stats queue --start 2024-03-01 --end 2024-03-25
```

### 满意度统计

```bash
clink stats satisfaction --start 2024-03-01 --agent 1001
```

### 热线来电分析

```bash
clink stats hotline inbound --start 2024-03-01
```

---

## 📝 日志查询

### 操作日志

```bash
clink logs operation --start 2024-03-20
# 查看系统操作日志
```

### 座席日志

```bash
clink logs agent --agent 1001 --start 2024-03-20
# 特定座席的操作记录
```

### 登录日志

```bash
clink logs login --start 2024-03-20
# 座席登录登出记录
```

---

## 💬 短信发送

```bash
# 发送短信（使用模板）
clink sms send 13800138000 "验证码模板" --params "code=123456"

# 查看短信模板
clink sms templates
```

---

## 🎯 自然语言指令映射

以下是常见用户请求到 CLI 命令的映射：

| 用户请求 | Agent 应该执行 |
|---------|---------------|
| "查一下通话记录" | `clink records inbound` |
| "昨天谁接了电话" | `clink records inbound --start $(date -d yesterday +%Y-%m-%d)` |
| "张三在不在线" | `clink agents --agent <张三的工号>` |
| "让李四上线" | `clink agent online --agent <李四的工号>` |
| "给这个号码回电" | `clink call <phone>` |
| "挂断座席 1001 的电话" | `clink call hangup --agent 1001` |
| "看一下队列情况" | `clink queue status` |
| "导出本月数据" | `clink records inbound --start 2024-03-01 --output json > data.json` |
| "下载录音" | `clink records download <call-id> --output ./` |
| "获取录音链接" | `clink records url <call-id>` |
| "满意度怎么样" | `clink records satisfaction --start 2024-03-01` |
| "今天接了多少电话" | `clink records today --agent 1001` |
| "查一下历史记录" | `clink records history --start 30d` |
| "这通录音的客户侧" | `clink records url <call-id> --side 1` |

---

## 🔧 故障排除

### 认证失败

```
Error: access-id and access-secret are required
```

**解决方案**: 
- 检查环境变量 `CLINK_ACCESS_ID` 和 `CLINK_ACCESS_SECRET`
- 或通过 `--access-id` / `--access-secret` 参数传递

### 网络超时

```
Error: request timeout
```

**解决方案**:
- 检查网络连接
- 切换 API 节点：`--base-url https://api-bj.clink.cn`

### 参数错误

```
Error: invalid parameter
```

**解决方案**:
- 查看命令帮助：`clink <command> --help`
- 检查参数格式（如日期格式 `YYYY-MM-DD`）

### 权限不足

```
Error: forbidden
```

**解决方案**:
- 确认 Access Key 有该接口的调用权限
- 联系管理员开通权限

---

## 📚 相关文档

- [开发文档](DEVELOPMENT.md) - 开发和扩展指南
- [实施计划](IMPLEMENTATION_PLAN.md) - API 分批实施计划
- [GitHub 仓库](https://github.com/raymondtc/clink-cli)

---

**提示**: 本手册随 CLI 版本更新，最新版本请查看 GitHub 仓库。

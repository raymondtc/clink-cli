# Clink CLI 接口实施计划

> 基于 ti-net/clink-sdk (v3.0.16) 的 API 分批实施策略
> 
> 最后更新: 2026-03-25

## 📊 API 统计

| 分类 | 数量 | 说明 |
|------|------|------|
| **通话记录 (CDR)** | 32 | 呼入/外呼记录、录音、满意度 |
| **座席管理** | 18 | 座席CRUD、状态、登录控制 |
| **呼叫控制** | 24 | 外呼、转接、保持、挂断等 |
| **队列管理** | 10 | 队列CRUD、状态、座席绑定 |
| **配置管理** | 45 | IVR、号码、企业设置等 |
| **统计报表** | 20 | 各类统计查询 |
| **在线客服** | 35 | Chat、Session、留言 |
| **工单系统** | 25 | Ticket、表单、流程 |
| **其他** | 15 | 短信、云手机、外呼任务 |
| **总计** | **~224** | 去重后约 180 个独立接口 |

---

## 🎯 分批实施策略

### P1 - 核心功能 (Week 1-2)
**目标**: 覆盖 80% 日常使用场景，Agent 可以完成基本通话管理

| # | API | CLI 命令 | 优先级 | 说明 |
|---|-----|----------|--------|------|
| 1 | ✅ ListCdrIbs | `records inbound` | P0 | 呼入通话记录查询 |
| 2 | ✅ ListCdrObs | `records outbound` | P0 | 外呼通话记录查询 |
| 3 | ✅ AgentStatus | `agents` | P0 | 座席状态查询 |
| 4 | ✅ Webcall | `call <phone>` | P0 | WebCall 外呼 |
| 5 | ✅ Callout | `call agent <phone>` | P0 | 座席外呼 |
| 6 | ✅ Online | `agent online` | P0 | 座席上线 |
| 7 | ✅ Offline | `agent offline` | P0 | 座席下线 |
| 8 | ✅ Pause | `agent pause` | P0 | 座席置忙 |
| 9 | ✅ Unpause | `agent ready` | P0 | 座席置闲 |
| 10 | ✅ QueueStatus | `queue status` | P0 | 队列状态查询 |
| 11 | ✅ ListQueues | `queue list` | P0 | 队列列表查询 |
| 12 | ✅ Unlink | `call hangup` | P0 | 挂断通话 |
| 13 | ✅ Hold | `call hold` | P0 | 保持通话 |
| 14 | ✅ Unhold | `call unhold` | P0 | 恢复通话 |
| 15 | ✅ Transfer | `call transfer` | P0 | 转接电话 |
| 16 | DescribeRecordFileUrl | `records download` | P1 | 录音下载链接 |
| 17 | DownloadRecordFile | `records download --save` | P1 | 直接下载录音 |
| 18 | ListInvestigations | `records satisfaction` | P1 | 满意度记录 |
| 19 | ListTodayCdrsByCno | `records today` | P1 | 当日通话记录 |
| 20 | ListHistoryCdrs | `records history` | P1 | 历史通话记录 |

**P1 完成标准**:
- [ ] 20 个核心 API 全部可用
- [ ] Agent 可以通过自然语言完成：查记录、看状态、打电话、挂电话
- [ ] 基础错误处理和帮助信息完善

---

### P2 - 扩展功能 (Week 3-4)
**目标**: 覆盖配置管理和高级查询，支持管理员日常运维

#### 2.1 座席配置管理
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 21 | ListClients | `config agent list` | 座席列表 |
| 22 | DescribeClient | `config agent get <cno>` | 座席详情 |
| 23 | CreateClient | `config agent create` | 创建座席 |
| 24 | UpdateClient | `config agent update <cno>` | 更新座席 |
| 25 | DeleteClient | `config agent delete <cno>` | 删除座席 |
| 26 | BindClientTel | `config agent bind <cno>` | 绑定电话 |
| 27 | UnbindClientTel | `config agent unbind <cno>` | 解绑电话 |
| 28 | ListClientTels | `config agent tels <cno>` | 查看绑定电话 |

#### 2.2 队列配置管理
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 29 | DescribeQueue | `config queue get <qno>` | 队列详情 |
| 30 | CreateQueue | `config queue create` | 创建队列 |
| 31 | UpdateQueue | `config queue update <qno>` | 更新队列 |
| 32 | DeleteQueue | `config queue delete <qno>` | 删除队列 |
| 33 | ListQueuesWithAgentAction | `config queue agents <qno>` | 队列座席列表 |

#### 2.3 IVR 配置查询
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 34 | ListIvrs | `config ivr list` | IVR 列表 |
| 35 | ListIvrNodes | `config ivr nodes <ivr>` | IVR 节点详情 |

#### 2.4 通话中高级控制
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 36 | Spy | `call spy <agent>` | 监听通话 |
| 37 | Whisper | `call whisper <agent>` | 密语 |
| 38 | Barge | `call barge <agent>` | 强插 |
| 39 | Disconnect | `call disconnect <agent>` | 强拆 |
| 40 | Threeway | `call threeway <agent>` | 三方通话 |
| 41 | Consult | `call consult <agent>` | 咨询 |
| 42 | ConsultTransfer | `call consult-transfer` | 咨询转接 |
| 43 | ConsultThreeway | `call consult-3way` | 咨询三方 |
| 44 | Dtmf | `call dtmf <digits>` | 发送 DTMF |

#### 2.5 日志查询
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 45 | ListOperationLogs | `logs operation` | 操作日志 |
| 46 | ListAgentLogs | `logs agent` | 座席日志 |
| 47 | ListLoginLogs | `logs login` | 登录日志 |

#### 2.6 短信发送
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 48 | SmsSend | `sms send <phone> <template>` | 发送短信 |
| 49 | ListSmsTemplate | `sms templates` | 短信模板列表 |

**P2 完成标准**:
- [ ] 29 个扩展 API 可用
- [ ] 管理员可以配置座席、队列、IVR
- [ ] 支持通话中高级控制（监听、强插等）

---

### P3 - 高级功能 (Week 5-8)
**目标**: 覆盖专业场景，工单、知识库、外呼任务等

#### 3.1 统计报表
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 50 | StatClientWorkload | `stats agent workload` | 座席工作量 |
| 51 | StatQueue | `stats queue` | 队列统计 |
| 52 | StatClientStatus | `stats agent status` | 座席状态统计 |
| 53 | StatHotlineIb | `stats hotline inbound` | 热线来电分析 |
| 54 | StatInvestigationByCno | `stats satisfaction` | 满意度统计 |
| 55 | StatCallIbArea | `stats area` | 来电地区分布 |
| 56 | StatPreviewOb | `stats preview` | 预览外呼统计 |
| 57 | AbstractStat | `stats custom` | 自定义报表 |

#### 3.2 云手机 (隐私号码)
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 58 | CloudNumberAxBind | `cloud bind-ax` | AX 绑定 |
| 59 | CloudNumberAxbBind | `cloud bind-axb` | AXB 绑定 |
| 60 | ListCloudNumberCdrs | `cloud records` | 云手机通话记录 |

#### 3.3 外呼任务
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 61 | CreateTaskProperty | `task create` | 创建外呼任务 |
| 62 | ListAgentTaskProperties | `task list` | 座席任务列表 |
| 63 | ListAgentTaskInventories | `task items` | 任务明细列表 |
| 64 | TaskPropertyExecStatuses | `task status` | 任务执行状态 |

#### 3.4 工单系统 (Ticket)
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 65 | ListTicket | `ticket list` | 工单列表 |
| 66 | GetTicketDetail | `ticket get <id>` | 工单详情 |
| 67 | TicketSave | `ticket create` | 创建工单 |
| 68 | TicketUpdate | `ticket update <id>` | 更新工单 |
| 69 | TicketClose | `ticket close <id>` | 关闭工单 |
| 70 | TicketFinish | `ticket finish <id>` | 完工工单 |
| 71 | TicketFlow | `ticket flow <id>` | 工单流转 |
| 72 | TicketComment | `ticket comment <id>` | 工单备注 |

#### 3.5 知识库 (KB)
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 73 | ListRepositories | `kb repos` | 知识库列表 |
| 74 | ListArticles | `kb articles` | 文章列表 |
| 75 | DescribeArticle | `kb article <id>` | 文章详情 |
| 76 | CreateArticle | `kb article create` | 创建文章 |
| 77 | UpdateArticle | `kb article update <id>` | 更新文章 |
| 78 | ListStandardQuestion | `kb questions` | 标准问题列表 |
| 79 | ListAnswer | `kb answers` | 答案列表 |

#### 3.6 在线客服 (Chat)
| # | API | CLI 命令 | 说明 |
|---|-----|----------|------|
| 80 | ListChatMessage | `chat messages` | 查询聊天记录 |
| 81 | ChatRecord | `chat sessions` | 会话记录 |
| 82 | StatChatClientWorkload | `chat stats` | 客服工作量 |

**P3 完成标准**:
- [ ] 33 个高级 API 可用
- [ ] 支持工单全生命周期管理
- [ ] 支持知识库查询和维护

---

## 📅 实施时间表

```
Week 1:  P1 核心功能开发 (API 1-15)
Week 2:  P1 完善 + 测试 + 文档 (API 16-20)

Week 3:  P2 配置管理 + 高级控制 (API 21-44)
Week 4:  P2 日志 + 短信 + 完善 (API 45-49)

Week 5:  P3 统计报表 (API 50-57)
Week 6:  P3 云手机 + 外呼任务 (API 58-64)
Week 7:  P3 工单系统 (API 65-72)
Week 8:  P3 知识库 + 在线客服 (API 73-82)
```

---

## 🔄 维护流程

### 每周迭代
1. **周一**: 选择本周要实现的 API（从本计划）
2. **周三**: 更新 `config/cli.yaml` 配置
3. **周五**: 运行验证 + 生成代码 + 测试
4. **周末**: 更新 Agent 手册

### SDK 更新时
1. 运行 `make sync-sdk` 拉取最新 SDK
2. 运行 `make extract-openapi` 重新生成 OpenAPI
3. 检查新 API（对比 operationId 列表）
4. 将新 API 加入本计划后续批次
5. 更新 Agent 手册

### 添加新 API 的步骤
```bash
# 1. 编辑配置
vim config/cli.yaml

# 2. 验证配置
make validate-config

# 3. 生成代码
make generate

# 4. 构建测试
make dev-build
./bin/clink <new-command> --help

# 5. 更新文档
vim docs/AGENT_MANUAL.md

# 6. 提交
make validate-config && git add -A && git commit -m "feat: add xxx command"
```

---

## 📈 进度追踪

| 批次 | 计划 | 已完成 | 进度 |
|------|------|--------|------|
| P1 核心功能 | 20 | 15 | 75% ⏳ |
| P2 扩展功能 | 29 | 0 | 0% ⏸️ |
| P3 高级功能 | 33 | 0 | 0% ⏸️ |
| **总计** | **82** | **15** | **18%** |

---

## 📝 注意事项

1. **命名规范**
   - CLI 命令使用小写短横线：`records inbound`
   - 参数使用语义化名称：`--phone` 而非 `--customerNumber`
   - 保持与现有命令风格一致

2. **参数设计**
   - 常用参数给 shorthand：`-s`, `-e`, `-p`, `-a`
   - 时间参数支持相对值：`7d`, `1h`, `30d`
   - 布尔参数使用 `--enable-xxx` / `--disable-xxx`

3. **输出格式**
   - 列表默认 table，支持 `--output json`
   - 敏感信息（手机号）自动脱敏
   - 时间字段自动格式化

4. **错误处理**
   - API 错误转换为友好中文提示
   - 必填参数缺失时给出使用示例
   - 网络错误支持自动重试

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/generated"
	"github.com/spf13/cobra"
)

var (
	// Config flags
	accessID     string
	accessSecret string
	enterpriseID string
	baseURL      string
	debug        bool

	// Filter flags
	startDate string
	endDate   string
	phone     string
	agentID   string
	page      int
	output    string

	// WebCall flags
	ivrName        string
	displayNumber  string
	cdrAssociateID string
	orgID          string
)

// getEnvWithFallback 获取环境变量，支持多个备选名称
func getEnvWithFallback(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

// setupAuthConfig 创建认证配置
func setupAuthConfig() (*client.AuthConfig, string) {
	// 支持多种环境变量名称
	accessKey := getEnvWithFallback("CLINK_ACCESS_ID", "CLINK_ACCESS_KEY_ID")
	secret := getEnvWithFallback("CLINK_ACCESS_SECRET", "CLINK_SECRET")

	if accessID != "" {
		accessKey = accessID
	}
	if accessSecret != "" {
		secret = accessSecret
	}

	// 确定 baseURL
	url := baseURL
	if url == "" {
		url = "https://api-sh.clink.cn"
	}

	return &client.AuthConfig{
		AccessID:     accessKey,
		AccessSecret: secret,
	}, url
}

var rootCmd = &cobra.Command{
	Use:   "clink",
	Short: "天润融通 CLI 工具",
	Long:  "让 AI Agent 可以用自然语言操作天润融通呼叫中心",
}

var recordsCmd = &cobra.Command{
	Use:   "records [inbound|outbound]",
	Short: "查询通话记录",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		direction := args[0]

		authConfig, url := setupAuthConfig()

		// 创建 API 客户端（使用生成代码）
		a, err := api.NewGeneratedAPI(url, authConfig)
		if err != nil {
			return err
		}

		ctx := context.Background()

		var total int
		var err2 error

		if direction == "inbound" {
			records, t, e := a.GetInboundRecords(ctx, startDate, endDate, phone, agentID, page, 50)
			total = t
			err2 = e
			if err2 == nil {
				printInboundRecords(records, total)
			}
		} else {
			records, t, e := a.GetOutboundRecords(ctx, startDate, endDate, phone, agentID, page, 50)
			total = t
			err2 = e
			if err2 == nil {
				printOutboundRecords(records, total)
			}
		}

		if err2 != nil {
			return err2
		}

		return nil
	},
}

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "查询座席状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		authConfig, url := setupAuthConfig()

		a, err := api.NewGeneratedAPI(url, authConfig)
		if err != nil {
			return err
		}

		ctx := context.Background()
		agents, err := a.GetAgentStatus(ctx, agentID)
		if err != nil {
			return err
		}

		fmt.Printf("座席列表 (%d 人):\n\n", len(agents))
		for _, agent := range agents {
			status := "⚪"
			agentStatus := ""
			if agent.AgentStatus != nil {
				agentStatus = *agent.AgentStatus
			}
			switch agentStatus {
			case "空闲":
				status = "🟢"
			case "置忙":
				status = "🔴"
			case "离线":
				status = "⚪"
			}

			name := ""
			if agent.ClientName != nil {
				name = *agent.ClientName
			}
			cno := ""
			if agent.Cno != nil {
				cno = *agent.Cno
			}

			fmt.Printf("%s %s (%s) - %s\n", status, name, cno, agentStatus)
		}

		return nil
	},
}

var callCmd = &cobra.Command{
	Use:   "call [phone]",
	Short: "发起外呼/WebCall",
	Long:  "发起电话呼叫，默认使用 WebCall（无需座席ID）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phoneNum := args[0]

		authConfig, url := setupAuthConfig()

		a, err := api.NewGeneratedAPI(url, authConfig)
		if err != nil {
			return err
		}

		ctx := context.Background()

		var callID, status string

		// 如果指定了座席，使用 callout；否则使用 webcall
		if agentID != "" {
			fmt.Printf("使用座席 %s 发起外呼...\n", agentID)
			result, err := a.MakeCall(ctx, phoneNum, agentID, "")
			if err != nil {
				return err
			}
			if result.CallId != nil {
				callID = *result.CallId
			}
			if result.Status != nil {
				status = *result.Status
			}
		} else {
			fmt.Println("使用 WebCall 发起呼叫（无需座席）...")

			// 构建额外参数
			extraParams := map[string]string{}
			if cdrAssociateID != "" {
				extraParams["cdrAssociateId"] = cdrAssociateID
			}
			if orgID != "" {
				extraParams["orgId"] = orgID
			}

			result, err := a.Webcall(ctx, phoneNum, displayNumber, ivrName, extraParams)
			if err != nil {
				return err
			}
			if result.CallId != nil {
				callID = *result.CallId
			}
			if result.Status != nil {
				status = *result.Status
			}
		}

		fmt.Printf("✓ 已发起呼叫\n")
		fmt.Printf("  通话ID: %s\n", callID)
		fmt.Printf("  状态: %s\n", status)
		fmt.Printf("  号码: %s\n", phoneNum)

		return nil
	},
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "查询队列状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		authConfig, url := setupAuthConfig()

		a, err := api.NewGeneratedAPI(url, authConfig)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// First get queue list
		queues, err := a.ListQueues(ctx, 0, 10)
		if err != nil {
			return err
		}
		if len(queues) == 0 {
			fmt.Println("该企业暂无配置队列")
			return nil
		}

		// Get first queue status
		queueID := ""
		if queues[0].Qno != nil {
			queueID = *queues[0].Qno
		}

		queue, err := a.GetQueueStatus(ctx, queueID)
		if err != nil {
			// Check if it's just no data available
			if err.Error() == "no queue status data available" {
				fmt.Printf("队列列表 (%d 个):\n\n", len(queues))
				for _, q := range queues {
					name := ""
					if q.Name != nil {
						name = *q.Name
					}
					qno := ""
					if q.Qno != nil {
						qno = *q.Qno
					}
					fmt.Printf("  %s: %s\n", qno, name)
				}
				fmt.Printf("\n队列 %s 暂无实时状态数据\n", queueID)
				return nil
			}
			return err
		}

		qname := ""
		if queue.Qname != nil {
			qname = *queue.Qname
		}

		// Also update the queue list iteration to use Name instead of Qname
		for i, q := range queues {
			_ = i
			if q.Qno != nil && *q.Qno == queueID && q.Name != nil {
				qname = *q.Name
			}
		}

		waitCount := 0
		if queue.WaitCount != nil {
			waitCount = *queue.WaitCount
		}
		avgWait := 0
		if queue.QueueUpWaitTime != nil {
			avgWait = *queue.QueueUpWaitTime
		}
		onlineAgents := 0
		if queue.OnlineAgentCount != nil {
			onlineAgents = *queue.OnlineAgentCount
		}
		busyAgents := 0
		if queue.BusyAgentCount != nil {
			busyAgents = *queue.BusyAgentCount
		}

		fmt.Printf("队列: %s (%s)\n\n", qname, queueID)
		fmt.Printf("等待人数: %d\n", waitCount)
		fmt.Printf("平均等待: %d 秒\n", avgWait)
		fmt.Printf("在线座席: %d\n", onlineAgents)
		fmt.Printf("忙碌座席: %d\n", busyAgents)

		return nil
	},
}

// 辅助函数
func strPtr(s string) *string {
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// formatTimestamp 将 Unix 时间戳格式化为可读时间
func formatTimestamp(ts *int64) string {
	if ts == nil || *ts == 0 {
		return "-"
	}
	t := time.Unix(*ts, 0)
	return t.Format("01-02 15:04")
}

// formatTimestampStr 将时间字符串格式化为可读时间
func formatTimestampStr(ts *string) string {
	if ts == nil || *ts == "" {
		return "-"
	}
	// Try to parse as Unix timestamp first
	if unixTs, err := fmt.Sscanf(*ts, "%d", new(int64)); err == nil && unixTs > 0 {
		t := time.Unix(int64(unixTs), 0)
		return t.Format("01-02 15:04")
	}
	// Otherwise return as-is (truncated if too long)
	if len(*ts) > 16 {
		return (*ts)[:16]
	}
	return *ts
}

// formatDuration 格式化通话时长
func formatDuration(d *int) string {
	if d == nil || *d == 0 {
		return "-"
	}
	return fmt.Sprintf("%d秒", *d)
}

// printOutboundRecords 打印外呼记录表格
func printOutboundRecords(records []generated.OutboundCallRecord, total int) {
	fmt.Printf("总计: %d 条记录\n\n", total)
	if len(records) == 0 {
		fmt.Println("暂无记录")
		return
	}

	// 表头
	fmt.Printf("%-20s %-15s %-10s %-10s %-20s %-20s\n", "通话ID", "客户号码", "座席", "类型", "开始时间", "状态")
	fmt.Println("------------------------------------------------------------------------------------------------------------------")

	// 数据行
	for _, r := range records {
		callID := deref(r.CallId)
		if len(callID) > 18 {
			callID = callID[:15] + "..."
		}
		phone := deref(r.CustomerNumber)
		cno := deref(r.Cno)
		if cno == "" {
			cno = "-"
		}
		callType := deref(r.CallType)
		startTime := formatTimestamp(r.StartTime)
		status := deref(r.Status)

		fmt.Printf("%-20s %-15s %-10s %-10s %-20s %-20s\n", callID, phone, cno, callType, startTime, status)
	}
}

// formatInt64Ptr dereferences an int64 pointer safely
func formatInt64Ptr(v *int64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

// printInboundRecords 打印呼入记录表格
func printInboundRecords(records []generated.InboundCallRecord, total int) {
	fmt.Printf("总计: %d 条记录\n\n", total)
	if len(records) == 0 {
		fmt.Println("暂无记录")
		return
	}

	// 表头
	fmt.Printf("%-20s %-15s %-10s %-20s %-20s\n", "通话ID", "客户号码", "座席", "开始时间", "状态")
	fmt.Println("--------------------------------------------------------------------------------------------")

	// 数据行
	for _, r := range records {
		callID := deref(r.CallId)
		if len(callID) > 18 {
			callID = callID[:15] + "..."
		}
		phone := deref(r.CustomerNumber)
		cno := deref(r.Cno)
		if cno == "" {
			cno = "-"
		}
		startTime := formatTimestampStr(r.StartTime)
		status := deref(r.Status)

		fmt.Printf("%-20s %-15s %-10s %-20s %-20s\n", callID, phone, cno, startTime, status)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&accessID, "access-id", "", "Access ID (env: CLINK_ACCESS_ID or CLINK_ACCESS_KEY_ID)")
	rootCmd.PersistentFlags().StringVar(&accessSecret, "access-secret", "", "Access Secret (env: CLINK_ACCESS_SECRET or CLINK_SECRET)")
	rootCmd.PersistentFlags().StringVar(&enterpriseID, "enterprise-id", "", "Enterprise ID (optional, env: CLINK_ENTERPRISE_ID)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Base URL")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")

	// Records flags
	recordsCmd.Flags().StringVarP(&startDate, "start", "s", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "开始日期")
	recordsCmd.Flags().StringVarP(&endDate, "end", "e", time.Now().Format("2006-01-02"), "结束日期")
	recordsCmd.Flags().StringVarP(&phone, "phone", "p", "", "筛选电话号码")
	recordsCmd.Flags().StringVarP(&agentID, "agent", "a", "", "筛选座席ID")
	recordsCmd.Flags().IntVar(&page, "page", 1, "页码")

	// Agents flags
	agentsCmd.Flags().StringVarP(&agentID, "agent", "a", "", "座席ID")

	// Call flags - agent 现在是可选的
	callCmd.Flags().StringVarP(&agentID, "agent", "a", "", "座席ID（可选，如需指定座席外呼）")
	callCmd.Flags().StringVar(&displayNumber, "clid", "", "外显号码（可选）")
	callCmd.Flags().StringVar(&ivrName, "ivr", "工作时间", "IVR名称（可选，默认：工作时间）")
	callCmd.Flags().StringVar(&cdrAssociateID, "cdr-associate-id", "", "通话记录关联ID（可选）")
	callCmd.Flags().StringVar(&orgID, "org-id", "", "组织ID（可选）")

	// Add commands
	rootCmd.AddCommand(recordsCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(queueCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

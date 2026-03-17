package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/generated"
	"github.com/raymondtc/clink-cli/pkg/renderer"
	"github.com/spf13/cobra"
)

var (
	// Global config flags (mapped from environment variables or command line)
	accessID     string
	accessSecret string
	baseURL      string
	outputFormat string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "clink",
	Short: "天润融通 CLI 工具",
	Long:  "天润融通呼叫中心命令行工具 - 查询通话记录、座席状态、发起呼叫等",
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&accessID, "access-id", "",
		"Access ID (env: CLINK_ACCESS_ID or CLINK_ACCESS_KEY_ID)")
	rootCmd.PersistentFlags().StringVar(&accessSecret, "access-secret", "",
		"Access Secret (env: CLINK_ACCESS_SECRET or CLINK_SECRET)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "https://api-sh.clink.cn",
		"API base URL (default: https://api-sh.clink.cn)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format: table, json")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		renderer.PrintError(err)
		os.Exit(1)
	}
}

// createAPI creates API client with configuration from flags/env
func createAPI() (*api.GeneratedAPI, error) {
	// Resolve credentials from flags or environment
	id := resolveAccessID()
	secret := resolveAccessSecret()
	url := resolveBaseURL()

	if id == "" || secret == "" {
		return nil, fmt.Errorf("access-id and access-secret are required (set via flags or environment variables)")
	}

	config := &client.AuthConfig{
		AccessID:     id,
		AccessSecret: secret,
	}

	return api.NewGeneratedAPI(url, config)
}

// resolveAccessID gets access ID from flag or environment
func resolveAccessID() string {
	if accessID != "" {
		return accessID
	}
	if v := os.Getenv("CLINK_ACCESS_ID"); v != "" {
		return v
	}
	return os.Getenv("CLINK_ACCESS_KEY_ID")
}

// resolveAccessSecret gets access secret from flag or environment
func resolveAccessSecret() string {
	if accessSecret != "" {
		return accessSecret
	}
	if v := os.Getenv("CLINK_ACCESS_SECRET"); v != "" {
		return v
	}
	return os.Getenv("CLINK_SECRET")
}

// resolveBaseURL gets base URL from flag or environment
func resolveBaseURL() string {
	if baseURL != "" {
		return baseURL
	}
	if v := os.Getenv("CLINK_BASE_URL"); v != "" {
		return v
	}
	return "https://api-sh.clink.cn"
}

// renderOutput renders data using the configured format
func renderOutput(data interface{}) error {
	r := renderer.New(renderer.Format(outputFormat))
	return r.Render(data)
}

// renderList renders a list result with total count
func renderList(data interface{}, total int) error {
	fmt.Printf("总计: %d 条\n\n", total)
	return renderOutput(data)
}

// ============================================================================
// Records Command - 通话记录查询
// ============================================================================

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "查询通话记录",
	Long:  "查询呼入或呼出通话记录",
}

var recordsInboundCmd = &cobra.Command{
	Use:   "inbound",
	Short: "查询呼入通话记录",
	RunE:  runRecordsInbound,
}

var recordsOutboundCmd = &cobra.Command{
	Use:   "outbound",
	Short: "查询外呼通话记录",
	RunE:  runRecordsOutbound,
}

// records flags - auto-generated from OpenAPI parameters
var (
	recordsStartTime string
	recordsEndTime   string
	recordsPhone     string
	recordsAgent     string
	recordsOffset    int
	recordsLimit     int
)

func init() {
	rootCmd.AddCommand(recordsCmd)
	recordsCmd.AddCommand(recordsInboundCmd)
	recordsCmd.AddCommand(recordsOutboundCmd)

	// Flags mapped from OpenAPI: /cc/list_cdr_ibs parameters
	addTimeRangeFlags(recordsInboundCmd)
	addTimeRangeFlags(recordsOutboundCmd)

	// Optional filter flags
	recordsInboundCmd.Flags().StringVarP(&recordsPhone, "phone", "p", "",
		"客户号码筛选 (OpenAPI: customerNumber)")
	recordsInboundCmd.Flags().StringVarP(&recordsAgent, "agent", "a", "",
		"座席号筛选 (OpenAPI: cno)")
	recordsInboundCmd.Flags().IntVar(&recordsOffset, "offset", 0,
		"偏移量 (OpenAPI: offset, default: 0)")
	recordsInboundCmd.Flags().IntVar(&recordsLimit, "limit", 10,
		"查询条数 (OpenAPI: limit, range: 10-100, default: 10)")

	recordsOutboundCmd.Flags().StringVarP(&recordsPhone, "phone", "p", "",
		"客户号码筛选 (OpenAPI: customerNumber)")
	recordsOutboundCmd.Flags().StringVarP(&recordsAgent, "agent", "a", "",
		"座席号筛选 (OpenAPI: cno)")
	recordsOutboundCmd.Flags().IntVar(&recordsOffset, "offset", 0,
		"偏移量 (OpenAPI: offset, default: 0)")
	recordsOutboundCmd.Flags().IntVar(&recordsLimit, "limit", 10,
		"查询条数 (OpenAPI: limit, range: 10-100, default: 10)")
}

// addTimeRangeFlags adds standard time range flags
func addTimeRangeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&recordsStartTime, "start", "s",
		time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
		"开始时间 (OpenAPI: startTime, format: YYYY-MM-DD)")
	cmd.Flags().StringVarP(&recordsEndTime, "end", "e",
		time.Now().Format("2006-01-02"),
		"结束时间 (OpenAPI: endTime, format: YYYY-MM-DD)")
}

func runRecordsInbound(cmd *cobra.Command, args []string) error {
	a, err := createAPI()
	if err != nil {
		return err
	}

	records, total, err := a.GetInboundRecords(
		context.Background(),
		recordsStartTime,
		recordsEndTime,
		recordsPhone,
		recordsAgent,
		recordsOffset/recordsLimit+1,
		recordsLimit,
	)
	if err != nil {
		return err
	}

	return renderList(records, total)
}

func runRecordsOutbound(cmd *cobra.Command, args []string) error {
	a, err := createAPI()
	if err != nil {
		return err
	}

	records, total, err := a.GetOutboundRecords(
		context.Background(),
		recordsStartTime,
		recordsEndTime,
		recordsPhone,
		recordsAgent,
		recordsOffset/recordsLimit+1,
		recordsLimit,
	)
	if err != nil {
		return err
	}

	return renderList(records, total)
}

// ============================================================================
// Agents Command - 座席状态查询
// ============================================================================

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "查询座席状态",
	RunE:  runAgents,
}

var agentsAgentID string

func init() {
	rootCmd.AddCommand(agentsCmd)
	// Flag mapped from OpenAPI: /cc/agent_status cno parameter
	agentsCmd.Flags().StringVarP(&agentsAgentID, "agent", "a", "",
		"座席号 (OpenAPI: cno, optional - 不传则查询所有座席)")
}

func runAgents(cmd *cobra.Command, args []string) error {
	a, err := createAPI()
	if err != nil {
		return err
	}

	agents, err := a.GetAgentStatus(context.Background(), agentsAgentID)
	if err != nil {
		return err
	}

	// Custom rendering for agents with status icons
	if outputFormat == "table" {
		fmt.Printf("座席列表 (%d 人):\n\n", len(agents))
		table := &renderer.Table{
			Headers: []string{"状态", "姓名", "座席号", "状态"},
		}

		for _, agent := range agents {
			statusIcon := "⚪"
			status := deref(agent.AgentStatus)
			switch status {
			case "空闲":
				statusIcon = "🟢"
			case "置忙":
				statusIcon = "🔴"
			case "离线":
				statusIcon = "⚪"
			}

			table.Rows = append(table.Rows, renderer.Row{
				Cells: []renderer.Cell{
					{Value: statusIcon},
					{Value: deref(agent.ClientName)},
					{Value: deref(agent.Cno)},
					{Value: status},
				},
			})
		}

		r := renderer.New(renderer.FormatTable)
		return r.Render(table)
	}

	return renderOutput(agents)
}

// ============================================================================
// Call Command - 发起呼叫
// ============================================================================

var callCmd = &cobra.Command{
	Use:   "call [phone]",
	Short: "发起外呼/WebCall",
	Long:  "发起电话呼叫，默认使用 WebCall（无需座席ID）。如需指定座席，使用 --agent 参数。",
	Args:  cobra.ExactArgs(1),
	RunE:  runCall,
}

// call flags - auto-mapped from OpenAPI parameters
var (
	callAgentID    string // OpenAPI: cno
	callClid       string // OpenAPI: clid (外显号码)
	callIvrName    string // OpenAPI: ivrName
	callRequestID  string // OpenAPI: requestUniqueId
)

func init() {
	rootCmd.AddCommand(callCmd)

	// Flags from OpenAPI: /cc/callout and /cc/webcall requestBody
	callCmd.Flags().StringVarP(&callAgentID, "agent", "a", "",
		"座席号 (OpenAPI: cno) - 指定后使用座席外呼，否则使用 WebCall")
	callCmd.Flags().StringVar(&callClid, "clid", "",
		"外显号码 (OpenAPI: clid, optional)")
	callCmd.Flags().StringVar(&callIvrName, "ivr", "工作时间",
		"IVR名称 (OpenAPI: ivrName, default: 工作时间)")
	callCmd.Flags().StringVar(&callRequestID, "request-id", "",
		"请求唯一ID (OpenAPI: requestUniqueId, optional - 用于防重放)")
}

func runCall(cmd *cobra.Command, args []string) error {
	phone := args[0]

	a, err := createAPI()
	if err != nil {
		return err
	}

	ctx := context.Background()
	var result *generated.CallResult

	if callAgentID != "" {
		renderer.PrintSuccess(fmt.Sprintf("使用座席 %s 发起外呼...", callAgentID))
		result, err = a.MakeCall(ctx, phone, callAgentID, callClid)
	} else {
		renderer.PrintSuccess("使用 WebCall 发起呼叫（无需座席）...")
		extraParams := map[string]string{}
		if callRequestID != "" {
			extraParams["requestUniqueId"] = callRequestID
		}
		result, err = a.Webcall(ctx, phone, callClid, callIvrName, extraParams)
	}

	if err != nil {
		return err
	}

	// Render result as key-value pairs
	fmt.Println()
	renderer.PrintKV(map[string]string{
		"通话ID": deref(result.CallId),
		"状态":   deref(result.Status),
		"号码":   phone,
	})

	return nil
}

// ============================================================================
// Queue Command - 队列状态查询
// ============================================================================

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "查询队列状态",
	RunE:  runQueue,
}

var queueQnos string

func init() {
	rootCmd.AddCommand(queueCmd)
	// Flag mapped from OpenAPI: /cc/queue_status qnos parameter
	queueCmd.Flags().StringVar(&queueQnos, "qnos", "",
		"队列号列表 (OpenAPI: qnos, optional - 多个用逗号分隔)")
}

func runQueue(cmd *cobra.Command, args []string) error {
	a, err := createAPI()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get queue list first
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
			table := &renderer.Table{
				Headers: []string{"队列号", "名称"},
			}
			for _, q := range queues {
				table.Rows = append(table.Rows, renderer.Row{
					Cells: []renderer.Cell{
						{Value: deref(q.Qno)},
						{Value: deref(q.Name)},
					},
				})
			}
			r := renderer.New(renderer.FormatTable)
			return r.Render(table)
		}
		return err
	}

	qname := deref(queue.Qname)

	// Render queue status
	if outputFormat == "table" {
		fmt.Printf("队列: %s (%s)\n\n", qname, queueID)
		renderer.PrintKV(map[string]string{
			"等待人数": derefInt(queue.WaitCount),
			"平均等待": fmt.Sprintf("%s 秒", derefInt(queue.QueueUpWaitTime)),
			"在线座席": derefInt(queue.OnlineAgentCount),
			"忙碌座席": derefInt(queue.BusyAgentCount),
		})
		return nil
	}

	return renderOutput(queue)
}

// ============================================================================
// Utility Functions
// ============================================================================

// deref safely dereferences a string pointer
func deref(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// derefInt safely dereferences an int pointer
func derefInt(i *int) string {
	if i == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *i)
}

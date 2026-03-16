package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/models"
)

var (
	// Config flags
	accessID     string
	accessSecret string
	enterpriseID string
	baseURL      string
	
	// Filter flags
	startDate string
	endDate   string
	phone     string
	agentID   string
	page      int
	output    string
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
		
		config := client.DefaultConfig()
		
		// 支持多种环境变量名称
		accessKey := getEnvWithFallback("CLINK_ACCESS_ID", "CLINK_ACCESS_KEY_ID")
		secret := getEnvWithFallback("CLINK_ACCESS_SECRET", "CLINK_SECRET")
		
		if accessID != "" {
			accessKey = accessID
		}
		if accessSecret != "" {
			secret = accessSecret
		}
		
		if accessKey != "" && secret != "" {
			config.AccessID = accessKey
			config.AccessSecret = secret
			config.EnterpriseID = enterpriseID
			config.EnableMock = false
		}
		if baseURL != "" {
			config.BaseURL = baseURL
		}
		
		c := client.NewClient(config)
		a := api.NewAPI(c)
		
		ctx := context.Background()
		startTime := startDate + " 00:00:00"
		endTime := endDate + " 23:59:59"
		
		var records interface{}
		var total int
		var err error
		
		if direction == "inbound" {
			records, total, err = a.GetInboundRecords(ctx, startTime, endTime, phone, agentID, page, 50)
		} else {
			records, total, err = a.GetOutboundRecords(ctx, startTime, endTime, phone, agentID, page, 50)
		}
		
		if err != nil {
			return err
		}
		
		fmt.Printf("总计: %d 条记录\n\n", total)
		fmt.Printf("%+v\n", records)
		
		return nil
	},
}

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "查询座席状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := client.DefaultConfig()
		
		accessKey := getEnvWithFallback("CLINK_ACCESS_ID", "CLINK_ACCESS_KEY_ID")
		secret := getEnvWithFallback("CLINK_ACCESS_SECRET", "CLINK_SECRET")
		
		if accessID != "" {
			accessKey = accessID
		}
		if accessSecret != "" {
			secret = accessSecret
		}
		
		if accessKey != "" && secret != "" {
			config.AccessID = accessKey
			config.AccessSecret = secret
			config.EnterpriseID = enterpriseID
			config.EnableMock = false
		}
		
		c := client.NewClient(config)
		a := api.NewAPI(c)
		
		ctx := context.Background()
		agents, err := a.GetAgentStatus(ctx, agentID)
		if err != nil {
			return err
		}
		
		fmt.Printf("座席列表 (%d 人):\n\n", len(agents))
		for _, agent := range agents {
			status := "⚪"
			switch agent.Status {
			case "online":
				status = "🟢"
			case "busy":
				status = "🔴"
			case "offline":
				status = "⚪"
			}
			
			fmt.Printf("%s %s (%s) - %s\n", status, agent.AgentName, agent.AgentID, agent.Status)
			if agent.CurrentCall != "" {
				fmt.Printf("   当前通话: %s\n", agent.CurrentCall)
			}
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
		phone := args[0]
		
		config := client.DefaultConfig()
		
		accessKey := getEnvWithFallback("CLINK_ACCESS_ID", "CLINK_ACCESS_KEY_ID")
		secret := getEnvWithFallback("CLINK_ACCESS_SECRET", "CLINK_SECRET")
		
		if accessID != "" {
			accessKey = accessID
		}
		if accessSecret != "" {
			secret = accessSecret
		}
		
		if accessKey != "" && secret != "" {
			config.AccessID = accessKey
			config.AccessSecret = secret
			config.EnterpriseID = enterpriseID
			config.EnableMock = false
		}
		
		c := client.NewClient(config)
		a := api.NewAPI(c)
		
		ctx := context.Background()
		
		var result *models.CallResult
		var err error
		
		// 如果指定了座席，使用 callout；否则使用 webcall
		if agentID != "" {
			fmt.Printf("使用座席 %s 发起外呼...\n", agentID)
			result, err = a.MakeCall(ctx, phone, agentID, "")
		} else {
			fmt.Println("使用 WebCall 发起呼叫（无需座席）...")
			result, err = a.Webcall(ctx, phone, "")
		}
		
		if err != nil {
			return err
		}
		
		fmt.Printf("✓ 已发起呼叫\n")
		fmt.Printf("  通话ID: %s\n", result.CallID)
		fmt.Printf("  状态: %s\n", result.Status)
		fmt.Printf("  号码: %s\n", result.Phone)
		
		return nil
	},
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "查询队列状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := client.DefaultConfig()
		
		accessKey := getEnvWithFallback("CLINK_ACCESS_ID", "CLINK_ACCESS_KEY_ID")
		secret := getEnvWithFallback("CLINK_ACCESS_SECRET", "CLINK_SECRET")
		
		if accessID != "" {
			accessKey = accessID
		}
		if accessSecret != "" {
			secret = accessSecret
		}
		
		if accessKey != "" && secret != "" {
			config.AccessID = accessKey
			config.AccessSecret = secret
			config.EnterpriseID = enterpriseID
			config.EnableMock = false
		}
		
		c := client.NewClient(config)
		a := api.NewAPI(c)
		
		ctx := context.Background()
		queue, err := a.GetQueueStatus(ctx, "")
		if err != nil {
			return err
		}
		
		fmt.Printf("队列: %s (%s)\n\n", queue.QueueName, queue.QueueID)
		fmt.Printf("等待人数: %d\n", queue.WaitingCount)
		fmt.Printf("平均等待: %d 秒\n", queue.AvgWaitTime)
		fmt.Printf("在线座席: %d\n", queue.AgentsOnline)
		fmt.Printf("忙碌座席: %d\n", queue.AgentsBusy)
		
		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&accessID, "access-id", "", "Access ID (env: CLINK_ACCESS_ID or CLINK_ACCESS_KEY_ID)")
	rootCmd.PersistentFlags().StringVar(&accessSecret, "access-secret", "", "Access Secret (env: CLINK_ACCESS_SECRET or CLINK_SECRET)")
	rootCmd.PersistentFlags().StringVar(&enterpriseID, "enterprise-id", "", "Enterprise ID (optional, env: CLINK_ENTERPRISE_ID)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Base URL")
	
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

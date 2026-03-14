package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
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
		if accessID != "" {
			config.AccessID = accessID
			config.AccessSecret = accessSecret
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
		if accessID != "" {
			config.AccessID = accessID
			config.AccessSecret = accessSecret
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
	Short: "发起外呼",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phone := args[0]
		
		if agentID == "" {
			return fmt.Errorf("请指定座席ID: --agent")
		}
		
		config := client.DefaultConfig()
		if accessID != "" {
			config.AccessID = accessID
			config.AccessSecret = accessSecret
			config.EnterpriseID = enterpriseID
			config.EnableMock = false
		}
		
		c := client.NewClient(config)
		a := api.NewAPI(c)
		
		ctx := context.Background()
		result, err := a.MakeCall(ctx, phone, agentID, "")
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
		if accessID != "" {
			config.AccessID = accessID
			config.AccessSecret = accessSecret
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
	rootCmd.PersistentFlags().StringVar(&accessID, "access-id", os.Getenv("CLINK_ACCESS_ID"), "Access ID")
	rootCmd.PersistentFlags().StringVar(&accessSecret, "access-secret", os.Getenv("CLINK_ACCESS_SECRET"), "Access Secret")
	rootCmd.PersistentFlags().StringVar(&enterpriseID, "enterprise-id", os.Getenv("CLINK_ENTERPRISE_ID"), "Enterprise ID")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Base URL")
	
	// Records flags
	recordsCmd.Flags().StringVarP(&startDate, "start", "s", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "开始日期")
	recordsCmd.Flags().StringVarP(&endDate, "end", "e", time.Now().Format("2006-01-02"), "结束日期")
	recordsCmd.Flags().StringVarP(&phone, "phone", "p", "", "筛选电话号码")
	recordsCmd.Flags().StringVarP(&agentID, "agent", "a", "", "筛选座席ID")
	recordsCmd.Flags().IntVar(&page, "page", 1, "页码")
	
	// Agents flags
	agentsCmd.Flags().StringVarP(&agentID, "agent", "a", "", "座席ID")
	
	// Call flags
	callCmd.Flags().StringVarP(&agentID, "agent", "a", "", "座席ID")
	callCmd.MarkFlagRequired("agent")
	
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

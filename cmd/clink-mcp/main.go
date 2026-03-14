package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	mcp "github.com/mark3labs/mcp-go/mcp"
	server "github.com/mark3labs/mcp-go/server"
	
	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
)

func main() {
	// Create client
	config := client.DefaultConfig()
	
	// Override with env vars
	if id := os.Getenv("CLINK_ACCESS_ID"); id != "" {
		config.AccessID = id
		config.AccessSecret = os.Getenv("CLINK_ACCESS_SECRET")
		config.EnterpriseID = os.Getenv("CLINK_ENTERPRISE_ID")
		config.EnableMock = false
	}
	
	c := client.NewClient(config)
	a := api.NewAPI(c)
	
	// Create MCP server
	s := server.NewMCPServer("clink", "0.1.0")
	
	// Add tools
	s.AddTool(mcp.Tool{
		Name:        "get_inbound_records",
		Description: "获取呼入通话记录，支持时间筛选、电话号码筛选、座席筛选和分页",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"start_time": map[string]string{
					"type":        "string",
					"description": "开始时间，格式：YYYY-MM-DD HH:MM:SS",
				},
				"end_time": map[string]string{
					"type":        "string",
					"description": "结束时间，格式：YYYY-MM-DD HH:MM:SS",
				},
				"phone": map[string]string{
					"type":        "string",
					"description": "电话号码（可选，用于筛选）",
				},
				"agent_id": map[string]string{
					"type":        "string",
					"description": "座席ID（可选，用于筛选）",
				},
				"page": map[string]string{
					"type":        "integer",
					"description": "页码，默认为1",
				},
				"page_size": map[string]string{
					"type":        "integer",
					"description": "每页数量，默认为50",
				},
			},
			Required: []string{"start_time", "end_time"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		startTime, _ := args["start_time"].(string)
		endTime, _ := args["end_time"].(string)
		phone, _ := args["phone"].(string)
		agentID, _ := args["agent_id"].(string)
		page := 1
		if p, ok := args["page"].(float64); ok {
			page = int(p)
		}
		pageSize := 50
		if ps, ok := args["page_size"].(float64); ok {
			pageSize = int(ps)
		}
		
		records, total, err := a.GetInboundRecords(ctx, startTime, endTime, phone, agentID, page, pageSize)
		if err != nil {
			return nil, err
		}
		
		result := map[string]interface{}{
			"total":   total,
			"records": records,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	})
	
	s.AddTool(mcp.Tool{
		Name:        "get_outbound_records",
		Description: "获取外呼通话记录，支持时间筛选、电话号码筛选、座席筛选和分页",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"start_time": map[string]string{
					"type":        "string",
					"description": "开始时间，格式：YYYY-MM-DD HH:MM:SS",
				},
				"end_time": map[string]string{
					"type":        "string",
					"description": "结束时间，格式：YYYY-MM-DD HH:MM:SS",
				},
				"phone": map[string]string{
					"type":        "string",
					"description": "电话号码（可选，用于筛选）",
				},
				"agent_id": map[string]string{
					"type":        "string",
					"description": "座席ID（可选，用于筛选）",
				},
			},
			Required: []string{"start_time", "end_time"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		startTime, _ := args["start_time"].(string)
		endTime, _ := args["end_time"].(string)
		phone, _ := args["phone"].(string)
		agentID, _ := args["agent_id"].(string)
		
		records, total, err := a.GetOutboundRecords(ctx, startTime, endTime, phone, agentID, 1, 50)
		if err != nil {
			return nil, err
		}
		
		result := map[string]interface{}{
			"total":   total,
			"records": records,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	})
	
	s.AddTool(mcp.Tool{
		Name:        "get_agent_status",
		Description: "查询座席状态，可以查询单个座席或所有座席的实时状态",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]string{
					"type":        "string",
					"description": "座席ID（可选，不提供则返回所有座席）",
				},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		agentID, _ := args["agent_id"].(string)
		
		agents, err := a.GetAgentStatus(ctx, agentID)
		if err != nil {
			return nil, err
		}
		
		data, _ := json.MarshalIndent(agents, "", "  ")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	})
	
	s.AddTool(mcp.Tool{
		Name:        "make_call",
		Description: "发起外呼电话，指定电话号码和座席ID",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"phone": map[string]string{
					"type":        "string",
					"description": "要拨打的电话号码",
				},
				"agent_id": map[string]string{
					"type":        "string",
					"description": "执行外呼的座席ID",
				},
				"display_number": map[string]string{
					"type":        "string",
					"description": "外显号码（可选）",
				},
			},
			Required: []string{"phone", "agent_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		phone, _ := args["phone"].(string)
		agentID, _ := args["agent_id"].(string)
		displayNumber, _ := args["display_number"].(string)
		
		result, err := a.MakeCall(ctx, phone, agentID, displayNumber)
		if err != nil {
			return nil, err
		}
		
		data, _ := json.MarshalIndent(result, "", "  ")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	})
	
	s.AddTool(mcp.Tool{
		Name:        "get_queue_status",
		Description: "查询队列状态，包括等待人数、平均等待时间、在线座席数等",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"queue_id": map[string]string{
					"type":        "string",
					"description": "队列ID（可选，不提供则返回所有队列）",
				},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		queueID, _ := args["queue_id"].(string)
		
		queue, err := a.GetQueueStatus(ctx, queueID)
		if err != nil {
			return nil, err
		}
		
		data, _ := json.MarshalIndent(queue, "", "  ")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	})
	
	// Start server
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

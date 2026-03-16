package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/models"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	config := client.DefaultConfig()
	c := client.NewClient(config)
	a := api.NewAPI(c)

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		resp := handleRequest(a, req)
		data, _ := json.Marshal(resp)
		writer.Write(data)
		writer.WriteByte('\n')
		writer.Flush()
	}
}

func handleRequest(a *api.API, req JSONRPCRequest) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "clink-mcp",
				"version": "0.1.0",
			},
			"capabilities": map[string]interface{}{},
		}
	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "get_inbound_records",
					"description": "获取呼入通话记录",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"start_time": map[string]string{"type": "string", "description": "开始时间"},
							"end_time":   map[string]string{"type": "string", "description": "结束时间"},
							"phone":      map[string]string{"type": "string", "description": "电话号码"},
							"agent_id":   map[string]string{"type": "string", "description": "座席ID"},
						},
						"required": []string{"start_time", "end_time"},
					},
				},
				{
					"name":        "get_agent_status",
					"description": "查询座席状态",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"agent_id": map[string]string{"type": "string", "description": "座席ID"},
						},
					},
				},
				{
					"name":        "make_call",
					"description": "发起电话呼叫（默认使用WebCall，无需座席ID）",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"phone":          map[string]string{"type": "string", "description": "要拨打的电话号码"},
							"agent_id":       map[string]string{"type": "string", "description": "座席ID（可选，如需指定座席外呼）"},
							"display_number": map[string]string{"type": "string", "description": "外显号码（可选）"},
						},
						"required": []string{"phone"},
					},
				},
			},
		}
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)
		
		result, err := handleToolCall(a, params.Name, params.Arguments)
		if err != nil {
			resp.Error = &JSONRPCError{Code: -32600, Message: err.Error()}
		} else {
			resp.Result = result
		}
	}

	return resp
}

func handleToolCall(a *api.API, name string, args map[string]interface{}) (interface{}, error) {
	ctx := context.Background()
	
	switch name {
	case "get_inbound_records":
		startTime, _ := args["start_time"].(string)
		endTime, _ := args["end_time"].(string)
		phone, _ := args["phone"].(string)
		agentID, _ := args["agent_id"].(string)
		
		records, total, err := a.GetInboundRecords(ctx, startTime, endTime, phone, agentID, 1, 50)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"total":   total,
			"records": records,
		}, nil
		
	case "get_agent_status":
		agentID, _ := args["agent_id"].(string)
		agents, err := a.GetAgentStatus(ctx, agentID)
		if err != nil {
			return nil, err
		}
		return agents, nil

	case "make_call":
		phone, _ := args["phone"].(string)
		agentID, _ := args["agent_id"].(string)
		displayNumber, _ := args["display_number"].(string)

		var result *models.CallResult
		var err error

		// 如果指定了座席，使用 callout；否则使用 webcall
		if agentID != "" {
			result, err = a.MakeCall(ctx, phone, agentID, displayNumber)
		} else {
			result, err = a.Webcall(ctx, phone, displayNumber)
		}

		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"call_id": result.CallID,
			"status":  result.Status,
			"phone":   result.Phone,
		}, nil
	}
	
	return nil, fmt.Errorf("unknown tool: %s", name)
}

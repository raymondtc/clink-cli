// Package models defines data models for Clink API
package models

// CallRecord represents a call record
type CallRecord struct {
	ID           string `json:"id"`
	Phone        string `json:"phone"`
	AgentID      string `json:"agentId"`
	AgentName    string `json:"agentName"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	Duration     int    `json:"duration"`
	Status       string `json:"status"`
	RecordingURL string `json:"recordingUrl,omitempty"`
}

// Agent represents an agent
type Agent struct {
	AgentID     string `json:"agentId"`
	AgentName   string `json:"agentName"`
	Status      string `json:"status"`
	CurrentCall string `json:"currentCall,omitempty"`
	LoginTime   string `json:"loginTime,omitempty"`
}

// Queue represents a queue
type Queue struct {
	QueueID      string `json:"queueId"`
	QueueName    string `json:"queueName"`
	WaitingCount int    `json:"waitingCount"`
	AvgWaitTime  int    `json:"avgWaitTime"`
	AgentsOnline int    `json:"agentsOnline"`
	AgentsBusy   int    `json:"agentsBusy"`
}

// CallResult represents a call result
type CallResult struct {
	CallID string `json:"callId"`
	Status string `json:"status"`
	Phone  string `json:"phone"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ListResponse represents a list response
type ListResponse struct {
	Total int         `json:"total"`
	List  interface{} `json:"list"`
}

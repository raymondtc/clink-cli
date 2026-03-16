// Package models defines data models for Clink API
package models

// CallRecord represents a call record
type CallRecord struct {
	ID           string `json:"uniqueId,omitempty"`
	Phone        string `json:"customerNumber,omitempty"`
	AgentID      string `json:"cno,omitempty"`
	AgentName    string `json:"clientName,omitempty"`
	StartTime    string `json:"startTime,omitempty"`
	EndTime      string `json:"endTime,omitempty"`
	Duration     int    `json:"callDuration,omitempty"`
	Status       string `json:"status,omitempty"`
	RecordingURL string `json:"recordFile,omitempty"`
}

// Agent represents an agent
type Agent struct {
	AgentID     string `json:"cno,omitempty"`
	AgentName   string `json:"clientName,omitempty"`
	Status      string `json:"agentStatus,omitempty"`
	CurrentCall string `json:"customerNumber,omitempty"`
	LoginTime   string `json:"loginTime,omitempty"`
}

// Queue represents a queue
type Queue struct {
	QueueID      string `json:"qno,omitempty"`
	QueueName    string `json:"qname,omitempty"`
	WaitingCount int    `json:"waitCount,omitempty"`
	AvgWaitTime  int    `json:"queueUpWaitTime,omitempty"`
	AgentsOnline int    `json:"onlineAgentCount,omitempty"`
	AgentsBusy   int    `json:"busyAgentCount,omitempty"`
}

// CallResult represents a call result
type CallResult struct {
	CallID string `json:"callId,omitempty"`
	Status string `json:"status,omitempty"`
	Phone  string `json:"customerNumber,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code        int         `json:"code,omitempty"`
	Message     string      `json:"message,omitempty"`
	Data        interface{} `json:"data,omitempty"`
	Result      interface{} `json:"result,omitempty"` // Webcall等API使用
	RequestID   string      `json:"requestId,omitempty"`
	TotalCount  int         `json:"totalCount,omitempty"`
	AgentStatus []Agent     `json:"agentStatus,omitempty"`
	CdrIbs      []CallRecord `json:"cdrIbs,omitempty"`
	CdrObs      []CallRecord `json:"cdrObs,omitempty"`
}

// ListResponse represents a list response
type ListResponse struct {
	Total int         `json:"totalCount,omitempty"`
	List  interface{} `json:"result,omitempty"`
}

package agent

import (
	"context"
	"fmt"
)

func (am *AgentManager) SeedDefaultAgents(ctx context.Context) error {
	primaryAgent := RegisterAgentRequest{
		ID:          "primary_agent",
		Name:        "Primary Agent",
		Description: "The main AI assistant with comprehensive tool access and reasoning capabilities. Handles complex tasks, code generation, debugging, and system design.",
		Type:        "primary",
		Status:      "active",
		ModelName:   "gpt-4o",
		Capabilities: []AgentCapability{
			{CapabilityType: "tool", Name: "file_read", Description: "Read files from the filesystem"},
			{CapabilityType: "tool", Name: "file_write", Description: "Write or modify files"},
			{CapabilityType: "tool", Name: "file_search", Description: "Search for files using glob patterns"},
			{CapabilityType: "tool", Name: "grep_search", Description: "Search file contents using regex"},
			{CapabilityType: "tool", Name: "bash_execute", Description: "Execute shell commands"},
			{CapabilityType: "tool", Name: "web_fetch", Description: "Fetch content from URLs"},
			{CapabilityType: "tool", Name: "memory_read", Description: "Read from long-term memory"},
			{CapabilityType: "tool", Name: "memory_write", Description: "Write to long-term memory"},
			{CapabilityType: "tool", Name: "trace_create", Description: "Create execution traces"},
			{CapabilityType: "tool", Name: "artifact_create", Description: "Create and manage artifacts"},
			{CapabilityType: "skill", Name: "code_generation", Description: "Generate code in multiple languages"},
			{CapabilityType: "skill", Name: "debugging", Description: "Debug and troubleshoot code issues"},
			{CapabilityType: "skill", Name: "refactoring", Description: "Refactor and improve code quality"},
			{CapabilityType: "skill", Name: "testing", Description: "Write and execute tests"},
			{CapabilityType: "skill", Name: "architecture", Description: "Design system architecture"},
			{CapabilityType: "model", Name: "gpt-4o", Description: "OpenAI GPT-4 Optimized"},
		},
	}

	miniAgent := RegisterAgentRequest{
		ID:          "mini_delegation_agent",
		Name:        "Mini Delegation Agent",
		Description: "A lightweight, cost-effective agent optimized for simple tasks and quick responses. Best for routine operations and straightforward queries.",
		Type:        "utility",
		Status:      "active",
		ModelName:   "gpt-4o-mini",
		Capabilities: []AgentCapability{
			{CapabilityType: "tool", Name: "file_read", Description: "Read files from the filesystem"},
			{CapabilityType: "tool", Name: "file_write", Description: "Write or modify files"},
			{CapabilityType: "tool", Name: "bash_execute", Description: "Execute shell commands"},
			{CapabilityType: "skill", Name: "simple_coding", Description: "Handle simple coding tasks"},
			{CapabilityType: "skill", Name: "quick_responses", Description: "Fast responses to queries"},
			{CapabilityType: "model", Name: "gpt-4o-mini", Description: "OpenAI GPT-4 Mini"},
		},
	}

	agents := []RegisterAgentRequest{primaryAgent, miniAgent}

	for _, agentReq := range agents {
		_, err := am.RegisterAgent(ctx, agentReq)
		if err != nil {
			return fmt.Errorf("failed to seed agent %s: %w", agentReq.ID, err)
		}
	}

	return nil
}

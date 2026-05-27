package service

import (
	"testing"

	"eino-agent/internal/domain"
)

func TestResolveModelConfig(t *testing.T) {
	// svc 的 repo 为 nil，getModelDefaults 会降级到硬编码默认值
	svc := &ChatService{}

	tests := []struct {
		name         string
		agent        *domain.AgentConfig
		input        *ChatInput
		wantProvider string // 返回的SDK提供者
		wantBaseURL  string
		wantAPIKey   string
		wantModel    string
		wantErr      bool
	}{
		{
			name:         "deepseek request override when global fallback is enabled",
			agent:        &domain.AgentConfig{UseGlobalModelConfig: true},
			input:        &ChatInput{VendorKey: "deepseek", APIKey: "sk-deepseek", ModelName: "deepseek-chat"},
			wantProvider: "deepseek",
			wantBaseURL:  "https://api.deepseek.com/v1",
			wantAPIKey:   "sk-deepseek",
			wantModel:    "deepseek-chat",
		},
		{
			name:         "openai unified with other providers",
			agent:        &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini"},
			input:        &ChatInput{},
			wantProvider: "openai",
			wantBaseURL:  "https://api.openai.com/v1",
			wantAPIKey:   "",
			wantModel:    "gpt-4o-mini",
		},
		{
			name:         "openrouter maps to openai sdk",
			agent:        &domain.AgentConfig{UseGlobalModelConfig: true},
			input:        &ChatInput{VendorKey: "openrouter", ModelName: "openai/gpt-4o-mini"},
			wantProvider: "openai",
			wantBaseURL:  "https://openrouter.ai/api/v1",
			wantAPIKey:   "",
			wantModel:    "openai/gpt-4o-mini",
		},
		{
			name:         "modelscope maps to openai sdk",
			agent:        &domain.AgentConfig{UseGlobalModelConfig: true},
			input:        &ChatInput{VendorKey: "modelscope", ModelName: "Qwen/Qwen2.5-7B-Instruct"},
			wantProvider: "openai",
			wantBaseURL:  "https://api-inference.modelscope.cn/v1",
			wantAPIKey:   "",
			wantModel:    "Qwen/Qwen2.5-7B-Instruct",
		},
		{
			name:         "agent-specific provider should override global input",
			agent:        &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini", UseGlobalModelConfig: false},
			input:        &ChatInput{VendorKey: "deepseek", APIKey: "sk-deepseek", ModelName: "deepseek-chat"},
			wantProvider: "openai",
			wantBaseURL:  "https://api.openai.com/v1",
			wantAPIKey:   "",
			wantModel:    "gpt-4o-mini",
		},
		{
			name:    "agent-specific config should not silently fall back to global input when incomplete",
			agent:   &domain.AgentConfig{ModelProvider: "openai", UseGlobalModelConfig: false},
			input:   &ChatInput{VendorKey: "deepseek", APIKey: "sk-deepseek", ModelName: "deepseek-chat"},
			wantErr: true,
		},
		{
			name:    "custom requires base url when global fallback is enabled",
			agent:   &domain.AgentConfig{UseGlobalModelConfig: true},
			input:   &ChatInput{VendorKey: "custom"},
			wantErr: true,
		},
		{
			name:         "ollama maps to local sdk with unified defaults",
			agent:        &domain.AgentConfig{UseGlobalModelConfig: true},
			input:        &ChatInput{VendorKey: "ollama", ModelName: "qwen2.5"},
			wantProvider: "local",
			wantBaseURL:  "http://localhost:11434",
			wantAPIKey:   "",
			wantModel:    "qwen2.5",
		},
		{
			name:         "local provider works same as ollama",
			agent:        &domain.AgentConfig{UseGlobalModelConfig: true},
			input:        &ChatInput{VendorKey: "local", ModelName: "llama3"},
			wantProvider: "local",
			wantBaseURL:  "http://localhost:11434",
			wantAPIKey:   "",
			wantModel:    "llama3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, baseURL, apiKey, modelName, err := svc.resolveModelConfig(tt.agent, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider != tt.wantProvider {
				t.Fatalf("provider = %s, want %s", provider, tt.wantProvider)
			}
			if baseURL != tt.wantBaseURL {
				t.Fatalf("baseURL = %s, want %s", baseURL, tt.wantBaseURL)
			}
			if apiKey != tt.wantAPIKey {
				t.Fatalf("apiKey = %s, want %s", apiKey, tt.wantAPIKey)
			}
			if modelName != tt.wantModel {
				t.Fatalf("modelName = %s, want %s", modelName, tt.wantModel)
			}
		})
	}
}

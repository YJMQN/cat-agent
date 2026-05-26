package service

import (
	"testing"

	"eino-agent/internal/config"
	"eino-agent/internal/domain"
)

func TestResolveModelConfig(t *testing.T) {
	svc := &ChatService{cfg: &config.Config{OpenAIBase: "https://api.openai.com/v1", OpenAIKey: "env-openai-key", LocalModelURL: "http://localhost:11434"}}

	tests := []struct {
		name         string
		agent        *domain.AgentConfig
		input        *ChatInput
		wantProvider string
		wantBaseURL  string
		wantAPIKey   string
		wantModel    string
		wantErr      bool
	}{
		{
			name:         "deepseek request override",
			agent:        &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini"},
			input:        &ChatInput{VendorKey: "deepseek", APIKey: "sk-deepseek", ModelName: "deepseek-chat"},
			wantProvider: "deepseek",
			wantBaseURL:  "https://api.deepseek.com/v1",
			wantAPIKey:   "sk-deepseek",
			wantModel:    "deepseek-chat",
		},
		{
			name:         "openai fallback to env config",
			agent:        &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini"},
			input:        &ChatInput{},
			wantProvider: "openai",
			wantBaseURL:  "https://api.openai.com/v1",
			wantAPIKey:   "env-openai-key",
			wantModel:    "gpt-4o-mini",
		},
		{
			name:    "custom requires base url",
			agent:   &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini"},
			input:   &ChatInput{VendorKey: "custom"},
			wantErr: true,
		},
		{
			name:         "ollama uses local base url",
			agent:        &domain.AgentConfig{ModelProvider: "openai", ModelName: "gpt-4o-mini"},
			input:        &ChatInput{VendorKey: "ollama", ModelName: "qwen2.5"},
			wantProvider: "ollama",
			wantBaseURL:  "http://localhost:11434",
			wantAPIKey:   "",
			wantModel:    "qwen2.5",
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

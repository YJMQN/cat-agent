package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ai-agent-system/internal/agent"
	"ai-agent-system/internal/components/llm"
	"ai-agent-system/internal/components/memory"
	"ai-agent-system/internal/components/tool"
	"ai-agent-system/internal/security"
	"ai-agent-system/pkg/types"
)

func main() {
	log.Println("Starting AI Agent Server...")

	// 创建 LLM 适配器（使用 OpenAI 或兼容 API）
	llmConfig := &llm.ModelConfig{
		Name:        "openai",
		Endpoint:    getEnv("LLM_ENDPOINT", "https://api.openai.com/v1"),
		APIKey:      getEnv("LLM_API_KEY", ""),
		Model:       getEnv("LLM_MODEL", "gpt-3.5-turbo"),
		Temperature: 0.7,
		MaxTokens:   2048,
		TimeoutSecs: 60,
	}

	var llmAdapter llm.Adapter
	var err error

	// 根据配置选择模型后端
	if llmConfig.Endpoint == "https://api.openai.com/v1" {
		llmAdapter, err = llm.NewOpenAIAdapter(llmConfig)
	} else {
		llmAdapter, err = llm.NewPrivateModelAdapter(llmConfig)
	}
	if err != nil {
		log.Fatalf("Failed to create LLM adapter: %v", err)
	}
	log.Printf("LLM Adapter created: %s", llmAdapter.GetModelName())

	// 创建工具注册中心
	toolRegistry := tool.NewRegistry()

	// 注册示例工具
	httpTool := tool.NewHTTPCallerTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Printf("Failed to register http_caller: %v", err)
	} else {
		log.Println("Tool registered: http_caller")
	}

	calcTool := tool.NewCalculatorTool()
	if err := toolRegistry.Register(calcTool); err != nil {
		log.Printf("Failed to register calculator: %v", err)
	} else {
		log.Println("Tool registered: calculator")
	}

	// 创建记忆管理器
	memStore := memory.NewInMemoryStore()
	memManager := memory.NewMemoryManager(memStore, 50, 8000)

	// 创建安全守卫
	securityGuard := security.NewSecurityGuard()

	// 创建 Agent
	agentConfig := &agent.Config{
		LLMAdapter:    llmAdapter,
		ToolRegistry:  toolRegistry,
		MemoryManager: memManager,
		SecurityGuard: securityGuard,
		MaxIterations: 10,
		RetryCount:    3,
	}

	aiAgent, err := agent.NewAgent(agentConfig)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	log.Println("Agent created successfully")

	// 设置 HTTP 处理器
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		chatHandler(w, r, aiAgent)
	})
	http.HandleFunc("/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		streamHandler(w, r, aiAgent)
	})

	// 启动服务器
	port := getEnv("SERVER_PORT", "8080")
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ChatRequest HTTP 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	Stream    bool   `json:"stream,omitempty"`
}

// chatHandler 处理聊天请求（非流式）
func chatHandler(w http.ResponseWriter, r *http.Request, ag *agent.Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}
	if req.UserID == "" {
		req.UserID = "anonymous"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	response, err := ag.Run(ctx, req.SessionID, req.UserID, req.Message)
	if err != nil {
		respondError(w, fmt.Sprintf("Agent error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Session-ID", response.SessionID)
	json.NewEncoder(w).Encode(response)
}

// streamHandler 处理流式聊天请求
func streamHandler(w http.ResponseWriter, r *http.Request, ag *agent.Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}
	if req.UserID == "" {
		req.UserID = "anonymous"
	}

	// 设置 SSE 头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Session-ID", req.SessionID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	streamChan, err := ag.RunStream(ctx, req.SessionID, req.UserID, req.Message)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
		flusher.Flush()
		return
	}

	for chunk := range streamChan {
		data, _ := json.Marshal(map[string]interface{}{
			"content":      chunk.Content,
			"is_finished":  chunk.IsFinished,
			"finish_reason": chunk.FinishReason,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if chunk.IsFinished {
			break
		}
	}
}

// healthHandler 健康检查
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

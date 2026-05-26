// ========== 用户 ==========
export interface User {
  id: number
  username: string
  role: 'admin' | 'operator' | 'user'
  created_at: string
  updated_at: string
}

// ========== Agent ==========
export interface AgentConfig {
  id: number
  name: string
  description: string
  model_provider: string
  model_name: string
  use_global_model_config: boolean
  system_prompt: string
  max_tokens: number
  temperature: number
  tool_ids: string
  status: 'running' | 'stopped'
  created_by: number
  created_at: string
  updated_at: string
}

export interface CreateAgentRequest {
  name: string
  description?: string
  model_provider: string
  model_name: string
  use_global_model_config: boolean
  system_prompt?: string
  max_tokens?: number
  temperature?: number
  tool_names?: string[]
}

// ========== 工具 ==========
export interface Tool {
  id: number
  name: string
  display_name: string
  description: string
  tool_type: 'builtin' | 'http' | 'script'
  params_schema: string
  http_endpoint: string
  http_method: string
  http_headers: string
  script_lang: string
  script_code: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface RegisterToolRequest {
  name: string
  display_name?: string
  description?: string
  tool_type: string
  params_schema?: string
  http_endpoint?: string
  http_method?: string
  http_headers?: string
  script_lang?: string
  script_code?: string
}

// ========== 会话 ==========
export interface Session {
  id: number
  uuid: string
  agent_id: number
  user_id: number
  title: string
  status: 'active' | 'closed'
  token_used: number
  created_at: string
  updated_at: string
}

// ========== 消息 ==========
export interface Message {
  id: number
  session_id: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  tool_calls: string
  tool_call_id: string
  tokens: number
  created_at: string
}

// ========== 记忆 ==========
export interface Memory {
  id: number
  user_id: number
  session_id: number
  category: 'profile' | 'preference' | 'summary' | 'fact'
  key: string
  content: string
  source: 'auto' | 'manual'
  created_at: string
  updated_at: string
}

// ========== 统计 ==========
export interface AdminStats {
  total_sessions: number
  total_messages: number
  total_tokens: number
  total_users: number
  success_rate: number
  avg_latency_ms: number
  active_sessions: number
}

export interface TokenUsage {
  date: string
  total: number
  input?: number
  output?: number
}

// ========== 流式事件 ==========
export interface StreamEvent {
  type: 'text' | 'tool_call' | 'tool_result' | 'tool_confirmation' | 'error' | 'done' | 'session'
  content: string
  tool?: string
  args?: string
  confirmation_id?: string
}

// ========== API响应 ==========
export interface ApiResponse<T> {
  data: T
  message?: string
  error?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  size: number
}

// ========== 登录 ==========
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: number
  user: User
}

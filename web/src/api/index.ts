import request from './request'
import type {
  ApiResponse,
  PaginatedResponse,
  LoginRequest,
  LoginResponse,
  AgentConfig,
  CreateAgentRequest,
  Tool,
  RegisterToolRequest,
  Session,
  Message,
  Memory,
  AdminStats,
  TokenUsage,
  User,
} from '@/types'

// ========== 认证 ==========
export const authApi = {
  login(data: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return request.post('/auth/login', data)
  },
  register(data: { username: string; password: string }): Promise<ApiResponse<User>> {
    return request.post('/auth/register', data)
  },
}

// ========== Agent管理 ==========
export const agentApi = {
  list(): Promise<ApiResponse<AgentConfig[]>> {
    return request.get('/admin/agents')
  },
  get(id: number): Promise<ApiResponse<AgentConfig>> {
    return request.get(`/admin/agents/${id}`)
  },
  create(data: CreateAgentRequest): Promise<ApiResponse<AgentConfig>> {
    return request.post('/admin/agents', data)
  },
  update(id: number, data: CreateAgentRequest): Promise<ApiResponse<AgentConfig>> {
    return request.put(`/admin/agents/${id}`, data)
  },
  delete(id: number): Promise<ApiResponse<null>> {
    return request.delete(`/admin/agents/${id}`)
  },
  start(id: number): Promise<ApiResponse<null>> {
    return request.post(`/admin/agents/${id}/start`)
  },
  stop(id: number): Promise<ApiResponse<null>> {
    return request.post(`/admin/agents/${id}/stop`)
  },
}

// ========== 工具管理 ==========
export const toolApi = {
  list(): Promise<ApiResponse<Tool[]>> {
    return request.get('/admin/tools')
  },
  get(id: number): Promise<ApiResponse<Tool>> {
    return request.get(`/admin/tools/${id}`)
  },
  create(data: RegisterToolRequest): Promise<ApiResponse<Tool>> {
    return request.post('/admin/tools', data)
  },
  update(id: number, data: RegisterToolRequest): Promise<ApiResponse<Tool>> {
    return request.put(`/admin/tools/${id}`, data)
  },
  delete(id: number): Promise<ApiResponse<null>> {
    return request.delete(`/admin/tools/${id}`)
  },
  test(id: number, args: Record<string, unknown>): Promise<ApiResponse<{ result: string }>> {
    return request.post(`/admin/tools/${id}/test`, args)
  },
  enable(id: number): Promise<ApiResponse<null>> {
    return request.post(`/admin/tools/${id}/enable`)
  },
  disable(id: number): Promise<ApiResponse<null>> {
    return request.post(`/admin/tools/${id}/disable`)
  },
}

// ========== 会话管理 ==========
export const sessionApi = {
  list(page = 1, size = 20): Promise<PaginatedResponse<Session>> {
    return request.get(`/admin/sessions?page=${page}&size=${size}`)
  },
  get(id: number): Promise<ApiResponse<Session>> {
    return request.get(`/admin/sessions/${id}`)
  },
  getMessages(id: number, limit = 100): Promise<ApiResponse<Message[]>> {
    return request.get(`/admin/sessions/${id}/messages?limit=${limit}`)
  },
  inject(id: number, content: string): Promise<ApiResponse<null>> {
    return request.post(`/admin/sessions/${id}/inject`, { content })
  },
  reset(id: number): Promise<ApiResponse<null>> {
    return request.post(`/admin/sessions/${id}/reset`)
  },
}

// ========== 统计 ==========
export const statsApi = {
  overview(): Promise<ApiResponse<AdminStats>> {
    return request.get('/admin/stats/overview')
  },
  tokenUsage(days = 7): Promise<ApiResponse<TokenUsage[]>> {
    return request.get(`/admin/stats/tokens?days=${days}`)
  },
  toolRanking(): Promise<ApiResponse<Array<{ tool_name: string; call_count: number }>>> {
    return request.get('/admin/stats/tools')
  },
}

// ========== 记忆管理 ==========
export const memoryApi = {
  list(userId?: number): Promise<ApiResponse<Memory[]>> {
    const query = userId ? `?user_id=${userId}` : ''
    return request.get(`/admin/memories${query}`)
  },
  get(id: number): Promise<ApiResponse<Memory>> {
    return request.get(`/admin/memories/${id}`)
  },
  update(id: number, content: string): Promise<ApiResponse<null>> {
    return request.put(`/admin/memories/${id}`, { content })
  },
  delete(id: number): Promise<ApiResponse<null>> {
    return request.delete(`/admin/memories/${id}`)
  },
}

// ========== 用户管理 ==========
export const userApi = {
  list(): Promise<ApiResponse<User[]>> {
    return request.get('/admin/users')
  },
  updateRole(id: number, role: string): Promise<ApiResponse<null>> {
    return request.put(`/admin/users/${id}/role`, { role })
  },
}

// ========== 聊天 ==========
export const chatApi = {
  send(data: { agent_id: number; session_id?: number; content: string }): Promise<ApiResponse<{ events: unknown[] }>> {
    return request.post('/chat', data)
  },
  confirmTool(data: { confirmation_id: string; approved: boolean; session_id?: number }): Promise<ApiResponse<{ tool: string; content: string }>> {
    return request.post('/chat/tool/confirm', data)
  },
}

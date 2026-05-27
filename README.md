# cat-agent - AI Agent 管理系统

基于 Go (Gin) + Vue 3 + Element Plus 的完整 AI Agent 系统。

## 项目结构

```
cat-agent/
├── cmd/
│   └── main.go                    # 服务入口、路由注册
├── internal/
│   ├── config/
│   │   └── config.go              # 配置加载
│   ├── domain/
│   │   └── models.go              # 数据模型定义
│   ├── repository/
│   │   ├── database.go            # 数据库初始化 & 自动迁移
│   │   └── repo.go                # 仓储层接口 & SQLite实现
│   ├── model/
│   │   ├── interface.go           # 模型抽象接口
│   │   ├── openai_adapter.go      # OpenAI兼容模型适配器
│   │   └── local_adapter.go       # 本地模型适配器 (Ollama)
│   ├── tool/
│   │   └── tools.go               # 工具系统 (注册表 + 内置工具)
│   ├── service/
│   │   ├── services.go            # 服务层聚合
│   │   ├── auth.go                # 认证服务 (JWT)
│   │   ├── chat.go                # 对话服务 (Agent核心循环)
│   │   ├── agent.go               # Agent管理服务
│   │   ├── admin.go               # 管理服务
│   │   ├── enhanced.go            # 增强服务 (记忆/Cron/导出/RAG)
│   │   └── orchestrate.go         # 多Agent协作编排
│   └── api/
│       ├── handler/
│       │   ├── handler.go         # 处理器聚合
│       │   ├── auth.go            # 认证端点
│       │   ├── chat.go            # 对话端点 (SSE)
│       │   ├── admin.go           # 管理端点
│       │   ├── enhanced.go        # 增强API端点
│       │   └── orchestrate.go     # 编排API端点
│       └── middleware/
│           ├── middleware.go       # CORS + 速率限制 + JWT中间件
│           └── jwt.go             # JWT解析器
├── web/                           # Vue 3 前端
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── api/                   # API请求封装
│       ├── types/                 # TypeScript类型
│       ├── router/                # 路由 + 守卫
│       ├── stores/                # Pinia状态
│       ├── layouts/               # 布局组件
│       └── pages/                 # 7个页面
├── go.mod
└── README.md
```

## 快速启动

### 后端

```bash
cd cat-agent

# 编译运行
go mod tidy
go build -o server ./cmd/main.go
./server
# 服务启动于 http://localhost:8080
```

> **AI模型配置无需环境变量**：OpenAI Key、BaseURL、模型名称等均在数据库管理。
> 首次启动后通过 API 配置（见下方「全局模型配置」）。

### 前端

```bash
cd cat-agent/web
npm install
npm run dev
# 前端启动于 http://localhost:3000
```

### 默认账户

- 用户名: `admin`
- 密码: `admin123`

## 核心功能

### Agent 核心
- 自主规划 + 多步工具调用循环 (ReAct)
- 流式输出 (SSE)
- 短期会话上下文 + 长期记忆
- 输入安全清洗 & 防注入
- 工具参数校验 & 执行超时控制
- 多Agent协作编排 (工作流定义与执行)

### 模型适配（全部通过数据库管理）
- **OpenAI兼容**: 支持 GPT-4o、GPT-4o-mini、DeepSeek、OpenRouter、ModelScope 等
- **本地模型**: 兼容 Ollama (qwen2.5、llama3 等)
- **自定义**: 任意 OpenAI 兼容接口
- 所有模型提供者的 BaseURL、API Key、默认模型均通过 `/api/v2/model-config` API 管理

### 内置工具
- `weather` - 天气查询 (wttr.in)
- `calculator` - 数学计算 (递归下降解析器)
- `web_search` - 网络搜索
- `web_fetch` - 网页抓取
- `file` - 文件系统操作 (读写/列表/删除/创建目录)
- `sandbox` - 安全沙箱执行
- `email_send` / `email_read` - 邮件工具

### 增强功能 (第三/四阶段)
- **智能记忆系统**: 分层记忆架构 (工作/短期/长期) + 语义检索 + 关键词抽取
- **Cron调度器**: 定时任务自动执行 + 执行日志追溯
- **RAG文档检索**: 文档分块索引 + 语义搜索 + 上下文构建
- **动态插件系统**: HTTP/Script插件动态注册和运行
- **Token预算监控**: 日/月限额 + 告警阈值提醒
- **对话导出**: Markdown/JSON格式导出
- **审计日志**: 全管理操作可追溯

### 管理系统
- Agent 实例管理 (CRUD、启停)
- 工具注册 & 在线测试
- 对话监控 (实时消息流、注入、重置)
- 数据统计 (Token用量、成功率、工具排行)
- 记忆库管理 (查看、编辑、删除)
- 用户权限 (admin/operator/user)
- JWT认证 + 速率限制 + 路由守卫
- 审计日志查询

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | 8080 | 服务端口 |
| `DATABASE_PATH` | ./data/cat-agent.db | SQLite路径 |
| `DB_ENGINE` | sqlite | 数据库引擎 (sqlite / postgres) |
| `DB_DSN` | (空) | PostgreSQL DSN (DB_ENGINE=postgres时使用) |
| `JWT_SECRET` | (必须设置) | JWT密钥 |
| `JWT_EXPIRE_HOURS` | 72 | JWT过期时间(小时) |
| `APP_ENV` | development | 运行环境 |
| `RATE_LIMIT_REQUESTS` | 100 | 速率限制 (每分钟请求数) |
| `RATE_LIMIT_BURST` | 20 | 速率限制突发值 |
| `AGENT_TIMEOUT` | 5m | Agent编排执行超时 |
| `CRON_ENABLED` | true | 是否启用Cron调度器 |
| `SANDBOX_ENABLED` | false | 沙箱执行 |
| `WS_PORT` | 8081 | WebSocket端口 |
| `CONFIG_FILE` | (空) | YAML/TOML扩展配置路径 |
| `DOCUMENT_DIR` | ./data/documents | 文档存储目录 |
| `EXPORT_DIR` | ./data/exports | 导出文件目录 |
| `TOKEN_ALERT_WEBHOOK` | (空) | Token超限告警Webhook |

> **AI模型配置已全部迁移至数据库管理**，不再使用环境变量。
> 启动后通过 `/api/v2/model-config` 接口配置，见下方说明。

## 全局模型配置

所有AI模型提供者（OpenAI、DeepSeek、OpenRouter、ModelScope、Ollama等）的配置均存储在数据库 `global_model_configs` 表中，提供统一管理API。

### 默认配置（首次启动自动创建）

| 提供者 | BaseURL | 默认模型 |
|--------|---------|---------|
| `openai` | https://api.openai.com/v1 | gpt-4o-mini |
| `deepseek` | https://api.deepseek.com/v1 | deepseek-chat |
| `openrouter` | https://openrouter.ai/api/v1 | openai/gpt-4o-mini |
| `modelscope` | https://api-inference.modelscope.cn/v1 | Qwen/Qwen2.5-7B-Instruct |
| `local` | http://localhost:11434 | qwen2.5 |

### API 接口

所有接口需 `JWT` 认证，挂载于 `/api/v2` 路由下。

```bash
# 列出所有模型配置
GET    /api/v2/model-config

# 新增模型提供者
POST   /api/v2/model-config
# Body: {"provider":"openai","base_url":"...","api_key":"...","default_model":"...","enabled":true}

# 查询单个提供者配置
GET    /api/v2/model-config/:provider

# 更新提供者配置（修改API Key、BaseURL等）
PUT    /api/v2/model-config/:provider
# Body: {"base_url":"...","api_key":"...","default_model":"..."}

# 删除提供者配置
DELETE /api/v2/model-config/:provider
```

### 快速设置 OpenAI API Key

```bash
# 将 YOUR_API_KEY 替换为实际的 Key
curl -X PUT http://localhost:8080/api/v2/model-config/openai \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"api_key":"sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'
```

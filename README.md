# Eino Agent - AI Agent 管理系统

基于 Go (Gin) + Vue 3 + Element Plus 的完整 AI Agent 系统。

## 项目结构

```
eino-agent/
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
│   │   └── tools.go               # 工具系统 (注册表 + 3个内置工具)
│   ├── service/
│   │   ├── services.go            # 服务层聚合
│   │   ├── auth.go                # 认证服务 (JWT)
│   │   ├── chat.go                # 对话服务 (Agent核心循环)
│   │   ├── agent.go               # Agent管理服务
│   │   └── admin.go               # 管理服务
│   └── api/
│       ├── handler/
│       │   ├── handler.go         # 处理器聚合
│       │   ├── auth.go            # 认证端点
│       │   ├── chat.go            # 对话端点 (SSE)
│       │   └── admin.go           # 管理端点
│       └── middleware/
│           ├── middleware.go       # CORS + JWT中间件
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
cd eino-agent

# 设置环境变量 (可选)
export OPENAI_KEY=sk-xxx          # OpenAI API Key
export OPENAI_MODEL=gpt-4o-mini   # 默认模型

# 编译运行
go mod tidy
go build -o server ./cmd/main.go
./server
# 服务启动于 http://localhost:8080
```

### 前端

```bash
cd eino-agent/web
npm install
npm run dev
# 前端启动于 http://localhost:3000
```

### 默认账户

- 用户名: `admin`
- 密码: `admin123`

## 核心功能

### Agent 核心
- 自主规划 + 多步工具调用循环
- 流式输出 (SSE)
- 短期会话上下文 + 长期记忆
- 输入安全清洗 & 防注入
- 工具参数校验 & 执行超时控制

### 模型适配
- **OpenAI兼容**: 支持 GPT-4o、GPT-4o-mini 等
- **本地模型**: 兼容 Ollama (qwen2.5、llama3 等)

### 内置工具
- `weather` - 天气查询 (wttr.in)
- `calculator` - 数学计算 (递归下降解析器)
- `web_search` - 网络搜索

### 管理系统
- Agent 实例管理 (CRUD、启停)
- 工具注册 & 在线测试
- 对话监控 (实时消息流、注入、重置)
- 数据统计 (Token用量、成功率、工具排行)
- 记忆库管理 (查看、编辑、删除)
- 用户权限 (admin/operator/user)
- JWT认证 + 路由守卫

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | 8080 | 服务端口 |
| `DATABASE_PATH` | ./data/eino.db | SQLite路径 |
| `JWT_SECRET` | (内置) | JWT密钥 |
| `OPENAI_BASE` | https://api.openai.com/v1 | OpenAI端点 |
| `OPENAI_KEY` | (空) | API Key |
| `OPENAI_MODEL` | gpt-4o-mini | 默认模型 |
| `LOCAL_MODEL_URL` | http://localhost:11434 | Ollama端点 |
| `LOCAL_MODEL` | qwen2.5 | 本地模型名 |

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { agentApi, chatApi } from '@/api'
import { useModelVendorStore } from '@/stores/vendor'
import type { AgentConfig, StreamEvent } from '@/types'

const agents = ref<AgentConfig[]>([])
const vendorStore = useModelVendorStore()
const selectedAgentId = ref<number>(0)
const sessionId = ref<number>(0)
const inputText = ref('')
const sending = ref(false)
const messages = ref<Array<{ role: string; content: string; tool?: string; timestamp: Date }>>([])
const chatContainer = ref<HTMLElement | null>(null)

onMounted(async () => {
  try {
    const res = await agentApi.list()
    agents.value = res.data || []
    if (agents.value.length > 0) {
      selectedAgentId.value = agents.value[0].id
    }
  } catch { /* handled */ }
})

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

function parseCommandFromArgs(args: string) {
  if (!args) {
    return '未提供命令参数'
  }

  try {
    const parsed = JSON.parse(args)
    if (parsed && typeof parsed.command === 'string') {
      return parsed.command
    }
  } catch {
    // ignore
  }

  return args
}

async function sendMessage() {
  const content = inputText.value.trim()
  if (!content || sending.value) return
  if (!selectedAgentId.value) {
    ElMessage.warning('请先选择一个Agent')
    return
  }

  messages.value.push({ role: 'user', content, timestamp: new Date() })
  inputText.value = ''
  sending.value = true
  scrollToBottom()

  const assistantIdx = messages.value.length
  messages.value.push({ role: 'assistant', content: '', timestamp: new Date() })

  let pendingApproval: { confirmationId: string; tool: string; args: string } | null = null

  try {
    const requestConfig = vendorStore.currentRequestConfig
    const params = new URLSearchParams({
      agent_id: selectedAgentId.value.toString(),
      content,
      vendor_key: requestConfig.vendorKey,
      vendor_model: requestConfig.model,
    })

    if (sessionId.value) {
      params.set('session_id', sessionId.value.toString())
    }
    if (requestConfig.baseURL) {
      params.set('vendor_base_url', requestConfig.baseURL)
    }
    if (requestConfig.apiKey) {
      params.set('vendor_api_key', requestConfig.apiKey)
    }

    const token = localStorage.getItem('token')
    const response = await fetch(`/api/chat/stream?${params}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const reader = response.body?.getReader()
    const decoder = new TextDecoder()

    if (reader) {
      let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          let data = line.slice(6).trim()
          if (data.startsWith('"') && data.endsWith('"')) {
            try { data = JSON.parse(data) } catch { /* keep raw */ }
          }

          try {
            const event: StreamEvent = JSON.parse(data)
            switch (event.type) {
              case 'session':
                sessionId.value = Number(event.content) || 0
                break
              case 'text':
                messages.value[assistantIdx].content += event.content
                scrollToBottom()
                break
              case 'tool_call':
                messages.value.push({
                  role: 'tool',
                  content: `🔧 调用工具: ${event.tool}`,
                  tool: event.tool,
                  timestamp: new Date(),
                })
                scrollToBottom()
                break
              case 'tool_confirmation':
                pendingApproval = {
                  confirmationId: event.confirmation_id || '',
                  tool: event.tool || '',
                  args: event.args || '',
                }
                break
              case 'tool_result':
                messages.value.push({
                  role: 'tool',
                  content: `📋 工具结果: ${event.content}`,
                  tool: event.tool,
                  timestamp: new Date(),
                })
                scrollToBottom()
                break
              case 'error':
                messages.value[assistantIdx].content += `\n❌ ${event.content}`
                break
              case 'done':
                break
            }
          } catch {
            if (data && data !== '[DONE]') {
              messages.value[assistantIdx].content += data
            }
          }
        }
      }
    }

    if (pendingApproval) {
      const command = parseCommandFromArgs(pendingApproval.args)
      let approved = false
      try {
        await ElMessageBox.confirm(
          `即将执行本地命令：\n${command}\n\n是否继续执行？`,
          '需要确认',
          {
            confirmButtonText: '确认执行',
            cancelButtonText: '取消',
            type: 'warning',
          }
        )
        approved = true
      } catch {
        approved = false
      }

      const confirmResp = await chatApi.confirmTool({
        confirmation_id: pendingApproval.confirmationId,
        approved,
        session_id: sessionId.value || undefined,
      })

      messages.value.push({
        role: 'tool',
        content: `📋 工具结果: ${confirmResp.data.content}`,
        tool: pendingApproval.tool,
        timestamp: new Date(),
      })
      scrollToBottom()
    }
  } catch (e: any) {
    messages.value[assistantIdx].content = `❌ 请求失败: ${e.message}`
  } finally {
    sending.value = false
    scrollToBottom()
  }
}

function clearChat() {
  messages.value = []
  sessionId.value = 0
}

function getRoleClass(role: string) {
  return role
}

function formatTime(d: Date) {
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="chat-page">
    <el-card shadow="never" class="chat-card">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <div style="display: flex; align-items: center; gap: 16px; flex-wrap: wrap">
            <span>对话测试</span>
            <el-select v-model="selectedAgentId" placeholder="选择Agent" size="small" style="width: 200px">
              <el-option v-for="a in agents" :key="a.id" :label="a.name" :value="a.id" />
            </el-select>
            <el-tag type="success">当前厂商：{{ vendorStore.currentLabel }}</el-tag>
            <el-tag type="info">当前模型：{{ vendorStore.currentModel }}</el-tag>
          </div>
          <el-button size="small" @click="clearChat"><el-icon><Delete /></el-icon>清空</el-button>
        </div>
      </template>

      <div ref="chatContainer" class="chat-messages">
        <div v-if="messages.length === 0" class="chat-empty">
          <el-icon :size="48" color="#dcdfe6"><ChatDotRound /></el-icon>
          <p>选择一个Agent开始对话</p>
        </div>

        <div v-for="(msg, idx) in messages" :key="idx" class="chat-msg" :class="getRoleClass(msg.role)">
          <div class="msg-avatar">
            <el-icon v-if="msg.role === 'user'" :size="20"><User /></el-icon>
            <el-icon v-else-if="msg.role === 'assistant'" :size="20"><Cpu /></el-icon>
            <el-icon v-else :size="20"><SetUp /></el-icon>
          </div>
          <div class="msg-body">
            <div class="msg-meta">
              <span class="msg-role">{{ msg.role === 'user' ? '我' : msg.role === 'assistant' ? 'Agent' : '系统' }}</span>
              <span class="msg-time">{{ formatTime(msg.timestamp) }}</span>
            </div>
            <div class="msg-text">{{ msg.content || (sending && idx === messages.length - 1 ? '思考中...' : '') }}</div>
          </div>
        </div>
      </div>

      <div class="chat-input">
        <el-input
          v-model="inputText"
          type="textarea"
          :rows="2"
          placeholder="输入消息... (Enter发送，Shift+Enter换行)"
          :disabled="sending"
          @keydown.enter.exact.prevent="sendMessage"
        />
        <el-button type="primary" :loading="sending" :disabled="!inputText.trim()" @click="sendMessage" style="margin-top: 8px">
          发送
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.chat-page {
  height: calc(100vh - 140px);
}

.chat-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.chat-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #c0c4cc;
}

.chat-empty p {
  margin-top: 12px;
  font-size: 14px;
}

.chat-msg {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.chat-msg.user {
  flex-direction: row-reverse;
}

.msg-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #606266;
}

.chat-msg.user .msg-avatar {
  background: #6ea96a;
  color: #fff;
}

.chat-msg.assistant .msg-avatar {
  background: #8bcf7d;
  color: #fff;
}

.msg-body {
  max-width: 70%;
}

.msg-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
  font-size: 12px;
  color: #909399;
}

.chat-msg.user .msg-meta {
  flex-direction: row-reverse;
}

.msg-text {
  background: #f4f4f5;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.chat-msg.user .msg-text {
  background: #eef8ee;
  color: #1f3224;
}

.chat-msg.assistant .msg-text {
  background: #f2faf1;
}

.chat-msg.tool .msg-text {
  background: #fdf6ec;
  font-size: 13px;
  color: #909399;
}

.chat-input {
  padding: 16px;
  border-top: 1px solid #ebeef5;
  background: #fff;
}
</style>

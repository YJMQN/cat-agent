<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { sessionApi } from '@/api'
import type { Session, Message } from '@/types'

const sessions = ref<Session[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(true)
const selectedSession = ref<Session | null>(null)
const messages = ref<Message[]>([])
const msgLoading = ref(false)
const injectDialogVisible = ref(false)
const injectContent = ref('')

onMounted(loadData)

async function loadData() {
  loading.value = true
  try {
    const res = await sessionApi.list(page.value, size.value)
    sessions.value = res.data || []
    total.value = res.total || 0
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

async function selectSession(session: Session) {
  selectedSession.value = session
  msgLoading.value = true
  try {
    const res = await sessionApi.getMessages(session.id, 200)
    messages.value = res.data || []
  } catch { /* handled */ } finally {
    msgLoading.value = false
  }
}

async function handleInject() {
  if (!selectedSession.value || !injectContent.value.trim()) return
  try {
    await sessionApi.inject(selectedSession.value.id, injectContent.value)
    ElMessage.success('注入成功')
    injectDialogVisible.value = false
    injectContent.value = ''
    await selectSession(selectedSession.value)
  } catch { /* handled */ }
}

async function handleReset() {
  if (!selectedSession.value) return
  try {
    await sessionApi.reset(selectedSession.value.id)
    ElMessage.success('会话已重置')
    await loadData()
  } catch { /* handled */ }
}

function handlePageChange(p: number) {
  page.value = p
  loadData()
}

function getRoleTag(role: string) {
  const map: Record<string, string> = { user: '', assistant: 'success', system: 'warning', tool: 'info' }
  return map[role] || 'info'
}

function getRoleLabel(role: string) {
  const map: Record<string, string> = { user: '用户', assistant: '助手', system: '系统', tool: '工具' }
  return map[role] || role
}

function formatTime(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}
</script>

<template>
  <el-row :gutter="16" class="sessions-page">
    <!-- 左侧会话列表 -->
    <el-col :span="8">
      <el-card shadow="never" class="session-list-card">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center">
            <span>会话列表 ({{ total }})</span>
            <el-button size="small" @click="loadData"><el-icon><Refresh /></el-icon></el-button>
          </div>
        </template>

        <div v-loading="loading" class="session-list">
          <div
            v-for="s in sessions" :key="s.id"
            class="session-item"
            :class="{ active: selectedSession?.id === s.id }"
            @click="selectSession(s)"
          >
            <div class="session-title">{{ s.title || '新会话' }}</div>
            <div class="session-meta">
              <el-tag :type="s.status === 'active' ? 'success' : 'info'" size="small">
                {{ s.status === 'active' ? '活跃' : '已关闭' }}
              </el-tag>
              <span class="session-tokens">{{ s.token_used }} tokens</span>
            </div>
            <div class="session-time">{{ formatTime(s.updated_at) }}</div>
          </div>
          <el-empty v-if="!loading && sessions.length === 0" description="暂无会话" :image-size="40" />
        </div>

        <el-pagination
          v-if="total > size"
          small
          layout="prev, pager, next"
          :total="total"
          :page-size="size"
          :current-page="page"
          @current-change="handlePageChange"
          style="margin-top: 12px; justify-content: center"
        />
      </el-card>
    </el-col>

    <!-- 右侧消息流 -->
    <el-col :span="16">
      <el-card shadow="never" class="message-card">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center" v-if="selectedSession">
            <span>{{ selectedSession.title || '会话详情' }}</span>
            <div>
              <el-button size="small" @click="injectDialogVisible = true">注入消息</el-button>
              <el-button size="small" type="warning" @click="handleReset">重置会话</el-button>
            </div>
          </div>
          <span v-else>选择一个会话查看消息</span>
        </template>

        <div v-if="selectedSession" v-loading="msgLoading" class="message-list">
          <div v-for="msg in messages" :key="msg.id" class="message-item" :class="msg.role">
            <div class="msg-header">
              <el-tag :type="getRoleTag(msg.role)" size="small">{{ getRoleLabel(msg.role) }}</el-tag>
              <span class="msg-time">{{ formatTime(msg.created_at) }}</span>
              <span v-if="msg.tokens" class="msg-tokens">{{ msg.tokens }} tokens</span>
            </div>
            <div class="msg-content">{{ msg.content }}</div>
            <div v-if="msg.tool_calls" class="msg-tool-calls">
              <pre>{{ msg.tool_calls }}</pre>
            </div>
          </div>
          <el-empty v-if="messages.length === 0" description="暂无消息" :image-size="40" />
        </div>

        <el-empty v-else description="← 请从左侧选择会话" :image-size="80" />
      </el-card>
    </el-col>
  </el-row>

  <!-- 注入消息对话框 -->
  <el-dialog v-model="injectDialogVisible" title="注入系统消息" width="480px">
    <el-input v-model="injectContent" type="textarea" :rows="4" placeholder="输入要注入的系统消息内容" />
    <template #footer>
      <el-button @click="injectDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleInject">注入</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.sessions-page {
  height: calc(100vh - 140px);
}

.session-list-card,
.message-card {
  height: 100%;
}

.session-list-card :deep(.el-card__body),
.message-card :deep(.el-card__body) {
  height: calc(100% - 60px);
  overflow-y: auto;
  padding: 12px;
}

.session-list {
  min-height: 200px;
}

.session-item {
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 8px;
  border: 1px solid #ebeef5;
  transition: all 0.2s;
}

.session-item:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.session-item.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.session-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.session-tokens {
  font-size: 12px;
  color: #909399;
}

.session-time {
  font-size: 12px;
  color: #c0c4cc;
}

.message-list {
  min-height: 200px;
}

.message-item {
  margin-bottom: 16px;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.message-item.user {
  border-left: 3px solid #409eff;
}

.message-item.assistant {
  border-left: 3px solid #67c23a;
}

.message-item.system {
  border-left: 3px solid #e6a23c;
  background: #fdf6ec;
}

.message-item.tool {
  border-left: 3px solid #909399;
  background: #f4f4f5;
}

.msg-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.msg-time {
  font-size: 12px;
  color: #c0c4cc;
}

.msg-tokens {
  font-size: 12px;
  color: #909399;
  margin-left: auto;
}

.msg-content {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.msg-tool-calls {
  margin-top: 8px;
  background: #f5f7fa;
  padding: 8px;
  border-radius: 4px;
}

.msg-tool-calls pre {
  margin: 0;
  font-size: 12px;
  color: #606266;
  white-space: pre-wrap;
}
</style>

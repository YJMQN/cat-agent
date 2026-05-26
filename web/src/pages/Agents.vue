<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { agentApi, toolApi } from '@/api'
import type { AgentConfig, CreateAgentRequest, Tool } from '@/types'
import type { FormInstance, FormRules } from 'element-plus'

const agents = ref<AgentConfig[]>([])
const tools = ref<Tool[]>([])
const loading = ref(true)
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const defaultAgentProvider = 'openrouter'
const defaultAgentModel = 'openai/gpt-4o-mini'

const form = ref<CreateAgentRequest>({
  name: '',
  description: '',
  model_provider: defaultAgentProvider,
  model_name: defaultAgentModel,
  use_global_model_config: true,
  system_prompt: '你是一个有用的AI助手。',
  max_tokens: 4096,
  temperature: 0.7,
  tool_names: [],
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入Agent名称', trigger: 'blur' }],
}

const modelPresets = {
  openrouter: ['openai/gpt-4o-mini', 'openai/gpt-4o', 'anthropic/claude-3.5-sonnet', 'deepseek/deepseek-chat'],
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo'],
  modelscope: ['Qwen/Qwen2.5-7B-Instruct', 'Qwen/Qwen2.5-3B-Instruct', 'Qwen/Qwen2.5-1.5B-Instruct'],
  local: ['qwen2.5', 'llama3', 'deepseek-r1', 'glm4'],
}

onMounted(async () => {
  await loadData()
})

async function loadData() {
  loading.value = true
  try {
    const [agentsRes, toolsRes] = await Promise.all([agentApi.list(), toolApi.list()])
    agents.value = agentsRes.data || []
    tools.value = toolsRes.data || []
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = {
    name: '',
    description: '',
    model_provider: defaultAgentProvider,
    model_name: defaultAgentModel,
    use_global_model_config: true,
    system_prompt: '你是一个有用的AI助手。',
    max_tokens: 4096,
    temperature: 0.7,
    tool_names: [],
  }
  dialogVisible.value = true
}

function openEdit(agent: AgentConfig) {
  editingId.value = agent.id
  let toolNames: string[] = []
  try { toolNames = JSON.parse(agent.tool_ids || '[]') } catch { /* ignore */ }
  form.value = {
    name: agent.name,
    description: agent.description,
    model_provider: agent.model_provider || defaultAgentProvider,
    model_name: agent.model_name || defaultAgentModel,
    use_global_model_config: agent.use_global_model_config,
    system_prompt: agent.system_prompt,
    max_tokens: agent.max_tokens,
    temperature: agent.temperature,
    tool_names: toolNames,
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  if (form.value.use_global_model_config) {
    form.value.model_provider = defaultAgentProvider
    form.value.model_name = defaultAgentModel
  }

  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  try {
    if (editingId.value) {
      await agentApi.update(editingId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await agentApi.create(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadData()
  } catch { /* handled */ }
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('确定删除该Agent？', '提示', { type: 'warning' })
  try {
    await agentApi.delete(id)
    ElMessage.success('删除成功')
    await loadData()
  } catch { /* handled */ }
}

async function handleToggle(agent: AgentConfig) {
  try {
    if (agent.status === 'running') {
      await agentApi.stop(agent.id)
      ElMessage.success('已停止')
    } else {
      await agentApi.start(agent.id)
      ElMessage.success('已启动')
    }
    await loadData()
  } catch { /* handled */ }
}

function parseToolNames(json: string): string[] {
  try { return JSON.parse(json || '[]') } catch { return [] }
}
</script>

<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>Agent列表</span>
          <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新建Agent</el-button>
        </div>
      </template>

      <el-table :data="agents" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="模型来源" width="120">
          <template #default="{ row }">
            <el-tag :type="row.use_global_model_config ? 'success' : 'warning'" size="small">
              {{ row.use_global_model_config ? '继承全局' : '独立配置' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="模型提供者" width="130">
          <template #default="{ row }">
            <span>{{ row.use_global_model_config ? '继承全局' : row.model_provider }}</span>
          </template>
        </el-table-column>
        <el-table-column label="模型" width="170">
          <template #default="{ row }">
            <span>{{ row.use_global_model_config ? '继承全局' : row.model_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="绑定工具" min-width="160">
          <template #default="{ row }">
            <el-tag v-for="t in parseToolNames(row.tool_ids)" :key="t" size="small" style="margin: 2px">{{ t }}</el-tag>
            <span v-if="parseToolNames(row.tool_ids).length === 0" style="color: #909399">全部内置工具</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'running' ? 'success' : 'info'" size="small">
              {{ row.status === 'running' ? '运行中' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :type="row.status === 'running' ? 'warning' : 'success'" @click="handleToggle(row)">
              {{ row.status === 'running' ? '停止' : '启动' }}
            </el-button>
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑Agent' : '新建Agent'" width="600px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="Agent名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="功能描述" />
        </el-form-item>
        <el-form-item label="模型配置">
          <el-switch v-model="form.use_global_model_config" active-text="继承全局配置" inactive-text="使用独立配置" />
          <div style="color: #909399; font-size: 12px; margin-top: 6px;">
            {{ form.use_global_model_config ? '当前将使用全局厂商、API Key 和模型设置。' : '当前将使用下面的独立模型配置。' }}
          </div>
        </el-form-item>
        <template v-if="!form.use_global_model_config">
          <el-form-item label="模型提供者">
            <el-select v-model="form.model_provider" style="width: 100%">
              <el-option label="OpenRouter" value="openrouter" />
              <el-option label="OpenAI兼容" value="openai" />
              <el-option label="ModelScope" value="modelscope" />
              <el-option label="本地模型(Ollama)" value="local" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型名称">
            <el-select v-model="form.model_name" filterable allow-create style="width: 100%" :placeholder="'选择或输入模型'">
              <el-option v-for="m in modelPresets[form.model_provider as keyof typeof modelPresets] || []" :key="m" :label="m" :value="m" />
            </el-select>
          </el-form-item>
        </template>
        <el-form-item label="系统提示词">
          <el-input v-model="form.system_prompt" type="textarea" :rows="4" placeholder="定义Agent的角色和行为" />
        </el-form-item>
        <el-form-item label="绑定工具">
          <el-select v-model="form.tool_names" multiple style="width: 100%" placeholder="留空则使用全部内置工具">
            <el-option v-for="t in tools" :key="t.name" :label="t.display_name || t.name" :value="t.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="form.max_tokens" :min="256" :max="128000" :step="256" />
        </el-form-item>
        <el-form-item label="Temperature">
          <el-slider v-model="form.temperature" :min="0" :max="2" :step="0.1" show-input :format-tooltip="(v: number) => v.toFixed(1)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">{{ editingId ? '更新' : '创建' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

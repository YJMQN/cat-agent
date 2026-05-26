<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { toolApi } from '@/api'
import type { Tool, RegisterToolRequest } from '@/types'
import type { FormInstance, FormRules } from 'element-plus'

const tools = ref<Tool[]>([])
const loading = ref(true)
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const testArgs = ref('{}')
const testResult = ref('')
const testingTool = ref<Tool | null>(null)

const form = ref<RegisterToolRequest>({
  name: '',
  display_name: '',
  description: '',
  tool_type: 'builtin',
  params_schema: '{}',
  http_endpoint: '',
  http_method: 'GET',
  http_headers: '{}',
  script_lang: 'javascript',
  script_code: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入工具名称', trigger: 'blur' }],
  tool_type: [{ required: true, message: '请选择工具类型', trigger: 'change' }],
}

onMounted(loadData)

async function loadData() {
  loading.value = true
  try {
    const res = await toolApi.list()
    tools.value = res.data || []
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = { name: '', display_name: '', description: '', tool_type: 'builtin', params_schema: '{}', http_endpoint: '', http_method: 'GET', http_headers: '{}', script_lang: 'javascript', script_code: '' }
  dialogVisible.value = true
}

function openEdit(tool: Tool) {
  editingId.value = tool.id
  form.value = { ...tool }
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  try {
    if (editingId.value) {
      await toolApi.update(editingId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await toolApi.create(form.value)
      ElMessage.success('注册成功')
    }
    dialogVisible.value = false
    await loadData()
  } catch { /* handled */ }
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('确定删除该工具？', '提示', { type: 'warning' })
  try {
    await toolApi.delete(id)
    ElMessage.success('删除成功')
    await loadData()
  } catch { /* handled */ }
}

async function handleToggle(tool: Tool) {
  try {
    if (tool.enabled) {
      await toolApi.disable(tool.id)
      ElMessage.success('已禁用')
    } else {
      await toolApi.enable(tool.id)
      ElMessage.success('已启用')
    }
    await loadData()
  } catch { /* handled */ }
}

function openTest(tool: Tool) {
  testingTool.value = tool
  testArgs.value = '{}'
  testResult.value = ''
  testDialogVisible.value = true
}

async function runTest() {
  if (!testingTool.value) return
  try {
    const args = JSON.parse(testArgs.value)
    const res = await toolApi.test(testingTool.value.id, args)
    testResult.value = res.data.result
  } catch (e: any) {
    testResult.value = '错误: ' + (e.message || '参数格式错误')
  }
}
</script>

<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>工具列表</span>
          <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>注册工具</el-button>
        </div>
      </template>

      <el-row :gutter="16">
        <el-col :span="8" v-for="tool in tools" :key="tool.id" style="margin-bottom: 16px">
          <el-card shadow="hover" class="tool-card">
            <div class="tool-header">
              <div>
                <h3>{{ tool.display_name || tool.name }}</h3>
                <el-tag :type="tool.tool_type === 'builtin' ? 'success' : tool.tool_type === 'http' ? 'warning' : 'info'" size="small">
                  {{ tool.tool_type }}
                </el-tag>
              </div>
              <el-switch v-model="tool.enabled" @change="handleToggle(tool)" :active-value="true" :inactive-value="false" />
            </div>
            <p class="tool-desc">{{ tool.description || '暂无描述' }}</p>
            <div class="tool-actions">
              <el-button size="small" @click="openTest(tool)" :disabled="!tool.enabled">测试</el-button>
              <el-button size="small" @click="openEdit(tool)">编辑</el-button>
              <el-button size="small" type="danger" @click="handleDelete(tool.id)">删除</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-empty v-if="!loading && tools.length === 0" description="暂无工具" />
    </el-card>

    <!-- 注册/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑工具' : '注册工具'" width="600px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="tool_name (英文标识)" :disabled="!!editingId" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="form.display_name" placeholder="中文显示名" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="类型" prop="tool_type">
          <el-select v-model="form.tool_type" style="width: 100%">
            <el-option label="内置工具" value="builtin" />
            <el-option label="HTTP接口" value="http" />
            <el-option label="脚本" value="script" />
          </el-select>
        </el-form-item>
        <el-form-item label="参数Schema">
          <el-input v-model="form.params_schema" type="textarea" :rows="4" placeholder='JSON Schema格式' />
        </el-form-item>
        <template v-if="form.tool_type === 'http'">
          <el-form-item label="端点">
            <el-input v-model="form.http_endpoint" placeholder="https://api.example.com/endpoint" />
          </el-form-item>
          <el-form-item label="HTTP方法">
            <el-select v-model="form.http_method">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
            </el-select>
          </el-form-item>
        </template>
        <template v-if="form.tool_type === 'script'">
          <el-form-item label="脚本语言">
            <el-select v-model="form.script_lang">
              <el-option label="JavaScript" value="javascript" />
              <el-option label="Python" value="python" />
            </el-select>
          </el-form-item>
          <el-form-item label="脚本代码">
            <el-input v-model="form.script_code" type="textarea" :rows="6" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">{{ editingId ? '更新' : '注册' }}</el-button>
      </template>
    </el-dialog>

    <!-- 测试对话框 -->
    <el-dialog v-model="testDialogVisible" :title="`测试工具: ${testingTool?.name}`" width="500px">
      <el-form label-width="80px">
        <el-form-item label="参数(JSON)">
          <el-input v-model="testArgs" type="textarea" :rows="4" placeholder='{"key": "value"}' />
        </el-form-item>
      </el-form>
      <el-button type="primary" @click="runTest" style="margin-bottom: 16px">执行测试</el-button>
      <el-card v-if="testResult" shadow="never" style="background: #f5f5f5">
        <pre style="white-space: pre-wrap; margin: 0; font-size: 13px">{{ testResult }}</pre>
      </el-card>
      <template #footer>
        <el-button @click="testDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tool-card :deep(.el-card__body) {
  padding: 16px;
}
.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.tool-header h3 {
  margin: 0 0 8px;
  font-size: 16px;
  color: #303133;
}
.tool-desc {
  color: #909399;
  font-size: 13px;
  margin: 12px 0;
  min-height: 36px;
}
.tool-actions {
  display: flex;
  gap: 8px;
}
</style>

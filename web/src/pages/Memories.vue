<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { memoryApi } from '@/api'
import type { Memory } from '@/types'

const memories = ref<Memory[]>([])
const loading = ref(true)
const editDialogVisible = ref(false)
const editingMemory = ref<Memory | null>(null)
const editContent = ref('')

const categoryLabels: Record<string, string> = {
  profile: '用户画像',
  preference: '偏好',
  summary: '会话摘要',
  fact: '事实',
}

const categoryColors: Record<string, string> = {
  profile: '',
  preference: 'success',
  summary: 'warning',
  fact: 'info',
}

onMounted(loadData)

async function loadData() {
  loading.value = true
  try {
    const res = await memoryApi.list()
    memories.value = res.data || []
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

function openEdit(mem: Memory) {
  editingMemory.value = mem
  editContent.value = mem.content
  editDialogVisible.value = true
}

async function handleSave() {
  if (!editingMemory.value) return
  try {
    await memoryApi.update(editingMemory.value.id, editContent.value)
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    await loadData()
  } catch { /* handled */ }
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('确定删除该记忆？', '提示', { type: 'warning' })
  try {
    await memoryApi.delete(id)
    ElMessage.success('删除成功')
    await loadData()
  } catch { /* handled */ }
}

function formatTime(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div style="display: flex; justify-content: space-between; align-items: center">
        <span>长期记忆库</span>
        <el-button size="small" @click="loadData"><el-icon><Refresh /></el-icon>刷新</el-button>
      </div>
    </template>

    <el-table :data="memories" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column label="分类" width="100">
        <template #default="{ row }">
          <el-tag :type="categoryColors[row.category] as any" size="small">
            {{ categoryLabels[row.category] || row.category }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="key" label="键" min-width="150" show-overflow-tooltip />
      <el-table-column prop="content" label="内容" min-width="300" show-overflow-tooltip />
      <el-table-column label="来源" width="80">
        <template #default="{ row }">
          <el-tag :type="row.source === 'auto' ? 'info' : 'success'" size="small">
            {{ row.source === 'auto' ? '自动' : '手动' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="updated_at" label="更新时间" width="170">
        <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="editDialogVisible" title="编辑记忆" width="500px">
    <el-form label-width="60px">
      <el-form-item label="键">
        <el-input :value="editingMemory?.key" disabled />
      </el-form-item>
      <el-form-item label="内容">
        <el-input v-model="editContent" type="textarea" :rows="6" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

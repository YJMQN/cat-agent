<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { userApi } from '@/api'
import type { User } from '@/types'

const users = ref<User[]>([])
const loading = ref(true)

onMounted(loadData)

async function loadData() {
  loading.value = true
  try {
    const res = await userApi.list()
    users.value = res.data || []
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

async function handleRoleChange(user: User, role: string) {
  try {
    await userApi.updateRole(user.id, role)
    ElMessage.success('角色更新成功')
    await loadData()
  } catch { /* handled */ }
}

function getRoleTag(role: string) {
  const map: Record<string, { type: string; label: string }> = {
    admin: { type: 'danger', label: '管理员' },
    operator: { type: 'warning', label: '运维' },
    user: { type: 'info', label: '普通用户' },
  }
  return map[role] || { type: 'info', label: role }
}

function formatTime(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div style="display: flex; justify-content: space-between; align-items: center">
        <span>用户管理</span>
        <el-button size="small" @click="loadData"><el-icon><Refresh /></el-icon>刷新</el-button>
      </div>
    </template>

    <el-table :data="users" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" label="用户名" min-width="120" />
      <el-table-column label="角色" width="140">
        <template #default="{ row }">
          <el-tag :type="getRoleTag(row.role).type as any" size="small">
            {{ getRoleTag(row.role).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="角色设置" width="180">
        <template #default="{ row }">
          <el-select
            :model-value="row.role"
            size="small"
            @change="(val: string) => handleRoleChange(row, val)"
          >
            <el-option label="管理员" value="admin" />
            <el-option label="运维" value="operator" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="注册时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'
import type { FormInstance, FormRules } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: '',
  password: '',
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 32, message: '长度在3到32个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码不少于6位', trigger: 'blur' },
  ],
}

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const res = await authApi.login({ username: form.username, password: form.password })
    userStore.setUser(res.data.user, res.data.token)
    ElMessage.success('登录成功')
    router.push('/chat')
  } catch {
    // 错误已在拦截器中处理
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <el-icon :size="48" color="#6ea96a"><Cpu /></el-icon>
        <h1>Eino Agent</h1>
        <p>AI Agent 管理系统</p>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" size="large" @submit.prevent="handleLogin">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" :prefix-icon="Lock" show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">
            登 录
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <p>默认账户: admin / admin123</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { User, Lock } from '@element-plus/icons-vue'
export default { components: { User, Lock } }
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at top, #eef8ee 0%, #dff1df 48%, #c8e5c7 100%);
}

.login-card {
  width: 400px;
  padding: 40px;
  background: rgba(255, 255, 255, 0.96);
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(81, 122, 76, 0.18);
  border: 1px solid rgba(110, 169, 106, 0.15);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-header h1 {
  margin: 12px 0 4px;
  font-size: 24px;
  color: #1f3224;
}

.login-header p {
  color: #5f6f5f;
  font-size: 14px;
}

.login-footer {
  text-align: center;
  margin-top: 16px;
}

.login-footer p {
  color: #5f6f5f;
  font-size: 12px;
}
</style>

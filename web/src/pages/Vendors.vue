<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useModelVendorStore } from '@/stores/vendor'

const vendorStore = useModelVendorStore()
const customURL = ref(vendorStore.customURL)
const apiKey = ref(vendorStore.apiKey)
const modelName = ref(vendorStore.model)

const isCustom = computed(() => vendorStore.selectedKey === 'custom')

const modelOptions = computed(() => {
  const vendorKey = vendorStore.selectedKey
  if (vendorKey === 'openrouter') {
    return ['openai/gpt-4o-mini', 'openai/gpt-4o', 'deepseek/deepseek-chat', 'anthropic/claude-3.5-sonnet']
  }
  if (vendorKey === 'deepseek') {
    return ['deepseek-chat', 'deepseek-reasoner']
  }
  if (vendorKey === 'openai') {
    return ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo']
  }
  if (vendorKey === 'modelscope') {
    return ['Qwen/Qwen2.5-7B-Instruct', 'Qwen/Qwen2.5-3B-Instruct', 'Qwen/Qwen2.5-1.5B-Instruct']
  }
  if (vendorKey === 'ollama') {
    return ['qwen2.5', 'llama3', 'deepseek-r1', 'glm4']
  }
  return []
})

watch(
  () => vendorStore.selectedKey,
  (key) => {
    if (key === 'custom') {
      return
    }
    if (!modelName.value || !modelOptions.value.includes(modelName.value)) {
      modelName.value = modelOptions.value[0] || ''
    }
  },
  { immediate: true }
)

function handleSave() {
  vendorStore.setSelected(vendorStore.selectedKey)
  vendorStore.setCustomURL(customURL.value)
  vendorStore.setAPIKey(apiKey.value)
  vendorStore.setModel(modelName.value)
  ElMessage.success('模型厂商配置已保存')
}

function openCurrentLink() {
  const url = vendorStore.currentLink
  if (!url) {
    ElMessage.warning('请先填写自定义链接')
    return
  }

  window.open(url, '_blank', 'noopener,noreferrer')
}

function handleReset() {
  vendorStore.reset()
  customURL.value = ''
  apiKey.value = ''
  modelName.value = vendorStore.model
  ElMessage.success('已恢复默认厂商配置')
}
</script>

<template>
  <div class="vendors-page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>模型厂商</span>
          <el-tag type="success">当前：{{ vendorStore.currentLabel }}</el-tag>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="提示"
        description="这里用于保存当前对话要使用的厂商、API Key 和模型。保存后聊天页会自动带上这些配置。"
      />

      <el-form class="vendor-form" label-width="120px">
        <el-form-item label="厂商选择">
          <el-select v-model="vendorStore.selectedKey" placeholder="请选择模型厂商" style="width: 100%">
            <el-option
              v-for="vendor in vendorStore.vendorOptions"
              :key="vendor.key"
              :label="vendor.name"
              :value="vendor.key"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="模型选择">
          <el-select
            v-model="modelName"
            filterable
            allow-create
            placeholder="选择或输入模型名称"
            style="width: 100%"
          >
            <el-option v-for="item in modelOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>

        <el-form-item label="API Key">
          <el-input
            v-model="apiKey"
            type="password"
            show-password
            placeholder="留空则使用默认配置"
            clearable
          />
        </el-form-item>

        <el-form-item v-if="isCustom" label="自定义链接">
          <el-input
            v-model="customURL"
            placeholder="https://your-model-service.example.com/v1"
            clearable
          />
        </el-form-item>

        <el-form-item label="当前链接">
          <div class="link-box">
            <span>{{ vendorStore.currentLink }}</span>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSave">保存配置</el-button>
          <el-button @click="handleReset">恢复默认</el-button>
          <el-button type="info" plain @click="openCurrentLink" :disabled="!vendorStore.currentLink">打开链接</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.vendors-page {
  max-width: 900px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.vendor-form {
  margin-top: 20px;
}

.link-box {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  background: #f5f7fa;
  color: #606266;
  word-break: break-all;
}
</style>

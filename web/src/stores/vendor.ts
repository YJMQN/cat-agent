import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

interface VendorOption {
  key: string
  name: string
  baseURL: string
  description: string
}

interface VendorConfig {
  selectedKey: string
  customURL: string
  apiKey: string
  model: string
}

const storageKey = 'model-vendor-config'

const defaultVendors: VendorOption[] = [
  {
    key: 'openrouter',
    name: 'OpenRouter',
    baseURL: 'https://openrouter.ai/api/v1',
    description: 'OpenRouter 统一模型入口',
  },
  {
    key: 'deepseek',
    name: 'DeepSeek',
    baseURL: 'https://api.deepseek.com/v1',
    description: 'DeepSeek 开放模型接口',
  },
  {
    key: 'openai',
    name: 'OpenAI',
    baseURL: 'https://api.openai.com/v1',
    description: 'OpenAI 兼容接口',
  },
  {
    key: 'modelscope',
    name: 'ModelScope',
    baseURL: 'https://api-inference.modelscope.cn/v1',
    description: 'ModelScope 开放模型推理接口',
  },
  {
    key: 'ollama',
    name: 'Ollama',
    baseURL: 'http://localhost:11434',
    description: '本地 Ollama 服务',
  },
  {
    key: 'custom',
    name: '自定义链接',
    baseURL: '',
    description: '使用你自己的模型服务地址',
  },
]

const defaultModelByVendor: Record<string, string> = {
  openrouter: 'openai/gpt-4o-mini',
  deepseek: 'deepseek-chat',
  openai: 'gpt-4o-mini',
  modelscope: 'Qwen/Qwen2.5-7B-Instruct',
  ollama: 'qwen2.5',
  custom: '',
}

function loadConfig(): VendorConfig {
  const defaultKey = 'openrouter'

  if (typeof window === 'undefined') {
    return {
      selectedKey: defaultKey,
      customURL: '',
      apiKey: '',
      model: defaultModelByVendor[defaultKey],
    }
  }

  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) {
      return {
        selectedKey: defaultKey,
        customURL: '',
        apiKey: '',
        model: defaultModelByVendor[defaultKey],
      }
    }

    const parsed = JSON.parse(raw) as Partial<VendorConfig>
    const selectedKey = defaultVendors.some(item => item.key === parsed.selectedKey)
      ? parsed.selectedKey || defaultKey
      : defaultKey

    return {
      selectedKey,
      customURL: parsed.customURL || '',
      apiKey: parsed.apiKey || '',
      model: parsed.model || defaultModelByVendor[selectedKey] || '',
    }
  } catch {
    return {
      selectedKey: defaultKey,
      customURL: '',
      apiKey: '',
      model: defaultModelByVendor[defaultKey],
    }
  }
}

export const useModelVendorStore = defineStore('modelVendor', () => {
  const { selectedKey: initialSelectedKey, customURL: initialCustomURL, apiKey: initialApiKey, model: initialModel } = loadConfig()

  const selectedKey = ref<string>(initialSelectedKey)
  const customURL = ref(initialCustomURL)
  const apiKey = ref(initialApiKey)
  const model = ref(initialModel)

  const vendorOptions = computed(() => defaultVendors)

  const currentVendor = computed<VendorOption>(() => {
    if (selectedKey.value === 'custom') {
      return {
        key: 'custom',
        name: '自定义链接',
        baseURL: customURL.value.trim() || 'https://example.com',
        description: '当前使用自定义模型服务链接',
      }
    }

    const vendor = defaultVendors.find(item => item.key === selectedKey.value)
    return vendor || defaultVendors[0]
  })

  const currentLabel = computed(() => currentVendor.value.name)
  const currentLink = computed(() => currentVendor.value.baseURL)
  const currentModel = computed(() => model.value || defaultModelByVendor[selectedKey.value] || '')

  const currentRequestConfig = computed(() => ({
    vendorKey: selectedKey.value,
    baseURL: selectedKey.value === 'custom' ? customURL.value.trim() : currentVendor.value.baseURL,
    apiKey: apiKey.value,
    model: currentModel.value,
  }))

  function persist() {
    if (typeof window === 'undefined') {
      return
    }

    window.localStorage.setItem(
      storageKey,
      JSON.stringify({
        selectedKey: selectedKey.value,
        customURL: customURL.value,
        apiKey: apiKey.value,
        model: model.value,
      })
    )
  }

  function setSelected(key: string) {
    selectedKey.value = key
    persist()
  }

  function setCustomURL(url: string) {
    customURL.value = url
    persist()
  }

  function setAPIKey(value: string) {
    apiKey.value = value
    persist()
  }

  function setModel(value: string) {
    model.value = value
    persist()
  }

  function save() {
    persist()
  }

  function reset() {
    selectedKey.value = 'openrouter'
    customURL.value = ''
    apiKey.value = ''
    model.value = defaultModelByVendor.openrouter
    persist()
  }

  return {
    selectedKey,
    customURL,
    apiKey,
    model,
    vendorOptions,
    currentVendor,
    currentLabel,
    currentLink,
    currentModel,
    currentRequestConfig,
    setSelected,
    setCustomURL,
    setAPIKey,
    setModel,
    save,
    reset,
  }
})

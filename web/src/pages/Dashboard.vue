<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { statsApi } from '@/api'
import type { AdminStats, TokenUsage } from '@/types'

const stats = ref<AdminStats>({
  total_sessions: 0,
  total_messages: 0,
  total_tokens: 0,
  total_users: 0,
  success_rate: 0,
  avg_latency_ms: 0,
  active_sessions: 0,
})
const tokenData = ref<TokenUsage[]>([])
const toolRanking = ref<Array<{ tool_name: string; call_count: number }>>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const [overviewRes, tokenRes, toolRes] = await Promise.all([
      statsApi.overview(),
      statsApi.tokenUsage(7),
      statsApi.toolRanking(),
    ])
    stats.value = overviewRes.data
    tokenData.value = tokenRes.data || []
    toolRanking.value = toolRes.data || []
  } catch {
    // 使用默认值
  } finally {
    loading.value = false
  }
})

function formatNumber(n: number): string {
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'k'
  return n.toString()
}
</script>

<template>
  <div class="dashboard" v-loading="loading">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #e8f5e9"><el-icon :size="28" color="#4caf50"><ChatDotRound /></el-icon></div>
          <div class="stat-info">
            <div class="stat-value">{{ formatNumber(stats.total_sessions) }}</div>
            <div class="stat-label">总会话数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #e3f2fd"><el-icon :size="28" color="#2196f3"><Comment /></el-icon></div>
          <div class="stat-info">
            <div class="stat-value">{{ formatNumber(stats.total_messages) }}</div>
            <div class="stat-label">总消息数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #fff3e0"><el-icon :size="28" color="#ff9800"><Coin /></el-icon></div>
          <div class="stat-info">
            <div class="stat-value">{{ formatNumber(stats.total_tokens) }}</div>
            <div class="stat-label">Token消耗</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #fce4ec"><el-icon :size="28" color="#e91e63"><Connection /></el-icon></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.active_sessions }}</div>
            <div class="stat-label">活跃会话</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <!-- 成功率 & 延迟 -->
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>服务质量</span></template>
          <div class="quality-item">
            <span>请求成功率</span>
            <el-progress :percentage="Math.round(stats.success_rate)" :stroke-width="20" :text-inside="true" />
          </div>
          <div class="quality-item" style="margin-top: 20px">
            <span>平均延迟</span>
            <el-statistic :value="stats.avg_latency_ms" suffix="ms" />
          </div>
        </el-card>
      </el-col>

      <!-- Token用量趋势 -->
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>近7日Token用量</span></template>
          <div v-if="tokenData.length" class="token-chart">
            <div v-for="item in tokenData" :key="item.date" class="token-bar">
              <div class="bar-fill" :style="{ height: Math.max(4, (item.total / Math.max(...tokenData.map(t => t.total))) * 100) + '%' }"></div>
              <span class="bar-label">{{ item.date.slice(5) }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="60" />
        </el-card>
      </el-col>

      <!-- 热门工具 -->
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>热门工具排行</span></template>
          <div v-if="toolRanking.length">
            <div v-for="(item, idx) in toolRanking" :key="idx" class="tool-rank-item">
              <span class="rank-badge" :class="{ top: idx < 3 }">{{ idx + 1 }}</span>
              <span class="tool-name">{{ item.tool_name || '未知' }}</span>
              <span class="tool-count">{{ item.call_count }}次</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="60" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.stat-cards .el-card {
  border-radius: 8px;
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 0;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.quality-item span {
  display: block;
  margin-bottom: 8px;
  color: #606266;
  font-size: 14px;
}

.token-chart {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 160px;
  padding-top: 10px;
}

.token-bar {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
}

.bar-fill {
  width: 100%;
  background: linear-gradient(180deg, #409eff, #79bbff);
  border-radius: 4px 4px 0 0;
  min-height: 4px;
  transition: height 0.3s;
}

.bar-label {
  font-size: 11px;
  color: #909399;
  margin-top: 6px;
}

.tool-rank-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.tool-rank-item:last-child {
  border-bottom: none;
}

.rank-badge {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #e4e7ed;
  color: #606266;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  margin-right: 12px;
  flex-shrink: 0;
}

.rank-badge.top {
  background: linear-gradient(135deg, #f7ba2a, #f56c6c);
  color: #fff;
}

.tool-name {
  flex: 1;
  font-size: 14px;
  color: #303133;
}

.tool-count {
  font-size: 13px;
  color: #909399;
}
</style>

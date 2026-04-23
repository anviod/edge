<template>
  <v-card class="glass-card device-metrics-card" :class="{ 'degraded': isDegraded }">
    <v-card-text class="pa-3">
      <!-- 顶部：设备健康度 -->
      <div class="d-flex align-center mb-3">
        <!-- 健康度指示器 -->
        <div class="health-indicator mr-3">
          <v-progress-circular
            :model-value="healthScore"
            :color="getHealthColor(healthScore)"
            :size="48"
            :width="4"
            class="health-ring"
          >
            <v-icon 
              :color="getHealthColor(healthScore)"
              size="small"
            >
              {{ getHealthIcon(healthScore) }}
            </v-icon>
          </v-progress-circular>
        </div>
        
        <!-- 设备状态 -->
        <div class="flex-grow-1">
          <div class="d-flex align-center mb-1">
            <v-chip 
              size="small" 
              :color="getStateColor(device?.state)"
              variant="flat"
              class="mr-2"
            >
              {{ getStateText(device?.state) }}
            </v-chip>
            <v-chip
              v-if="isDegraded"
              size="small"
              color="warning"
              variant="outlined"
              prepend-icon="mdi-speedometer-slow"
            >
              已降级
            </v-chip>
          </div>
          <div class="text-caption text-grey-darken-1 d-flex align-center">
            <v-icon size="x-small" class="mr-1">mdi-clock-outline</v-icon>
            {{ lastCollectTime }}
          </div>
        </div>

        <!-- 失败计数徽章 -->
        <v-badge
          v-if="metrics?.consecutiveFailures > 0"
          :content="metrics.consecutiveFailures"
          color="error"
          offset-x="-8"
          offset-y="8"
        >
          <v-icon color="error">mdi-alert-circle</v-icon>
        </v-badge>
      </div>

      <!-- 核心采集指标 -->
      <v-row dense class="metrics-row">
        <v-col cols="4">
          <div class="metric-box text-center" :class="{ 'has-issue': metrics?.pointSuccessRate < 0.95 }">
            <div class="text-caption text-grey-darken-1">点位成功率</div>
            <div class="text-h6 font-weight-bold" :class="getPointSuccessColor(metrics?.pointSuccessRate)">
              {{ formatPercent(metrics?.pointSuccessRate) }}
            </div>
          </div>
        </v-col>
        <v-col cols="4">
          <div class="metric-box text-center">
            <div class="text-caption text-grey-darken-1">采集耗时</div>
            <div class="text-h6 font-weight-bold">
              {{ formatDuration(metrics?.avgCollectTime) }}
            </div>
          </div>
        </v-col>
        <v-col cols="4">
          <div class="metric-box text-center" :class="{ 'has-issue': metrics?.nullValueRate > 0.05 }">
            <div class="text-caption text-grey-darken-1">Null值比例</div>
            <div class="text-h6 font-weight-bold" :class="getNullRateColor(metrics?.nullValueRate)">
              {{ formatPercent(metrics?.nullValueRate) }}
            </div>
          </div>
        </v-col>
      </v-row>

      <!-- 扩展详情 -->
      <v-expand-transition>
        <div v-show="showDetails" class="mt-3 pt-3 border-t">
          <v-row dense>
            <v-col cols="6">
              <div class="detail-row">
                <span class="text-caption text-grey-darken-1">调度周期</span>
                <span class="text-body-2">{{ device?.interval || '-' }}</span>
              </div>
            </v-col>
            <v-col cols="6">
              <div class="detail-row">
                <span class="text-caption text-grey-darken-1">健康评分</span>
                <span class="text-body-2 font-weight-bold" :class="`text-${getHealthColor(healthScore)}`">
                  {{ healthScore }}
                </span>
              </div>
            </v-col>
            <v-col cols="6">
              <div class="detail-row">
                <span class="text-caption text-grey-darken-1">异常点位</span>
                <span class="text-body-2" :class="metrics?.abnormalPoints > 0 ? 'text-warning' : ''">
                  {{ metrics?.abnormalPoints || 0 }}
                </span>
              </div>
            </v-col>
            <v-col cols="6">
              <div class="detail-row">
                <span class="text-caption text-grey-darken-1">无效值</span>
                <span class="text-body-2" :class="metrics?.invalidValues > 0 ? 'text-error' : ''">
                  {{ metrics?.invalidValues || 0 }}
                </span>
              </div>
            </v-col>
          </v-row>

          <!-- 连续失败警告 -->
          <v-alert
            v-if="metrics?.consecutiveFailures >= 3"
            type="warning"
            variant="tonal"
            density="compact"
            class="mt-2"
          >
            <div class="d-flex align-center">
              <v-icon class="mr-2">mdi-alert</v-icon>
              <span>连续失败 {{ metrics.consecutiveFailures }} 次，设备已降级</span>
            </div>
          </v-alert>

          <!-- 恢复中提示 -->
          <v-alert
            v-if="device?.recovering"
            type="info"
            variant="tonal"
            density="compact"
            class="mt-2"
          >
            <div class="d-flex align-center">
              <v-progress-circular indeterminate size="16" width="2" class="mr-2"></v-progress-circular>
              <span>设备恢复中...</span>
            </div>
          </v-alert>
        </div>
      </v-expand-transition>
    </v-card-text>
  </v-card>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  device: {
    type: Object,
    default: () => ({})
  },
  metrics: {
    type: Object,
    default: () => ({})
  },
  showDetails: {
    type: Boolean,
    default: false
  }
})

// 是否降级
const isDegraded = computed(() => {
  return props.device?.degraded || props.metrics?.consecutiveFailures >= 3
})

// 健康评分计算
const healthScore = computed(() => {
  if (!props.metrics && !props.device) return 100
  
  let health = 100
  const m = props.metrics || {}
  
  // 连续失败扣分 (每次扣10分)
  if (m.consecutiveFailures) {
    health -= m.consecutiveFailures * 10
  }
  
  // 异常点位比例扣分
  if (m.abnormalPointRate) {
    health -= m.abnormalPointRate * 30
  }
  
  // 超时比例扣分
  if (m.timeoutRate) {
    health -= m.timeoutRate * 30
  }
  
  // Null值比例扣分
  if (m.nullValueRate) {
    health -= m.nullValueRate * 20
  }
  
  return Math.max(0, Math.round(health))
})

// 最后采集时间
const lastCollectTime = computed(() => {
  const ts = props.metrics?.lastCollectTime || props.device?.lastCollectTime
  if (!ts) return '从未采集'
  
  const date = new Date(ts)
  const now = new Date()
  const diff = now - date
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  return date.toLocaleString()
})

// 状态文本和颜色
const getStateText = (state) => {
  switch (state) {
    case 0: return '在线'
    case 1: return '不稳定'
    case 2: return '离线'
    case 3: return '隔离'
    default: return '未知'
  }
}

const getStateColor = (state) => {
  switch (state) {
    case 0: return 'success'
    case 1: return 'warning'
    case 2: return 'error'
    case 3: return 'grey'
    default: return 'grey'
  }
}

// 健康度相关
const getHealthLabel = (score) => {
  if (score >= 90) return 'Healthy'
  if (score >= 70) return 'Warning'
  if (score >= 50) return 'Risk'
  return 'Critical'
}

const getHealthColor = (score) => {
  if (score >= 90) return 'success'
  if (score >= 70) return 'warning'
  if (score >= 50) return 'orange'
  return 'error'
}

const getHealthIcon = (score) => {
  if (score >= 90) return 'mdi-heart-pulse'
  if (score >= 70) return 'mdi-heart-half-full'
  if (score >= 50) return 'mdi-heart-broken'
  return 'mdi-heart-off'
}

// 点位成功率颜色
const getPointSuccessColor = (rate) => {
  if (rate >= 0.98) return 'text-success'
  if (rate >= 0.90) return 'text-warning'
  return 'text-error'
}

// Null值比例颜色
const getNullRateColor = (rate) => {
  if (rate < 0.01) return 'text-success'
  if (rate < 0.05) return 'text-warning'
  return 'text-error'
}

// 格式化
const formatPercent = (val) => {
  if (val === undefined || val === null) return '-'
  return (val * 100).toFixed(1) + '%'
}

const formatDuration = (ms) => {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1) return '<1ms'
  if (ms < 1000) return ms.toFixed(2) + 'ms'
  return (ms / 1000).toFixed(2) + 's'
}
</script>

<style scoped>
.device-metrics-card {
  transition: all 0.3s ease;
  border-left: 3px solid transparent;
}

.device-metrics-card.degraded {
  border-left-color: rgb(var(--v-theme-warning));
  background: rgba(var(--v-theme-warning), 0.05);
}

.health-indicator {
  position: relative;
}

.health-ring {
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
}

.metrics-row {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 8px 0;
}

.metric-box {
  padding: 8px 4px;
  border-radius: 0;
  transition: all 0.2s;
}

.metric-box:hover {
  background: rgba(255, 255, 255, 0.05);
}

.metric-box.has-issue {
  background: rgba(var(--v-theme-error), 0.05);
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 8px;
  border-radius: 0;
}

.detail-row:hover {
  background: rgba(255, 255, 255, 0.05);
}

.border-t {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}
</style>


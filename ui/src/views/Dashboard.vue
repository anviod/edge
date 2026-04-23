<template>
  <div class="dashboard-container">
    <!-- Header with Theme Toggle -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">系统概览</h2>
    </div>

    <!-- System Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">CPU 使用率</div>
        <div class="stat-value" :style="{ color: getCpuColor(system.cpu_usage) }">
          {{ (system.cpu_usage || 0).toFixed(1) }}%
        </div>
        <div class="stat-bar">
          <div class="stat-progress" :style="{ width: (system.cpu_usage || 0) + '%', background: getCpuColor(system.cpu_usage) }"></div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">内存使用</div>
        <div class="stat-value" :style="{ color: getMemoryColor(system.memory_usage) }">
          {{ formatMemory(system.memory_usage) }}
        </div>
        <div class="stat-bar">
          <div class="stat-progress" :style="{ width: getMemoryPercent(system.memory_usage) + '%', background: getMemoryColor(system.memory_usage) }"></div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">协程数量</div>
        <div class="stat-value" style="color: #10b981;">
          {{ system.goroutines || 0 }}
        </div>
        <div class="stat-bar">
          <div class="stat-progress" style="width: 100%; background: #10b981;"></div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">磁盘使用率</div>
        <div class="stat-value" :style="{ color: getDiskColor(system.disk_usage) }">
          {{ (system.disk_usage || 0).toFixed(1) }}%
        </div>
        <div class="stat-bar">
          <div class="stat-progress" :style="{ width: (system.disk_usage || 0) + '%', background: getDiskColor(system.disk_usage) }"></div>
        </div>
      </div>
    </div>

    <!-- Collection Channels Section -->
    <div class="section">
      <div class="section-header">
        <h3 class="section-title">采集通道</h3>
        <div class="section-status">
          <span class="status-badge online">
            <span class="status-dot"></span>
            在线: {{ totalOnlineDevices }}
          </span>
          <span class="status-badge offline">
            <span class="status-dot"></span>
            离线: {{ totalOfflineDevices }}
          </span>
        </div>
      </div>
      
      <div class="channels-grid">
        <div v-for="ch in channels" :key="ch.id" class="channel-card" @click="$router.push(`/channels/${ch.id}/devices`)">
          <div class="channel-header">
            <div class="channel-icon" :class="getProtocolClass(ch.protocol)">
              <icon-link v-if="ch.protocol === 'bacnet-ip'" :size="20" />
              <icon-link v-else-if="ch.protocol === 'modbus-rtu'" :size="20" />
              <icon-link v-else-if="ch.protocol === 'modbus-tcp'" :size="20" />
              <icon-tool v-else-if="ch.protocol === 'opc-ua'" :size="20" />
              <icon-settings v-else-if="ch.protocol === 's7'" :size="20" />
              <icon-link v-else :size="20" />
            </div>
            <div class="channel-info">
              <div class="channel-name">
                {{ ch.name }}
                <span class="quality-score" :class="getQualityClass(ch.qualityScore)">{{ ch.qualityScore || '-' }}</span>
              </div>
              <div class="channel-meta">
                {{ ch.protocol }}
                <span class="divider">|</span>
                <span :class="['status-text', ch.enable ? 'enabled' : 'disabled']">{{ ch.enable ? '启用' : '禁用' }}</span>
              </div>
            </div>
            <icon-arrow-right :size="14" class="arrow-icon" />
          </div>
          
          <div class="channel-stats">
            <div class="stat-item">
              <div class="stat-item-label">设备</div>
              <div class="stat-item-value">{{ ch.device_count || 0 }}</div>
            </div>
            <div class="stat-item">
              <div class="stat-item-label online">在线</div>
              <div class="stat-item-value online">{{ ch.online_count || 0 }}</div>
            </div>
            <div class="stat-item">
              <div class="stat-item-label offline">离线</div>
              <div class="stat-item-value offline">{{ ch.offline_count || 0 }}</div>
            </div>
            <div class="stat-item">
              <div class="stat-item-label">成功率</div>
              <div class="stat-item-value" :class="getSuccessRateClass(ch.successRate)">{{ formatPercent(ch.successRate) }}</div>
            </div>
          </div>
          
          <div class="channel-metrics" v-if="ch.metrics">
            <div class="metrics-header">
              <span class="metrics-label">通信质量</span>
              <span class="metrics-rtt">RTT: {{ formatDuration(ch.metrics.avgRtt) }}</span>
            </div>
            <div class="quality-bar-container">
              <div class="quality-bar" :class="getQualityBarClass(ch.qualityScore)" :style="{ width: (ch.qualityScore || 0) + '%' }"></div>
            </div>
            <div v-if="ch.metrics.reconnectCount > 0" class="reconnect-info">
              <icon-refresh :size="12" />
              重连: {{ ch.metrics.reconnectCount }}
            </div>
          </div>
        </div>
        
        <div v-if="channels.length === 0" class="empty-card">
          <div class="empty-content">
            <icon-apps :size="48" style="margin-bottom: 12px;" />
            <p>暂无采集通道配置</p>
            <button class="btn-primary" @click="$router.push('/channels')">添加通道</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Northbound Section -->
    <div class="section">
      <div class="section-header">
        <h3 class="section-title">北向数据上报</h3>
      </div>
      <div class="northbound-grid">
        <div v-for="nb in northbound" :key="nb.id" class="northbound-card">
          <div class="northbound-header">
            <h4 class="northbound-name">{{ nb.name }}</h4>
            <span class="status-badge" :class="nb.status === 'Running' ? 'online' : (nb.status === 'Disabled' ? 'disabled' : 'offline')">
              {{ nb.status }}
            </span>
          </div>
          <div class="northbound-type">{{ nb.type }}</div>
          <div class="northbound-actions">
            <button class="btn-outline" @click="$router.push('/northbound')">配置</button>
          </div>
        </div>
        <div v-if="northbound.length === 0" class="empty-card">
          <div class="empty-content">
            <p>暂无北向数据上报配置</p>
            <button class="btn-primary" @click="$router.push('/northbound')">配置北向</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Edge Compute Section -->
    <div class="section">
      <div class="section-header">
        <h3 class="section-title">边缘计算状态</h3>
      </div>
      <div class="edge-compute-card" @click="$router.push('/edge-compute/metrics')">
        <div class="edge-stats">
          <div class="edge-stat-item">
            <div class="edge-stat-label">规则数</div>
            <div class="edge-stat-value">{{ edgeRules.rule_count || 0 }}</div>
          </div>
          <div class="edge-stat-item">
            <div class="edge-stat-label">已触发</div>
            <div class="edge-stat-value primary">{{ edgeRules.rules_triggered || 0 }}</div>
          </div>
          <div class="edge-stat-item">
            <div class="edge-stat-label">已执行</div>
            <div class="edge-stat-value success">{{ edgeRules.rules_executed || 0 }}</div>
          </div>
          <div class="edge-stat-item">
            <div class="edge-stat-label">工作池负载</div>
            <div class="edge-stat-bar">
              <div class="edge-progress" :style="{ width: getWorkerPoolPercent() + '%' }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import request from '@/utils/request'
import {
  IconRefresh,
  IconApps, IconLink, IconSettings, IconTool,
  IconArrowRight
} from '@arco-design/web-vue/es/icon'

const router = useRouter()

const system = ref({
    cpu_usage: 0,
    memory_usage: 0,
    disk_usage: 0,
    goroutines: 0
})
const channels = ref([])
const northbound = ref([])
const edgeRules = ref({})

let timer = null

// 计算总在线/离线设备数
const totalOnlineDevices = computed(() => {
  return channels.value.reduce((sum, ch) => sum + (ch.online_count || 0), 0)
})

const totalOfflineDevices = computed(() => {
  return channels.value.reduce((sum, ch) => sum + (ch.offline_count || 0), 0)
})

// 获取颜色
const getCpuColor = (val) => {
  if (val >= 80) return '#ef4444'
  if (val >= 60) return '#f59e0b'
  return '#6366f1'
}

const getMemoryColor = (val) => {
  if (val >= 1024 * 0.8) return '#ef4444'
  if (val >= 1024 * 0.6) return '#f59e0b'
  return '#3b82f6'
}

const getDiskColor = (val) => {
  if (val >= 80) return '#ef4444'
  if (val >= 60) return '#f59e0b'
  return '#f97316'
}

// 格式化
const formatMemory = (mb) => {
  if (!mb) return '0 MB'
  if (mb >= 1024) return (mb / 1024).toFixed(1) + ' GB'
  return Math.round(mb) + ' MB'
}

const getMemoryPercent = (mb) => {
  // 假设总内存为 8GB
  const total = 8 * 1024
  return Math.min(((mb || 0) / total) * 100, 100)
}

const formatPercent = (val) => {
  if (val === undefined || val === null) return '-'
  return (val * 100).toFixed(0) + '%'
}

const formatDuration = (ms) => {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1) return '<1ms'
  if (ms < 1000) return ms.toFixed(2) + 'ms'
  return (ms / 1000).toFixed(2) + 's'
}

const getWorkerPoolPercent = () => {
  const usage = edgeRules.value.worker_pool_usage || 0
  const size = edgeRules.value.worker_pool_size || 1
  return Math.min((usage / size) * 100, 100)
}

// 获取样式类
const getProtocolClass = (protocol) => {
  const classes = {
    'modbus-tcp': 'protocol-tcp',
    'modbus-rtu': 'protocol-rtu',
    'modbus-rtu-over-tcp': 'protocol-tcp',
    'bacnet-ip': 'protocol-bacnet',
    'opc-ua': 'protocol-opc',
    's7': 'protocol-s7',
    'ethernet-ip': 'protocol-ip',
    'mitsubishi-slmp': 'protocol-mitsubishi',
    'omron-fins': 'protocol-omron'
  }
  return classes[protocol] || 'protocol-default'
}

const getQualityClass = (score) => {
  if (score === undefined || score === null || score === 0) return 'quality-none'
  if (score === 100) return 'quality-perfect'
  if (score >= 90) return 'quality-good'
  if (score >= 80) return 'quality-fair'
  return 'quality-poor'
}

const getQualityBarClass = (score) => {
  if (score === undefined || score === null || score === 0) return 'bar-none'
  if (score === 100) return 'bar-perfect'
  if (score >= 90) return 'bar-good'
  if (score >= 80) return 'bar-fair'
  return 'bar-poor'
}

const getSuccessRateClass = (rate) => {
  if (!rate && rate !== 0) return ''
  if (rate >= 0.99) return 'success'
  if (rate >= 0.95) return 'warning'
  return 'error'
}

const fetchData = async () => {
    try {
        const data = await request.get('/api/dashboard/summary')
        system.value = data.system
        
        // 处理通道数据，合并metrics
        channels.value = (data.channels || []).map(ch => {
          // 计算质量评分
          let qualityScore = 100
          if (ch.metrics) {
            const m = ch.metrics
            if (m.successRate !== undefined) qualityScore -= (1 - m.successRate) * 40
            if (m.crcErrorRate !== undefined) qualityScore -= m.crcErrorRate * 20
            if (m.retryRate !== undefined) qualityScore -= m.retryRate * 20
            qualityScore = Math.max(0, Math.round(qualityScore))
          }
          
          return {
            ...ch,
            qualityScore,
            successRate: ch.metrics?.successRate || ch.success_rate || 0
          }
        }).sort((a, b) => a.name.localeCompare(b.name))
        
        northbound.value = data.northbound || []
        edgeRules.value = data.edge_rules || {}
    } catch (e) {
        console.error(e)
    }
}

onMounted(() => {
    fetchData()
    timer = setInterval(fetchData, 2000)
})

onUnmounted(() => {
    if (timer) clearInterval(timer)
})
</script>

<style scoped>
.dashboard-container {
  padding: 24px;
  background: #f1f5f9;
  min-height: calc(100vh - 56px);
}

/* Dashboard Header */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--arco-border, #e2e8f0);
}

.dashboard-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--arco-text-1, #1e293b);
  margin: 0;
}

.dashboard-actions {
  display: flex;
  gap: 8px;
}

.theme-toggle {
  background: transparent;
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 6px;
  cursor: pointer;
  color: var(--arco-text-2, #64748b);
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.theme-toggle:hover {
  background: var(--arco-bg-2, #f1f5f9);
  border-color: var(--arco-border-2, #cbd5e1);
}

.theme-toggle svg {
  width: 18px;
  height: 18px;
}

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 20px;
  transition: all 0.2s ease;
}

.stat-card:hover {
  border-color: var(--arco-primary-6, #3b82f6);
}

.stat-label {
  font-size: 13px;
  color: var(--arco-text-2, #64748b);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--arco-text-1, #1e293b);
  margin-bottom: 12px;
}

.stat-bar {
  height: 4px;
  background: var(--arco-border, #e2e8f0);
  border-radius: 0;
  overflow: hidden;
}

.stat-progress {
  height: 100%;
  border-radius: 0;
  transition: width 0.3s ease;
}

/* Section */
.section {
  margin-bottom: 32px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--arco-border, #e2e8f0);
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--arco-text-1, #1e293b);
  margin: 0;
}

.section-status {
  display: flex;
  gap: 12px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  font-size: 13px;
  font-weight: 500;
  background: var(--arco-bg-2, #ffffff);
}

.status-badge.online {
  border-color: #16a34a;
  color: #16a34a;
  background: rgba(22, 163, 74, 0.05);
}

.status-badge.offline {
  border-color: #dc2626;
  color: #dc2626;
  background: rgba(220, 38, 38, 0.05);
}

.status-badge.disabled {
  border-color: var(--arco-border, #e2e8f0);
  color: var(--arco-text-3, #94a3b8);
  background: var(--arco-bg-2, #ffffff);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

/* Channels Grid */
.channels-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.channel-card {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.channel-card:hover {
  border-color: var(--arco-primary-6, #3b82f6);
}

.channel-header {
  display: flex;
  align-items: flex-start;
  margin-bottom: 16px;
}

.channel-icon {
  width: 40px;
  height: 40px;
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  flex-shrink: 0;
  background: var(--arco-bg-2, #ffffff);
}

.channel-icon.protocol-bacnet {
  border-color: #2563eb;
  color: #2563eb;
  background: rgba(37, 99, 235, 0.05);
}

.channel-icon.protocol-tcp {
  border-color: #16a34a;
  color: #16a34a;
  background: rgba(22, 163, 74, 0.05);
}

.channel-icon.protocol-rtu {
  border-color: #d97706;
  color: #d97706;
  background: rgba(217, 119, 6, 0.05);
}

.channel-icon.protocol-opc {
  border-color: #9333ea;
  color: #9333ea;
  background: rgba(147, 51, 234, 0.05);
}

.channel-icon.protocol-s7 {
  border-color: #db2777;
  color: #db2777;
  background: rgba(219, 39, 119, 0.05);
}

.channel-icon.protocol-default {
  border-color: var(--arco-border, #e2e8f0);
  color: var(--arco-text-3, #94a3b8);
  background: var(--arco-bg-2, #ffffff);
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--arco-text-1, #1e293b);
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.quality-score {
  font-size: 11px;
  padding: 2px 6px;
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  font-weight: 500;
  background: var(--arco-bg-2, #ffffff);
}

.quality-score.quality-perfect {
  border-color: #2563eb;
  color: #2563eb;
  background: rgba(37, 99, 235, 0.05);
}

.quality-score.quality-good {
  border-color: #16a34a;
  color: #16a34a;
  background: rgba(22, 163, 74, 0.05);
}

.quality-score.quality-fair {
  border-color: #d97706;
  color: #d97706;
  background: rgba(217, 119, 6, 0.05);
}

.quality-score.quality-poor {
  border-color: #dc2626;
  color: #dc2626;
  background: rgba(220, 38, 38, 0.05);
}

.quality-score.quality-none {
  border-color: var(--arco-border, #e2e8f0);
  color: var(--arco-text-3, #94a3b8);
  background: var(--arco-bg-2, #ffffff);
}

.channel-meta {
  font-size: 12px;
  color: var(--arco-text-2, #64748b);
}

.divider {
  margin: 0 6px;
  color: var(--arco-border, #e2e8f0);
}

.status-text.enabled {
  color: #16a34a;
}

.status-text.disabled {
  color: var(--arco-text-3, #94a3b8);
}

.arrow-icon {
  color: var(--arco-text-3, #94a3b8);
  flex-shrink: 0;
  margin-left: 8px;
}

.channel-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  padding: 16px 0;
  border-top: 1px solid var(--arco-border, #e2e8f0);
  border-bottom: 1px solid var(--arco-border, #e2e8f0);
  margin-bottom: 16px;
}

.stat-item {
  text-align: center;
}

.stat-item-label {
  font-size: 12px;
  color: var(--arco-text-2, #64748b);
  margin-bottom: 4px;
}

.stat-item-label.online {
  color: #16a34a;
}

.stat-item-label.offline {
  color: #dc2626;
}

.stat-item-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--arco-text-1, #1e293b);
}

.stat-item-value.online {
  color: #16a34a;
}

.stat-item-value.offline {
  color: #dc2626;
}

.stat-item-value.success {
  color: #16a34a;
}

.stat-item-value.warning {
  color: #d97706;
}

.stat-item-value.error {
  color: #dc2626;
}

/* Channel Metrics */
.channel-metrics {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 12px;
}

.metrics-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.metrics-label {
  font-size: 12px;
  color: var(--arco-text-2, #64748b);
  display: flex;
  align-items: center;
  gap: 4px;
}

.metrics-rtt {
  font-size: 12px;
  color: var(--arco-text-3, #94a3b8);
}

.quality-bar-container {
  height: 4px;
  background: var(--arco-border, #e2e8f0);
  border-radius: 0;
  overflow: hidden;
}

.quality-bar {
  height: 100%;
  border-radius: 0;
  transition: width 0.3s ease;
}

.quality-bar.bar-perfect {
  background: #3b82f6;
}

.quality-bar.bar-good {
  background: #22c55e;
}

.quality-bar.bar-fair {
  background: #f59e0b;
}

.quality-bar.bar-poor {
  background: #ef4444;
}

.quality-bar.bar-none {
  background: var(--arco-border, #e2e8f0);
}

.reconnect-info {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  font-size: 12px;
  color: #3b82f6;
}

/* Empty Card */
.empty-card {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 40px;
  grid-column: 1 / -1;
  text-align: center;
}

.empty-content {
  text-align: center;
  color: var(--arco-text-2, #64748b);
}

.empty-content p {
  margin: 0 0 16px 0;
}

/* Buttons */
.btn-primary {
  background: #3b82f6;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 0;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #2563eb;
}

.btn-outline {
  background: transparent;
  color: #3b82f6;
  border: 1px solid #3b82f6;
  padding: 6px 14px;
  border-radius: 0;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-outline:hover {
  background: rgba(59, 130, 246, 0.05);
}

/* Northbound */
.northbound-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.northbound-card {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 20px;
}

.northbound-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.northbound-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--arco-text-1, #1e293b);
  margin: 0;
}

.northbound-type {
  font-size: 13px;
  color: var(--arco-text-2, #64748b);
  margin-bottom: 16px;
}

.northbound-actions {
  display: flex;
  justify-content: flex-end;
}

/* Edge Compute */
.edge-compute-card {
  background: var(--arco-bg-2, #ffffff);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 0;
  padding: 24px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.edge-compute-card:hover {
  border-color: var(--arco-primary-6, #3b82f6);
}

.edge-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.edge-stat-item {
  text-align: center;
}

.edge-stat-label {
  font-size: 13px;
  color: var(--arco-text-2, #64748b);
  margin-bottom: 8px;
}

.edge-stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--arco-text-1, #1e293b);
}

.edge-stat-value.primary {
  color: #3b82f6;
}

.edge-stat-value.success {
  color: #22c55e;
}

.edge-stat-bar {
  height: 4px;
  background: var(--arco-border, #e2e8f0);
  border-radius: 0;
  overflow: hidden;
  margin-top: 8px;
}

.edge-progress {
  height: 100%;
  background: #f59e0b;
  border-radius: 0;
  transition: width 0.3s ease;
}

/* Responsive */
@media (max-width: 1200px) {
  .channels-grid,
  .northbound-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .channels-grid,
  .northbound-grid {
    grid-template-columns: 1fr;
  }
  
  .edge-stats {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>


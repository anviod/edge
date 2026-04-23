<template>
  <div class="edge-compute-metrics-container">
    <a-row :gutter="[16, 16]" class="metrics-row">
      <a-col :span="12" :md="4" class="metrics-col">
        <a-card class="metrics-card">
          <div class="metrics-label">规则总数</div>
          <div class="metrics-value">{{ metrics.rule_count }}</div>
        </a-card>
      </a-col>
      <a-col :span="12" :md="4" class="metrics-col">
        <a-card class="metrics-card">
          <div class="metrics-label">共享源数量</div>
          <div class="metrics-value">{{ metrics.shared_source_count }}</div>
        </a-card>
      </a-col>
      <a-col :span="12" :md="4" class="metrics-col">
        <a-card class="metrics-card">
          <div class="metrics-label">缓存大小</div>
          <div class="metrics-value">{{ metrics.cache_size }}</div>
        </a-card>
      </a-col>
      <a-col :span="12" :md="6" class="metrics-col">
        <a-card class="metrics-card">
          <div class="metrics-label">执行/触发/丢弃</div>
          <div class="metrics-composite">
            <span class="metrics-composite-value">{{ metrics.rules_executed }}</span>
            <span class="metrics-composite-separator">/ {{ metrics.rules_triggered }} / {{ metrics.rules_dropped }}</span>
          </div>
        </a-card>
      </a-col>
      <a-col :span="12" :md="6" class="metrics-col">
        <a-card class="metrics-card">
            <div class="metrics-label">并发执行 (使用/总量)</div>
            <div class="metrics-composite mb-2">
                {{ metrics.worker_pool_usage }} / {{ metrics.worker_pool_size }}
            </div>
            <a-progress
                :percentage="workerUsagePercent"
                :stroke-width="4"
                color="#111827"
                track-color="#e5e7eb"
            />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[16, 16]" class="mt-4">
      <a-col :span="24">
        <a-card class="metrics-card">
          <template #title>
            <span class="card-title">共享源详情</span>
          </template>
            <a-table
                :columns="columns"
                :data="sharedSources"
                size="small"
                :bordered="false"
            >
                <template #subscribers="{ record }">
                  <div class="subscribers-line">
                    <span
                      v-for="sub in record.subscribers"
                      :key="sub"
                      class="sub-item"
                    >
                      {{ sub }}
                    </span>
                  </div>
                </template>
            </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { Card, Row, Col, Progress, Table, Tag } from '@arco-design/web-vue'
import request from '@/utils/request'
const metrics = ref({
    worker_pool_size: 0,
    worker_pool_usage: 0,
    rule_count: 0,
    shared_source_count: 0,
    cache_size: 0,
    rules_triggered: 0,
    rules_executed: 0,
    rules_dropped: 0
})

const sharedSources = ref([])
const columns = [
    { title: '数据源 ID', dataIndex: 'source_id' },
    { title: '订阅数量', dataIndex: 'subscriber_count' },
    { title: '订阅规则', dataIndex: 'subscribers', slotName: 'subscribers' }
]

const workerUsagePercent = computed(() => {
    if (metrics.value.worker_pool_size === 0) return 0;
    return (metrics.value.worker_pool_usage / metrics.value.worker_pool_size) * 100;
})

let timer = null

const fetchMetrics = async () => {
    try {
        const data = await request.get('/api/edge/metrics')
        if (data) {
            metrics.value = data
        }
    } catch (e) {
        console.error(e)
    }
}

const fetchSharedSources = async () => {
    try {
        const data = await request.get('/api/edge/shared-sources')
        if (data) {
            sharedSources.value = data || []
        }
    } catch (e) {
        console.error(e)
    }
}

const fetchData = () => {
    fetchMetrics()
    fetchSharedSources()
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
.edge-compute-metrics-container {
  padding: 24px;
  min-height: calc(100vh - 56px);
  background: #f1f5f9;
}

.metrics-row {
  display: flex;
  flex-wrap: wrap;
  align-items: stretch;
}

.metrics-col {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.metrics-card {
  border: 1px solid #e5e7eb;
  border-radius: 0;
  padding: 16px;
  height: 100%;
  background: #ffffff;
  position: relative;
  display: flex;
  flex-direction: column;
}

.metrics-card::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 1px;
  background: #0f172a;
  opacity: 0.05;
}

.metrics-label {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 8px;
}

.metrics-value {
  font-size: 26px;
  font-weight: 600;
  color: #111827;
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.5px;
}

.metrics-composite {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
  display: flex;
  align-items: baseline;
}

.metrics-composite-separator {
  font-size: 14px;
  color: #6b7280;
  margin-left: 8px;
}

.card-title {
  font-size: 12px;
  font-weight: 600;
  color: #374151;
  letter-spacing: 0.5px;
}

.subscribers-line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sub-item {
  font-size: 11px;
  padding: 2px 6px;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  color: #374151;
  background: #fafafa;
}

.arco-card:hover {
  box-shadow: none !important;
  border-color: #111827;
}

:deep(.arco-table-th) {
  background: #fafafa;
  border-bottom: 1px solid #e5e7eb;
  font-size: 11px;
  color: #6b7280;
  font-weight: 500;
}

:deep(.arco-table-td) {
  font-size: 12px;
  border-bottom: 1px solid #f1f3f5;
}

:deep(.arco-table-tr:hover .arco-table-td) {
  background: #f9fafb;
}

/* 确保卡片内部结构也能拉伸 */
:deep(.arco-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

:deep(.arco-card-body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 16px;
}
</style>


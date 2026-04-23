<template>
  <a-card v-for="item in items" :key="item.id" class="northbound-card" hoverable>
    <template #title>
      <div class="card-title-row">
        <span class="protocol-tag">HTTP</span>
        <span class="card-name">{{ item.name || item.id }}</span>
      </div>
    </template>
    <template #extra>
      <a-space size="small">
        <a-tooltip content="配置">
          <a-button type="text" size="mini" @click="$emit('settings', item)">
            <template #icon><icon-settings :size="14" /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip content="运行监控">
          <a-button type="text" size="mini" @click="$emit('stats', item)">
            <template #icon><icon-bar-chart :size="14" /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip content="删除">
          <a-button type="text" size="mini" status="danger" @click="$emit('delete', 'http', item.id)">
            <template #icon><icon-delete :size="14" /></template>
          </a-button>
        </a-tooltip>
      </a-space>
    </template>

    <div class="card-info-list">
      <div class="info-row">
        <span class="info-label"><icon-cloud :size="14" /> 服务器地址</span>
        <span class="info-value text-ellipsis">
          {{ item.url }}
          <a-button type="text" size="mini" @click="copyToClipboard(item.url)" style="margin-left: 4px">
            <template #icon><icon-copy :size="12" /></template>
          </a-button>
        </span>
      </div>
      <div class="info-row">
        <span class="info-label"><icon-send :size="14" /> 请求方法</span>
        <span class="info-value">{{ item.method || 'POST' }}</span>
      </div>
      <div class="info-row" v-if="item.data_endpoint">
        <span class="info-label"><icon-code-block :size="14" /> 数据端点</span>
        <span class="info-value text-ellipsis">{{ item.data_endpoint }}</span>
      </div>
    </div>

    <template #actions>
      <a-tag :color="item.enable ? 'green' : 'gray'" size="small">
        {{ item.enable ? '启用' : '禁用' }}
      </a-tag>
    </template>
  </a-card>
</template>

<script setup>
import { IconSettings, IconDelete, IconCloud, IconSend, IconCodeBlock, IconCopy, IconBarChart } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'

defineProps({
  items: { type: Array, default: () => [] }
})

defineEmits(['settings', 'delete', 'stats'])

const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    showMessage('已复制到剪贴板', 'success')
  }).catch(() => {
    showMessage('复制失败', 'error')
  })
}
</script>

<style scoped>
.northbound-card {
  border: 1px solid #e5e7eb;
  border-radius: 0;
  margin-bottom: 16px;
  width: 100%;
  display: flex;
  flex-direction: column;
}

.card-info-list {
  display: flex;
  flex-direction: column;
  gap: 0;
  flex: 1;
  padding: 8px 0 0;
}

.northbound-card:hover {
  border-color: #0f172a;
}

.card-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.protocol-tag {
  background: #0ea5e9;
  color: #fff;
  font-family: monospace;
  font-size: 10px;
  padding: 0 4px;
  border-radius: 0;
  line-height: 20px;
}

.card-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-row {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  font-size: 13px;
  border-bottom: 1px dashed #cbd5e1;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  color: #6b7280;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.info-value {
  color: #334155;
  max-width: 60%;
  text-align: right;
}

.text-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>


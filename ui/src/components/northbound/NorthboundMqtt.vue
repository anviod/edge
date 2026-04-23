<template>
  <a-card v-for="item in items" :key="item.id" class="northbound-card" hoverable>
    <template #title>
      <div class="card-title-row">
        <span class="protocol-tag">MQTT</span>
        <span class="card-name">{{ item.name || item.id }}</span>
      </div>
    </template>
    <template #extra>
      <a-space size="small">
        <a-tooltip content="帮助文档">
          <a-button type="text" size="mini" @click="$emit('help', item)">
            <template #icon><icon-question-circle :size="14" /></template>
          </a-button>
        </a-tooltip>
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
          <a-button type="text" size="mini" status="danger" @click="$emit('delete', 'mqtt', item.id)">
            <template #icon><icon-delete :size="14" /></template>
          </a-button>
        </a-tooltip>
      </a-space>
    </template>

    <div class="card-info-list">
      <div class="info-row">
        <span class="info-label"><icon-cloud :size="14" /> Broker地址</span>
        <span class="info-value text-ellipsis">
          {{ item.broker }}
          <a-button type="text" size="mini" @click="copyToClipboard(item.broker)" style="margin-left: 4px">
            <template #icon><icon-copy :size="12" /></template>
          </a-button>
        </span>
      </div>
      <div class="info-row">
        <span class="info-label"><icon-idcard :size="14" /> Client ID</span>
        <span class="info-value text-ellipsis">{{ item.client_id }}</span>
      </div>
      <div class="info-row">
        <span class="info-label"><icon-send :size="14" /> 发布主题</span>
        <span class="info-value text-ellipsis">{{ item.topic }}</span>
      </div>
      <div class="info-row" v-if="item.subscribe_topic">
        <span class="info-label"><icon-download :size="14" /> 订阅主题</span>
        <span class="info-value text-ellipsis">{{ item.subscribe_topic }}</span>
      </div>
    </div>

    <template #actions>
      <a-tag v-if="!item.enable" color="gray" size="small">未启用</a-tag>
      <a-tag v-else-if="connectionStatus && connectionStatus[item.id] === 1" color="green" size="small">已连接</a-tag>
      <a-tag v-else-if="connectionStatus && connectionStatus[item.id] === 2" color="orangered" size="small">重连中</a-tag>
      <a-tag v-else color="red" size="small">连接断开</a-tag>
    </template>
  </a-card>
</template>

<script setup>
import { IconQuestionCircle, IconSettings, IconBarChart, IconDelete, IconCloud, IconIdcard, IconSend, IconDownload, IconCopy } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'

defineProps({
  items: { type: Array, default: () => [] },
  connectionStatus: { type: Object, default: () => ({}) }
})

defineEmits(['help', 'settings', 'stats', 'delete'])

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

.card-info-list {
  padding: 8px 0 0;
}

.info-row {
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


<template>
  <a-modal v-model:visible="visible" title="MQTT 接入文档" :width="900" :footer="false" unmount-on-close>
    <a-tabs v-model:active-key="activeTab" type="line">
      <a-tab-pane key="reporting" title="数据上报">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">数据上报 (Data Reporting)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">设备采集的数据将按照以下格式自动上报到 Broker。</p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #0ea5e9">
          <div style="font-size: 12px; font-weight: 600; color: #0ea5e9; margin-bottom: 8px">Topic (发布主题)</div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ topic }}</span>
            <a-button type="text" size="mini" @click="copyToClipboard(topic)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">Payload 格式 (JSON)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{
  "timestamp": 1678888888888,
  "node": "device_name",
  "group": "channel_name",
  "values": {
    "point_name": 123.45
  },
  "errors": {},
  "metas": {}
}</pre>
      </a-tab-pane>

      <a-tab-pane key="control" title="设备控制">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">设备控制 (Device Control)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">向设备写入数据，支持多点位同时写入。</p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #165dff">
          <div style="font-size: 12px; font-weight: 600; color: #165dff; margin-bottom: 8px">Topic (订阅主题 - 发送请求)</div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ subscribe_topic || '未配置' }}</span>
            <a-button type="text" size="mini" @click="copyToClipboard(subscribe_topic)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">请求 Payload (JSON)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; margin-bottom: 16px; overflow-x: auto">{
  "uuid": "req_123456",
  "group": "channel_name",
  "node": "device_name",
  "values": {
    "point_name": 1
  }
}</pre>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #00b42a">
          <div style="font-size: 12px; font-weight: 600; color: #00b42a; margin-bottom: 8px">Topic (响应主题 - 接收结果)</div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ write_response_topic || (subscribe_topic ? subscribe_topic + '/resp' : '未配置') }}</span>
            <a-button type="text" size="mini" @click="copyToClipboard(write_response_topic || (subscribe_topic ? subscribe_topic + '/resp' : ''))">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">响应 Payload (JSON)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{
  "uuid": "req_123456",
  "success": true,
  "message": "error msg"
}</pre>
      </a-tab-pane>

      <a-tab-pane key="status" title="在线状态">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">上下线状态 (Status)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">网关/通道以及南向设备的连接状态变更时发布。</p>
        </div>

        <a-alert type="info" style="margin-bottom: 16px">
          支持变量替换: <code>{status}</code>, <code>{timestamp}</code>, <code>{device_id}</code>, <code>{device_name}</code>。
        </a-alert>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #ff7d00">
          <div style="font-size: 12px; font-weight: 600; color: #ff7d00; margin-bottom: 8px">Topic (状态主题)</div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ status_topic || topic + '/status' }}</span>
            <a-button type="text" size="mini" @click="copyToClipboard(status_topic || topic + '/status')">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">Payload (上线 - Online)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; margin-bottom: 16px; overflow-x: auto">{{ online_payload || '{\n  "status": "online",\n  "timestamp": 1678888888888\n}' }}</pre>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">Payload (离线/遗嘱 - Offline/LWT)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{{ offline_payload || '{\n  "status": "offline",\n  "timestamp": 1678888888888\n}' }}</pre>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup>
import { ref, computed } from 'vue'
import { IconCopy } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'

const props = defineProps({
  visible: { type: Boolean, default: false },
  topic: { type: String, default: '' },
  subscribe_topic: { type: String, default: '' },
  write_response_topic: { type: String, default: '' },
  status_topic: { type: String, default: '' },
  online_payload: { type: String, default: '' },
  offline_payload: { type: String, default: '' }
})

const emit = defineEmits(['update:visible'])

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})
const activeTab = ref('reporting')

const copyToClipboard = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    showMessage('已复制到剪贴板', 'success')
  }).catch(() => {
    showMessage('复制失败', 'error')
  })
}
</script>


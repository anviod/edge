<template>
  <a-modal v-model:visible="visible" title="Sparkplug B 接入文档" :width="900" :footer="false" unmount-on-close>
    <a-tabs v-model:active-key="activeTab" type="line">
      <a-tab-pane key="overview" title="协议概述">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">Sparkplug B 协议</h4>
          <p style="color: #6b7280; margin: 0 0 16px">
            Sparkplug B 是基于 MQTT 的工业物联网协议，提供标准化的话题结构和消息格式。
          </p>
        </div>

        <a-alert type="info" style="margin-bottom: 16px">
          Sparkplug B 使用 Protocol Buffers 编码，确保数据传输的高效性和互操作性。
        </a-alert>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">Topic 结构</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; margin-bottom: 16px; overflow-x: auto">namespace/group_id/message_type/edge_node_id/[device_id]</pre>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">消息类型</div>
        <a-table :data="messageTypes" :bordered="false" size="small" :pagination="false">
          <template #columns>
            <a-table-column title="类型" data-index="type" :width="120" />
            <a-table-column title="说明" data-index="description" />
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="publish" title="数据上报">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">数据上报 (Data Publishing)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">
            设备数据通过 NBIRTH/DBIRTH 消息上报出生声明，通过 NDATA/DDATA 消息上报数据更新。
          </p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #0ea5e9">
          <div style="font-size: 12px; font-weight: 600; color: #0ea5e9; margin-bottom: 8px">
            NBIRTH Topic (节点出生)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/NBIRTH/{{ nodeId || 'EdgeX' }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/NBIRTH/${nodeId || 'EdgeX'}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #165dff">
          <div style="font-size: 12px; font-weight: 600; color: #165dff; margin-bottom: 8px">
            DBIRTH Topic (设备出生)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/DBIRTH/{{ nodeId || 'EdgeX' }}/{{ deviceId }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/DBIRTH/${nodeId || 'EdgeX'}/${deviceId}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">Payload 结构 (Protobuf)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{
  "timestamp": 1678888888888,
  "metrics": [
    {
      "name": "temperature",
      "value": 25.5,
      "type": "Float"
    }
  ],
  "seq": 0
}</pre>
      </a-tab-pane>

      <a-tab-pane key="command" title="设备控制">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">设备控制 (Device Control)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">
            通过 NCMD/DCMD 消息向设备发送控制指令。
          </p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #ff7d00">
          <div style="font-size: 12px; font-weight: 600; color: #ff7d00; margin-bottom: 8px">
            NCMD Topic (节点命令)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/NCMD/{{ nodeId || 'EdgeX' }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/NCMD/${nodeId || 'EdgeX'}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #00b42a">
          <div style="font-size: 12px; font-weight: 600; color: #00b42a; margin-bottom: 8px">
            DCMD Topic (设备命令)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/DCMD/{{ nodeId || 'EdgeX' }}/{{ deviceId }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/DCMD/${nodeId || 'EdgeX'}/${deviceId}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">命令 Payload (Protobuf)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{
  "timestamp": 1678888888888,
  "metrics": [
    {
      "name": "setpoint",
      "value": 100,
      "type": "Int32"
    }
  ],
  "seq": 1
}</pre>
      </a-tab-pane>

      <a-tab-pane key="status" title="状态管理">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">状态管理 (State Management)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">
            Sparkplug B 通过 NDEATH/DDEATH 消息管理设备离线状态，支持遗嘱消息 (LWT)。
          </p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #f53f3f">
          <div style="font-size: 12px; font-weight: 600; color: #f53f3f; margin-bottom: 8px">
            NDEATH Topic (节点死亡)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/NDEATH/{{ nodeId || 'EdgeX' }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/NDEATH/${nodeId || 'EdgeX'}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #722ed1">
          <div style="font-size: 12px; font-weight: 600; color: #722ed1; margin-bottom: 8px">
            DDEATH Topic (设备死亡)
          </div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              spBv1.0/{{ groupId || '采集通' }}/DDEATH/{{ nodeId || 'EdgeX' }}/{{ deviceId }}
            </span>
            <a-button type="text" size="mini" @click="copyToClipboard(`spBv1.0/${groupId || '采集通'}/DDEATH/${nodeId || 'EdgeX'}/${deviceId}`)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
        </a-card>

        <a-alert type="warning" style="margin-bottom: 16px">
          NDEATH 消息应配置为 MQTT 遗嘱消息 (LWT)，确保异常断开时能自动发布。
        </a-alert>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">状态 Payload (Protobuf)</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; overflow-x: auto">{
  "timestamp": 1678888888888,
  "seq": 255
}</pre>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { IconCopy } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'

const props = defineProps({
  visible: { type: Boolean, default: false },
  groupId: { type: String, default: '采集通道' },
  nodeId: { type: String, default: '' },
  deviceId: { type: String, default: 'device_1' }
})

const emit = defineEmits(['update:visible'])

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})
const activeTab = ref('overview')

const messageTypes = ref([
  { type: 'NBIRTH', description: '节点出生声明，包含节点所有指标初始值' },
  { type: 'DBIRTH', description: '设备出生声明，包含设备所有指标初始值' },
  { type: 'NDATA', description: '节点数据更新，仅包含变化的指标' },
  { type: 'DDATA', description: '设备数据更新，仅包含变化的指标' },
  { type: 'NCMD', description: '节点命令，向节点发送控制指令' },
  { type: 'DCMD', description: '设备命令，向设备发送控制指令' },
  { type: 'NDEATH', description: '节点死亡通知，通常作为遗嘱消息' },
  { type: 'DDEATH', description: '设备死亡通知' }
])

watch(() => props.modelValue, (val) => { visible.value = val })
watch(visible, (val) => { emit('update:modelValue', val) })

const copyToClipboard = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    showMessage('已复制到剪贴板', 'success')
  }).catch(() => {
    showMessage('复制失败', 'error')
  })
}
</script>


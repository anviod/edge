<template>
  <a-modal
    v-model:visible="modelValue"
    :title="isEdit ? '编辑 edgeOS(MQTT) 配置' : '新增 edgeOS(MQTT) 配置'"
    :width="1000"
    @ok="saveSettings"
    :ok-loading="loading"
    unmount-on-close
    :footer="true"
    :mask-closable="false"
    class="industrial-modal"
  >
    <a-tabs v-model:active-key="activeTab" type="line" class="industrial-tabs">
      <a-tab-pane key="basic">
        <template #title><icon-settings /> 基本配置</template>
        <a-form :model="form" layout="horizontal" :label-col-props="{ span: 5 }" :wrapper-col-props="{ span: 19 }" class="industrial-form">
          <a-form-item label="通道名称" required>
            <a-input v-model="form.name" placeholder="例如: edgeOS MQTT 生产通道" />
          </a-form-item>

          <a-form-item label="启用状态">
            <a-switch v-model="form.enable" type="round" />
          </a-form-item>

          <a-form-item label="Broker 地址" required>
            <a-input v-model="form.broker" placeholder="tcp://127.0.0.1:1883" class="mono-text" />
          </a-form-item>

          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="Client ID" required>
                <a-input v-model="form.client_id" placeholder="edgex-node-001" class="mono-text" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="节点 ID" required>
                <a-input v-model="form.node_id" placeholder="edgex-node-001" class="mono-text" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-form-item label="QoS 级别">
            <a-select v-model="form.qos">
              <a-option :value="0">0 - 最多一次</a-option>
              <a-option :value="1">1 - 至少一次</a-option>
              <a-option :value="2">2 - 恰好一次</a-option>
            </a-select>
          </a-form-item>

          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="用户名">
                <a-input v-model="form.username" placeholder="可选" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="密码">
                <a-input-password v-model="form.password" placeholder="可选" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="心跳间隔">
                <a-input-number v-model="form.keep_alive" :min="10" :max="3600" placeholder="60" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="心跳周期">
                <a-input v-model="form.heartbeat_interval" placeholder="30s" class="mono-text" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="保留消息">
                <a-switch v-model="form.retain" type="round" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="自动重连">
                <a-switch v-model="form.auto_reconnect" type="round" />
              </a-form-item>
            </a-col>
          </a-row>
        </a-form>
      </a-tab-pane>

      <a-tab-pane key="device-mapping">
        <template #title><icon-scan /> 设备映射</template>
        <div class="table-header">
          <a-button type="primary" size="small" @click="autoFillDevices">
            <template #icon><icon-check /></template>一键填充所有设备
          </a-button>
        </div>
        <div class="table-container">
          <a-table 
            :columns="deviceColumns" 
            :data="deviceTableData" 
            size="small" 
            :bordered="{ wrapper: true, cell: true }" 
            :pagination="false"
            class="industrial-table-inline"
          >
            <template #state="{ record }">
              <a-tag v-if="record.state === 0" color="green" size="small" class="proto-tag-mini">在线</a-tag>
              <a-tag v-else-if="record.state === 1" color="orangered" size="small" class="proto-tag-mini">不稳定</a-tag>
              <a-tag v-else color="red" size="small" class="proto-tag-mini">离线</a-tag>
            </template>
            <template #enable="{ record }">
              <a-switch v-model="record._enable" size="small" @change="updateDeviceEnable(record)" />
            </template>
            <template #strategy="{ record }">
              <a-select v-model="record._strategy" size="small" :disabled="!record._enable" @change="updateDeviceStrategy(record)" class="mono-text">
                <a-option value="periodic">周期上报</a-option>
                <a-option value="change">变化上报</a-option>
              </a-select>
            </template>
            <template #interval="{ record }">
              <a-input v-if="record._strategy === 'periodic'" v-model="record._interval" size="small" :disabled="!record._enable" class="mono-text" @change="updateDeviceInterval(record)" />
            </template>
          </a-table>
        </div>
      </a-tab-pane>
    </a-tabs>

    <template #footer>
      <div class="industrial-modal-footer">
        <a-button @click="modelValue = false" class="btn-secondary">取消</a-button>
        <a-button type="primary" :loading="loading" @click="saveSettings" class="btn-primary">
          <template #icon><icon-plus /></template>保存通道配置
        </a-button>
      </div>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { IconPlus, IconSettings, IconScan, IconCheck } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'
import request from '@/utils/request'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  config: { type: Object, default: null },
  allDevices: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue', 'saved'])

const modelValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
const loading = ref(false)
const form = ref({})
const deviceTableData = ref([])
const activeTab = ref('basic')

const isEdit = computed(() => props.config && props.config.id)

const deviceColumns = [
  { title: '设备名称', dataIndex: 'name', width: 200 },
  { title: '通道', dataIndex: 'channelName', width: 120 },
  { title: '在线状态', slotName: 'state', width: 80, align: 'center' },
  { title: '启用', slotName: 'enable', width: 60, align: 'center' },
  { title: '策略', slotName: 'strategy', width: 120 },
  { title: '上报周期', slotName: 'interval', width: 100 }
]

watch(() => props.modelValue, (val) => {
  if (val) {
    if (props.config) {
      form.value = JSON.parse(JSON.stringify(props.config))
    } else {
      form.value = {
        id: 'edgeos-mqtt_' + Date.now(),
        enable: true,
        name: 'New edgeOS MQTT Channel',
        broker: 'tcp://127.0.0.1:1883',
        client_id: 'edgex-node-001',
        node_id: 'edgex-node-001',
        username: '',
        password: '',
        qos: 1,
        retain: false,
        keep_alive: 60,
        auto_reconnect: true,
        heartbeat_interval: '30s',
        devices: {}
      }
    }
    if (!form.value.devices) form.value.devices = {}

    buildDeviceTable()
  }
})

const buildDeviceTable = () => {
  deviceTableData.value = props.allDevices.map(dev => {
    const current = form.value.devices[dev.id]
    let _enable = false, _strategy = 'periodic', _interval = '10s'
    if (current === undefined || current === null) {
      _enable = false
    } else if (typeof current === 'boolean') {
      _enable = current
    } else if (typeof current === 'object') {
      _enable = !!current.enable
      _strategy = current.strategy || 'periodic'
      _interval = current.interval || '10s'
    }
    return { ...dev, _enable, _strategy, _interval }
  })
}

const updateDeviceEnable = (record) => {
  if (!form.value.devices[record.id]) {
    form.value.devices[record.id] = { enable: record._enable, strategy: 'periodic', interval: '10s' }
  } else if (typeof form.value.devices[record.id] === 'boolean') {
    form.value.devices[record.id] = { enable: record._enable, strategy: 'periodic', interval: '10s' }
  } else {
    form.value.devices[record.id].enable = record._enable
  }
}

const updateDeviceStrategy = (record) => {
  if (!form.value.devices[record.id] || typeof form.value.devices[record.id] === 'boolean') {
    form.value.devices[record.id] = { enable: record._enable, strategy: record._strategy, interval: record._interval }
  } else {
    form.value.devices[record.id].strategy = record._strategy
  }
}

const updateDeviceInterval = (record) => {
  if (!form.value.devices[record.id] || typeof form.value.devices[record.id] === 'boolean') {
    form.value.devices[record.id] = { enable: record._enable, strategy: record._strategy, interval: record._interval }
  } else {
    form.value.devices[record.id].interval = record._interval
  }
}

const autoFillDevices = () => {
  deviceTableData.value.forEach(record => {
    record._enable = true
    record._strategy = 'periodic'
    record._interval = '10s'
    updateDeviceEnable(record)
  })
  showMessage('已一键填充所有设备配置', 'success')
}

const saveSettings = async () => {
  loading.value = true
  try {
    const dataToSave = { ...form.value }
    await request.post('/api/northbound/edgeos-mqtt', dataToSave)
    showMessage('edgeOS(MQTT) 配置已保存', 'success')
    modelValue.value = false
    emit('saved')
  } catch (e) {
    showMessage('保存失败: ' + e.message, 'error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 弹窗整体风格优化 */
::deep(.arco-modal) {
  border-radius: 0;
}

::deep(.arco-modal-header) {
  border-bottom: 1px solid #e5e7eb;
  height: 48px;
}

/* 标签页对齐 */
.industrial-tabs :deep(.arco-tabs-nav-tab) {
  padding: 0 16px;
}

.industrial-tabs :deep(.arco-tabs-content) {
  padding: 24px;
}

/* 极简表单样式 */
.industrial-form :deep(.arco-form-item-label) {
  font-weight: 500;
  color: #475569;
  font-size: 13px;
  white-space: nowrap;
}

.industrial-form :deep(.arco-input),
.industrial-form :deep(.arco-select-view),
.industrial-form :deep(.arco-input-number) {
  border-radius: 0;
  background-color: #fcfcfc;
  border-color: #e5e7eb;
}

/* 技术数据专用字体 */
.mono-text {
  font-family: 'JetBrains Mono', 'Fira Code', monospace !important;
  font-size: 12px;
}

/* 表格融合规范 */
.table-container {
  border: 1px solid #e5e7eb;
  overflow-x: auto;
}

.table-header {
  display: flex;
  justify-content: flex-end;
  padding: 0 0 12px 0;
}

.industrial-table-inline {
  width: 100%;
  table-layout: fixed;
}

.industrial-table-inline :deep(.arco-table) {
  width: 100%;
  border-collapse: collapse;
}

.industrial-table-inline :deep(.arco-table-th) {
  background-color: #f8fafc;
  font-weight: bold;
  height: 34px;
  border-bottom: 1px solid #e5e7eb;
  border-right: 1px solid #e5e7eb;
  text-align: center;
  vertical-align: middle;
  padding: 0 8px;
}

.industrial-table-inline :deep(.arco-table-th:last-child) {
  border-right: none;
}

.industrial-table-inline :deep(.arco-table-td) {
  height: 34px;
  border-bottom: 1px solid #e5e7eb;
  border-right: 1px solid #e5e7eb;
  text-align: center;
  vertical-align: middle;
  padding: 0 8px;
}

.industrial-table-inline :deep(.arco-table-td:last-child) {
  border-right: none;
}

.industrial-table-inline :deep(.arco-table-td:first-child),
.industrial-table-inline :deep(.arco-table-th:first-child) {
  text-align: left;
  padding-left: 12px;
}

/* 底部操作区 */
.industrial-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 0 0;
}

.btn-primary {
  background-color: #0f172a !important;
  border-radius: 0 !important;
}

.btn-secondary {
  border-radius: 0 !important;
  border-color: #cbd5e1;
}
</style>

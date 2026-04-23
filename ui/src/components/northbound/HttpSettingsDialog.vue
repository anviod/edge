<template>
  <a-modal 
    v-model:visible="visible" 
    title="HTTP 导出通道配置" 
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
            <a-input v-model="form.name" placeholder="例如: 云端生产环境 HTTP" />
          </a-form-item>
          
          <a-form-item label="启用状态">
            <a-switch v-model="form.enable" type="round" />
          </a-form-item>

          <a-divider orientation="left">服务器配置</a-divider>

          <a-form-item label="服务器地址" required>
            <a-input v-model="form.url" placeholder="http://localhost:8080" class="mono-text" />
          </a-form-item>
          
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="请求方法">
                <a-select v-model="form.method">
                  <a-option value="POST">POST</a-option>
                  <a-option value="PUT">PUT</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="数据端点">
                <a-input v-model="form.data_endpoint" placeholder="/api/data" class="mono-text" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-form-item label="设备事件端点">
            <a-input v-model="form.device_event_endpoint" placeholder="/api/events" class="mono-text" />
          </a-form-item>

          <a-divider orientation="left">认证配置</a-divider>
          <a-form-item label="认证方式">
            <a-select v-model="form.auth_type">
              <a-option value="None">无认证</a-option>
              <a-option value="Basic">Basic Auth</a-option>
              <a-option value="Bearer">Bearer Token</a-option>
              <a-option value="APIKey">API Key</a-option>
            </a-select>
          </a-form-item>
          <a-row :gutter="16" v-if="form.auth_type === 'Basic'">
            <a-col :span="12">
              <a-form-item label="用户名">
                <a-input v-model="form.username" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="密码">
                <a-input-password v-model="form.password" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="Token" v-if="form.auth_type === 'Bearer'">
            <a-input-password v-model="form.token" />
          </a-form-item>
          <template v-if="form.auth_type === 'APIKey'">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="Key Name">
                  <a-input v-model="form.api_key_name" placeholder="X-API-Key" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="Key Value">
                  <a-input-password v-model="form.api_key_value" />
                </a-form-item>
              </a-col>
            </a-row>
          </template>

          <a-divider orientation="left">缓存配置</a-divider>
          <a-form-item label="启用缓存">
            <a-switch v-model="form.cache.enable" />
          </a-form-item>
          <a-row :gutter="16" v-if="form.cache.enable">
            <a-col :span="12">
              <a-form-item label="最大缓存数">
                <a-input-number v-model="form.cache.max_count" :min="1" :max="100000" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="刷新间隔">
                <a-input v-model="form.cache.flush_interval" placeholder="1m" class="mono-text" />
              </a-form-item>
            </a-col>
          </a-row>
        </a-form>
      </a-tab-pane>

      <a-tab-pane key="device-strategy">
        <template #title><icon-scan /> 上报策略</template>
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
        <a-button @click="visible = false" class="btn-secondary">取消</a-button>
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
  visible: { type: Boolean, default: false },
  config: { type: Object, default: null },
  allDevices: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:visible', 'saved'])

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})
const loading = ref(false)
const form = ref({})
const activeTab = ref('basic')

const deviceColumns = [
  { title: '设备名称', dataIndex: 'name', width: 200 },
  { title: '通道', dataIndex: 'channelName', width: 120 },
  { title: '在线状态', slotName: 'state', width: 80, align: 'center' },
  { title: '启用', slotName: 'enable', width: 60, align: 'center' },
  { title: '策略', slotName: 'strategy', width: 120 },
  { title: '上报周期', slotName: 'interval', width: 100 }
]

const deviceTableData = ref([])

watch(() => props.visible, (val) => {
  if (val) {
    if (props.config) {
      form.value = JSON.parse(JSON.stringify(props.config))
    } else {
      form.value = {
        id: 'http_' + Date.now(),
        enable: true,
        name: 'New HTTP',
        url: 'http://localhost:8080',
        method: 'POST',
        headers: {},
        auth_type: 'None',
        username: '',
        password: '',
        token: '',
        api_key_name: '',
        api_key_value: '',
        data_endpoint: '/api/data',
        device_event_endpoint: '/api/events',
        cache: { enable: true, max_count: 1000, flush_interval: '1m' },
        devices: {}
      }
    }
    if (!form.value.cache) form.value.cache = { enable: true, max_count: 1000, flush_interval: '1m' }
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
    const payload = JSON.parse(JSON.stringify(form.value))
    if (payload.devices && typeof payload.devices === 'object') {
      for (const k of Object.keys(payload.devices)) {
        const v = payload.devices[k]
        if (v && typeof v === 'object') {
          payload.devices[k] = !!v.enable
        } else {
          payload.devices[k] = !!v
        }
      }
    }
    await request.post('/api/northbound/http', payload)
    showMessage('HTTP 配置已保存', 'success')
    visible.value = false
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
:deep(.arco-modal) {
  border-radius: 0;
}

:deep(.arco-modal-header) {
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
.industrial-form :deep(.arco-textarea),
.industrial-form :deep(.arco-select-view),
.industrial-form :deep(.arco-input-number) {
  border-radius: 0; /* 直角设计 */
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

/* 协议标签 */
.proto-tag-mini {
  font-family: monospace;
  font-size: 10px;
  border-radius: 0;
  padding: 0 4px;
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

/* 消除 Arco Divider 默认外边距 */
:deep(.arco-divider-horizontal) {
  margin: 16px 0;
  border-bottom-style: dashed;
}
</style>

